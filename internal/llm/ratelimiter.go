package llm

import (
	"sync"
	"time"
)

// RateLimitExceededError возвращается, когда превышен лимит запросов
type RateLimitExceededError struct {
	RetryAfter time.Duration
}

func (e *RateLimitExceededError) Error() string {
	return "rate limit exceeded"
}

// TokenBucketRateLimiter реализует алгоритм token bucket для rate limiting
type TokenBucketRateLimiter struct {
	capacity     int           // Максимальное количество токенов
	tokens       int           // Текущее количество токенов
	refillRate   time.Duration // Интервал пополнения одного токена
	refillAmount int           // Количество токенов при пополнении
	lastRefill   time.Time     // Время последнего пополнения
	mu           sync.Mutex
	metrics      *RateLimitMetrics
}

// RateLimitMetrics хранит метрики rate limiting
type RateLimitMetrics struct {
	TotalRequests    int64
	AllowedRequests  int64
	RejectedRequests int64
}

// NewTokenBucketRateLimiter создает новый rate limiter
// capacity: максимальное количество токенов
// refillInterval: интервал пополнения токенов (например, time.Second для 1 токена/сек)
// refillAmount: количество токенов, добавляемых за каждый интервал
func NewTokenBucketRateLimiter(capacity int, refillInterval time.Duration, refillAmount int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		capacity:     capacity,
		tokens:       capacity,
		refillRate:   refillInterval,
		refillAmount: refillAmount,
		lastRefill:   time.Now(),
		metrics:      &RateLimitMetrics{},
	}
}

// TryAcquire пытается получить токен. Возвращает true если токен доступен.
// Если токенов нет, возвращает false и время ожидания до следующего пополнения.
func (r *TokenBucketRateLimiter) TryAcquire() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.TotalRequests++

	// Пополнение токенов на основе прошедшего времени
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	if elapsed >= r.refillRate {
		// Вычисляем сколько интервалов прошло
		intervals := int(elapsed / r.refillRate)

		// Ограничиваем количество токенов capacity
		tokensToAdd := intervals * r.refillAmount
		if r.tokens+tokensToAdd > r.capacity {
			r.tokens = r.capacity
		} else {
			r.tokens += tokensToAdd
		}

		// Обновляем время последнего пополнения
		// (сохраняем остаток времени для точности)
		r.lastRefill = now.Add(-elapsed % r.refillRate)
	}

	if r.tokens > 0 {
		r.tokens--
		r.metrics.AllowedRequests++
		return true, 0
	}

	// Вычисляем время до следующего пополнения
	timeToNextRefill := r.refillRate - (now.Sub(r.lastRefill) % r.refillRate)
	r.metrics.RejectedRequests++

	return false, timeToNextRefill
}

// Acquire блокирует до получения токена
func (r *TokenBucketRateLimiter) Acquire() {
	for {
		allowed, waitTime := r.TryAcquire()
		if allowed {
			return
		}
		time.Sleep(waitTime)
	}
}

// GetMetrics возвращает текущие метрики
func (r *TokenBucketRateLimiter) GetMetrics() RateLimitMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()

	return RateLimitMetrics{
		TotalRequests:    r.metrics.TotalRequests,
		AllowedRequests:  r.metrics.AllowedRequests,
		RejectedRequests: r.metrics.RejectedRequests,
	}
}

// Reset сбрасывает лимитер в начальное состояние
func (r *TokenBucketRateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens = r.capacity
	r.lastRefill = time.Now()
	r.metrics = &RateLimitMetrics{}
}

// GetAvailableTokens возвращает текущее количество доступных токенов
func (r *TokenBucketRateLimiter) GetAvailableTokens() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.tokens
}
