package docker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestContainer_TryIncrementPending(t *testing.T) {
	c := &Container{
		pending:    make(map[string]*pendingEntry),
		maxPending: 3,
	}
	for i := 0; i < 3; i++ {
		if !c.tryIncrementPending() {
			t.Errorf("should succeed at %d", i)
		}
	}
	if c.tryIncrementPending() {
		t.Error("should fail after limit")
	}
}

func TestContainer_TryIncrementPending_Concurrent(t *testing.T) {
	c := &Container{
		pending:    make(map[string]*pendingEntry),
		maxPending: 100,
	}
	var wg sync.WaitGroup
	var success atomic.Int64
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.tryIncrementPending() {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 100 {
		t.Errorf("expected 100 successes, got %d", success.Load())
	}
}

func TestContainer_Close(t *testing.T) {
	c := &Container{
		pending: make(map[string]*pendingEntry),
	}
	err := c.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubagentError(t *testing.T) {
	err := &SubagentError{
		Code:       ErrCodeDraining,
		Message:    "pool is draining",
		Retry:      true,
		RetryAfter: 5,
	}

	expected := "[DRAINING] pool is draining"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDockerError(t *testing.T) {
	err := &DockerError{
		Op:      "create",
		Err:     nil,
		Message: "failed to create container",
	}

	expected := "docker create: failed to create container: <nil>"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{RetryAfter: 30 * time.Second}
	expected := "rate limit exceeded, retry after 30s"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCircuitOpenError(t *testing.T) {
	err := &CircuitOpenError{RetryAfter: 10 * time.Second}
	expected := "circuit breaker open, retry after 10s"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
