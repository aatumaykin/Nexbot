package docker

import (
	"testing"
	"time"
)

func TestRateLimiter_AllowWithinLimit(t *testing.T) {
	rl := NewRateLimiter(60)
	for i := 0; i < 5; i++ {
		allowed, _ := rl.Allow()
		if !allowed {
			t.Errorf("should allow request %d", i)
		}
	}
}

func TestRateLimiter_BlockWhenExhausted(t *testing.T) {
	rl := NewRateLimiter(5)
	for i := 0; i < 5; i++ {
		rl.Allow()
	}
	allowed, wait := rl.Allow()
	if allowed {
		t.Error("should not allow when exhausted")
	}
	if wait <= 0 {
		t.Error("should return positive wait time")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(1)
	rl.Allow() // Use up the limit

	// Should be blocked
	allowed, _ := rl.Allow()
	if allowed {
		t.Error("should be blocked")
	}

	// Wait for window to reset
	time.Sleep(61 * time.Second)
}

func TestRateLimiter_MaxPerMinute(t *testing.T) {
	rl := NewRateLimiter(100)
	if rl.MaxPerMinute() != 100 {
		t.Errorf("expected MaxPerMinute=100, got %d", rl.MaxPerMinute())
	}
}

func TestRateLimiter_DefaultValue(t *testing.T) {
	rl := NewRateLimiter(0)
	if rl.MaxPerMinute() != 60 {
		t.Errorf("expected default MaxPerMinute=60, got %d", rl.MaxPerMinute())
	}
}
