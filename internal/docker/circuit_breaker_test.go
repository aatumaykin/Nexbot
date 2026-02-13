package docker

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second, nil)
	allowed, _ := cb.Allow()
	if !allowed {
		t.Error("should allow in closed state")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, nil)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Error("should be open after threshold failures")
	}
	allowed, _ := cb.Allow()
	if allowed {
		t.Error("should not allow when open")
	}
}

func TestCircuitBreaker_ConcurrentFailures(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second, nil)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}
	wg.Wait()
	if cb.State() != CircuitOpen {
		t.Error("should be open after concurrent failures")
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond, nil)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Error("should be open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open and allow one request
	allowed, _ := cb.Allow()
	if !allowed {
		t.Error("should allow in half-open state after timeout")
	}
}

func TestCircuitBreaker_SuccessResets(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Second, nil)

	// Record some failures but not enough to open
	cb.RecordFailure()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Error("success should reset to closed state")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Second, nil)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Error("reset should return to closed state")
	}
}
