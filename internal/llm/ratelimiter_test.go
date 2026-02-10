package llm

import (
	"testing"
	"time"
)

func TestTokenBucketRateLimiter_TryAcquire(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, time.Second, 1)

	// Test initial capacity
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.TryAcquire()
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// Test exceeding capacity
	allowed, _ := limiter.TryAcquire()
	if allowed {
		t.Error("Expected request to be rejected after capacity exceeded")
	}
}

func TestTokenBucketRateLimiter_TokenReplenishment(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(1, 100*time.Millisecond, 1)

	// Consume the only token
	allowed, _ := limiter.TryAcquire()
	if !allowed {
		t.Error("Expected first request to be allowed")
	}

	// Wait for token replenishment
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, _ = limiter.TryAcquire()
	if !allowed {
		t.Error("Expected request to be allowed after token replenishment")
	}
}

func TestTokenBucketRateLimiter_WaitTime(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(1, 100*time.Millisecond, 1)

	// Consume the only token
	allowed, _ := limiter.TryAcquire()
	if !allowed {
		t.Error("Expected first request to be allowed")
	}

	// Get wait time
	allowed, waitTime := limiter.TryAcquire()
	if allowed {
		t.Error("Expected request to be rejected")
	}
	if waitTime <= 0 {
		t.Errorf("Expected positive wait time, got %v", waitTime)
	}
}

func TestTokenBucketRateLimiter_Metrics(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(2, time.Second, 1)

	// Make some requests
	limiter.TryAcquire()
	limiter.TryAcquire()
	limiter.TryAcquire() // Should be rejected

	metrics := limiter.GetMetrics()
	if metrics.TotalRequests != 3 {
		t.Errorf("Expected TotalRequests=3, got %d", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 2 {
		t.Errorf("Expected AllowedRequests=2, got %d", metrics.AllowedRequests)
	}
	if metrics.RejectedRequests != 1 {
		t.Errorf("Expected RejectedRequests=1, got %d", metrics.RejectedRequests)
	}
}

func TestTokenBucketRateLimiter_Reset(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, time.Second, 1)

	// Consume all tokens
	for i := 0; i < 10; i++ {
		limiter.TryAcquire()
	}

	// Reset
	limiter.Reset()

	// Should have full capacity again
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.TryAcquire()
		if !allowed {
			t.Errorf("Expected request %d to be allowed after reset", i+1)
		}
	}
}

func TestTokenBucketRateLimiter_GetAvailableTokens(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, time.Second, 1)

	// Initial capacity
	if tokens := limiter.GetAvailableTokens(); tokens != 10 {
		t.Errorf("Expected 10 available tokens, got %d", tokens)
	}

	// Consume some tokens
	limiter.TryAcquire()
	limiter.TryAcquire()

	// Should have 8 tokens left
	if tokens := limiter.GetAvailableTokens(); tokens != 8 {
		t.Errorf("Expected 8 available tokens, got %d", tokens)
	}
}

func TestRateLimitExceededError(t *testing.T) {
	err := &RateLimitExceededError{RetryAfter: 5 * time.Second}
	if err.Error() != "rate limit exceeded" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
	if err.RetryAfter != 5*time.Second {
		t.Errorf("Unexpected RetryAfter: %v", err.RetryAfter)
	}
}
