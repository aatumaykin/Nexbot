package docker

import (
	"sync"
	"time"
)

// RateLimiter — простой counter + window (MVP)
type RateLimiter struct {
	mu     sync.Mutex
	count  int
	limit  int
	window time.Duration
	start  time.Time
}

func NewRateLimiter(maxPerMinute int) *RateLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 60
	}
	return &RateLimiter{
		limit:  maxPerMinute,
		window: time.Minute,
		start:  time.Now(),
	}
}

func (r *RateLimiter) Allow() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.start) >= r.window {
		r.count = 0
		r.start = now
	}

	if r.count < r.limit {
		r.count++
		return true, 0
	}

	waitTime := r.window - now.Sub(r.start)
	return false, waitTime
}

func (r *RateLimiter) MaxPerMinute() int {
	return r.limit
}
