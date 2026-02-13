package docker

import (
	"sync/atomic"
	"time"
)

type CircuitState int32

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker struct {
	state            atomic.Int32
	failures         atomic.Int32
	lastFail         atomic.Int64
	halfOpenAttempts atomic.Int32
	threshold        int32
	timeout          time.Duration
	metrics          *PoolMetrics
}

func NewCircuitBreaker(threshold int, timeout time.Duration, metrics *PoolMetrics) *CircuitBreaker {
	if threshold == 0 {
		threshold = 5
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &CircuitBreaker{
		threshold: int32(threshold),
		timeout:   timeout,
		metrics:   metrics,
	}
}

func (cb *CircuitBreaker) Allow() (bool, int64) {
	token := time.Now().UnixNano()

	for {
		state := CircuitState(cb.state.Load())

		switch state {
		case CircuitClosed:
			return true, token

		case CircuitOpen:
			lastFailNano := cb.lastFail.Load()
			lastFail := time.Unix(0, lastFailNano)
			if time.Since(lastFail) <= cb.timeout {
				return false, 0
			}
			if !cb.state.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen)) {
				continue
			}
			cb.halfOpenAttempts.Store(0)
			return true, token

		case CircuitHalfOpen:
			if cb.halfOpenAttempts.CompareAndSwap(0, 1) {
				return true, token
			}
			return false, 0
		}
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.failures.Store(0)
	cb.halfOpenAttempts.Store(0)
	cb.state.Store(int32(CircuitClosed))
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.failures.Add(1)
	cb.lastFail.Store(time.Now().UnixNano())

	state := CircuitState(cb.state.Load())
	if state == CircuitHalfOpen {
		cb.state.Store(int32(CircuitOpen))
	} else if cb.failures.Load() >= cb.threshold {
		if cb.state.CompareAndSwap(int32(CircuitClosed), int32(CircuitOpen)) {
			if cb.metrics != nil {
				cb.metrics.CircuitTrips.Add(1)
			}
		}
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

func (cb *CircuitBreaker) Reset() {
	cb.failures.Store(0)
	cb.halfOpenAttempts.Store(0)
	cb.state.Store(int32(CircuitClosed))
}
