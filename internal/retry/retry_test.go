package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRetryable_TimeoutErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "context deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "timeout error",
			err:  errors.New("request timeout"),
			want: true,
		},
		{
			name: "deadline exceeded in message",
			err:  errors.New("operation deadline exceeded after 5s"),
			want: true,
		},
		{
			name: "mixed case timeout",
			err:  errors.New("Connection Timeout"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable_RateLimit(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "429 status code",
			err:  errors.New("HTTP 429 Too Many Requests"),
			want: true,
		},
		{
			name: "too many requests text",
			err:  errors.New("too many requests, please retry later"),
			want: true,
		},
		{
			name: "mixed case rate limit",
			err:  errors.New("429 Rate Limit Exceeded"),
			want: true,
		},
		{
			name: "rate limit without code",
			err:  errors.New("Rate limit exceeded"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable_NonRetryableErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "401 unauthorized",
			err:  errors.New("HTTP 401 Unauthorized"),
			want: false,
		},
		{
			name: "403 forbidden",
			err:  errors.New("HTTP 403 Forbidden"),
			want: false,
		},
		{
			name: "400 bad request",
			err:  errors.New("HTTP 400 Bad Request"),
			want: false,
		},
		{
			name: "404 not found",
			err:  errors.New("HTTP 404 Not Found"),
			want: false,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	fn := func() (string, error) {
		callCount++
		return "success", nil
	}

	cfg := Config{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
	}

	result, err := DoWithRetry(ctx, fn, cfg)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("DoWithRetry() result = %v, want 'success'", result)
	}
	if callCount != 1 {
		t.Errorf("DoWithRetry() called %d times, want 1", callCount)
	}
}

func TestDoWithRetry_SuccessAfterRetry(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	fn := func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("timeout")
		}
		return "success", nil
	}

	cfg := Config{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}

	result, err := DoWithRetry(ctx, fn, cfg)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("DoWithRetry() result = %v, want 'success'", result)
	}
	if callCount != 3 {
		t.Errorf("DoWithRetry() called %d times, want 3", callCount)
	}
}

func TestDoWithRetry_AllFailures(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("connection refused")

	fn := func() (string, error) {
		return "", expectedErr
	}

	cfg := Config{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}

	result, err := DoWithRetry(ctx, fn, cfg)
	if err == nil {
		t.Fatal("DoWithRetry() error = nil, want error")
	}
	if result != "" {
		t.Errorf("DoWithRetry() result = %v, want empty string", result)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("DoWithRetry() error = %v, want to wrap %v", err, expectedErr)
	}
}

func TestDoWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	expectedErr := errors.New("HTTP 401 Unauthorized")

	fn := func() (string, error) {
		callCount++
		return "", expectedErr
	}

	cfg := Config{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
	}

	result, err := DoWithRetry(ctx, fn, cfg)
	if err == nil {
		t.Fatal("DoWithRetry() error = nil, want error")
	}
	if result != "" {
		t.Errorf("DoWithRetry() result = %v, want empty string", result)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("DoWithRetry() error = %v, want %v", err, expectedErr)
	}
	if callCount != 1 {
		t.Errorf("DoWithRetry() called %d times, want 1 (non-retryable should stop immediately)", callCount)
	}
}

func TestCalculateBackoff_Values(t *testing.T) {
	initial := 1 * time.Second
	max := 10 * time.Second

	tests := []struct {
		name     string
		attempt  int
		initial  time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{
			name:     "attempt 0",
			attempt:  0,
			initial:  initial,
			max:      max,
			expected: 1 * time.Second, // 2^0 * 1s = 1s
		},
		{
			name:     "attempt 1",
			attempt:  1,
			initial:  initial,
			max:      max,
			expected: 2 * time.Second, // 2^1 * 1s = 2s
		},
		{
			name:     "attempt 2",
			attempt:  2,
			initial:  initial,
			max:      max,
			expected: 4 * time.Second, // 2^2 * 1s = 4s
		},
		{
			name:     "attempt 3",
			attempt:  3,
			initial:  initial,
			max:      max,
			expected: 8 * time.Second, // 2^3 * 1s = 8s (not capped)
		},
		{
			name:     "attempt 4",
			attempt:  4,
			initial:  initial,
			max:      max,
			expected: 10 * time.Second, // 2^4 * 1s = 16s, capped at 10s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoff(tt.attempt, tt.initial, tt.max)
			if got != tt.expected {
				t.Errorf("calculateBackoff() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	t.Run("cancel before any attempts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		callCount := 0
		fn := func() (string, error) {
			callCount++
			return "", errors.New("timeout")
		}

		cfg := Config{
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     500 * time.Millisecond,
		}

		result, err := DoWithRetry(ctx, fn, cfg)
		if err != context.Canceled {
			t.Errorf("DoWithRetry() error = %v, want context.Canceled", err)
		}
		if result != "" {
			t.Errorf("DoWithRetry() result = %v, want empty string", result)
		}
		if callCount != 1 {
			t.Errorf("DoWithRetry() called %d times, want 1", callCount)
		}
	})

	t.Run("cancel during backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		callCount := 0
		fn := func() (string, error) {
			callCount++
			if callCount == 1 {
				// Cancel after first attempt
				go func() {
					time.Sleep(5 * time.Millisecond)
					cancel()
				}()
			}
			return "", errors.New("timeout")
		}

		cfg := Config{
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     500 * time.Millisecond,
		}

		result, err := DoWithRetry(ctx, fn, cfg)
		if err != context.Canceled {
			t.Errorf("DoWithRetry() error = %v, want context.Canceled", err)
		}
		if result != "" {
			t.Errorf("DoWithRetry() result = %v, want empty string", result)
		}
		if callCount != 1 {
			t.Errorf("DoWithRetry() called %d times, want 1", callCount)
		}
	})
}

func TestDoWithRetry_BackoffWithMax(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	backoffs := []time.Duration{}

	fn := func() (string, error) {
		callCount++
		if callCount < 4 {
			return "", errors.New("timeout")
		}
		return "success", nil
	}

	cfg := Config{
		MaxAttempts:    4,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     15 * time.Second,
	}

	// Mock time.After to capture backoff values
	// For this test, we'll just verify the logic doesn't crash
	result, err := DoWithRetry(ctx, fn, cfg)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("DoWithRetry() result = %v, want 'success'", result)
	}

	// Verify calculateBackoff directly
	t.Run("verify backoff calculation", func(t *testing.T) {
		tests := []struct {
			attempt  int
			expected time.Duration
		}{
			{0, 5 * time.Second},  // 2^0 * 5s = 5s
			{1, 10 * time.Second}, // 2^1 * 5s = 10s
			{2, 15 * time.Second}, // 2^2 * 5s = 20s, capped at 15s
			{3, 15 * time.Second}, // 2^3 * 5s = 40s, capped at 15s
		}

		for _, tt := range tests {
			backoff := calculateBackoff(tt.attempt, cfg.InitialBackoff, cfg.MaxBackoff)
			if backoff != tt.expected {
				t.Errorf("attempt %d: calculateBackoff() = %v, want %v", tt.attempt, backoff, tt.expected)
			}
			backoffs = append(backoffs, backoff)
		}
	})
}

func TestDoWithRetry_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	fn := func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("timeout")
		}
		return "success", nil
	}

	// Empty config should use defaults
	cfg := Config{}

	result, err := DoWithRetry(ctx, fn, cfg)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("DoWithRetry() result = %v, want 'success'", result)
	}
	if callCount != 3 {
		t.Errorf("DoWithRetry() called %d times, want 3 (default MaxAttempts)", callCount)
	}
}

func TestIsRetryable_NetworkErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "connection reset",
			err:  errors.New("connection reset by peer"),
			want: true,
		},
		{
			name: "network error",
			err:  errors.New("network unreachable"),
			want: true,
		},
		{
			name: "temporary failure",
			err:  errors.New("temporary failure in name resolution"),
			want: true,
		},
		{
			name: "eof error",
			err:  errors.New("EOF"),
			want: true,
		},
		{
			name: "500 server error",
			err:  errors.New("HTTP 500 Internal Server Error"),
			want: true,
		},
		{
			name: "503 service unavailable",
			err:  errors.New("HTTP 503 Service Unavailable"),
			want: true,
		},
		{
			name: "5xx error pattern",
			err:  errors.New("server returned 5xx error"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}
