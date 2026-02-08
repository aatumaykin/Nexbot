// Package retry provides retry mechanism for LLM calls with exponential backoff.
package retry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	defaultMaxAttempts  = 3
	defaultInitialDelay = 1 * time.Second
	defaultMaxDelay     = 10 * time.Second
)

// Config represents retry configuration.
type Config struct {
	MaxAttempts    int           // Maximum number of retry attempts (default: 3)
	InitialBackoff time.Duration // Initial backoff duration (default: 1s)
	MaxBackoff     time.Duration // Maximum backoff duration (default: 10s)
}

// DoWithRetry executes the given function with retry logic.
// It returns the result of the function or the last error if all attempts fail.
// Context cancellation is checked between attempts.
func DoWithRetry(ctx context.Context, fn func() (string, error), cfg Config) (string, error) {
	// Apply defaults
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultMaxAttempts
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = defaultInitialDelay
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaultMaxDelay
	}

	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		slog.Debug("retry attempt", "attempt", attempt+1, "max_attempts", cfg.MaxAttempts)

		result, err := fn()
		if err == nil {
			slog.Debug("retry success", "attempt", attempt+1)
			return result, nil
		}

		lastErr = err

		if !IsRetryable(err) {
			slog.Debug("non-retryable error", "error", err)
			return "", err
		}

		slog.Debug("retryable error", "attempt", attempt+1, "error", err)

		if attempt == cfg.MaxAttempts-1 {
			slog.Debug("max attempts reached, giving up")
			break
		}

		select {
		case <-ctx.Done():
			slog.Debug("context cancelled during retry")
			return "", ctx.Err()
		default:
		}

		backoff := calculateBackoff(attempt, cfg.InitialBackoff, cfg.MaxBackoff)
		slog.Debug("waiting before next attempt", "backoff", backoff)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			slog.Debug("context cancelled during backoff")
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("all %d attempts failed: %w", cfg.MaxAttempts, lastErr)
}

// IsRetryable checks if an error is retryable based on its message.
// Returns true for timeout, network, rate limit, and temporary errors.
// Returns false for authentication, authorization, not found, and context cancellation errors.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errLower := strings.ToLower(err.Error())

	// Non-retryable errors - return immediately
	nonRetryablePatterns := []string{
		"401",              // Unauthorized
		"403",              // Forbidden
		"400",              // Bad Request
		"404",              // Not Found
		"context canceled", // Explicit cancellation
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errLower, pattern) {
			return false
		}
	}

	// Retryable errors
	retryablePatterns := []string{
		"context deadline exceeded",
		"deadline exceeded",
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"temporary",
		"eof",
		"429", // Too Many Requests
		"too many requests",
		"rate limit",
		"5", // 5xx server errors (500-599)
		"connection",
		"network",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errLower, pattern) {
			return true
		}
	}

	// Unknown error - not retryable by default
	return false
}

// calculateBackoff calculates the backoff duration for a given attempt.
// Uses exponential backoff: 2^attempt * initial
// Capped at maxBackoff if the result exceeds it.
func calculateBackoff(attempt int, initial, max time.Duration) time.Duration {
	// Calculate exponential backoff: 2^attempt * initial
	backoff := time.Duration(1<<uint(attempt)) * initial

	// Cap at maxBackoff
	if backoff > max {
		return max
	}

	return backoff
}
