package heartbeat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestChecker creates a checker with a custom interval for testing.
// This is needed because NewChecker only accepts minutes, but we need
// shorter intervals for faster tests.
func newTestChecker(interval time.Duration, agent *mockAgent, log *logger.Logger) *Checker {
	return &Checker{
		interval: interval,
		agent:    agent,
		logger:   log,
		started:  false,
	}
}

// mockAgent implements the Agent interface for testing.
type mockAgent struct {
	response  string
	err       error
	callCount int
	mu        sync.Mutex
	delay     time.Duration // Optional delay to simulate processing time
}

func (m *mockAgent) ProcessHeartbeatCheck(ctx context.Context) (string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return m.response, m.err
}

// getCallCount returns the number of times ProcessHeartbeatCheck was called.
func (m *mockAgent) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestNewChecker(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK"}
	checker := NewChecker(10, agent, log)

	assert.NotNil(t, checker)
	assert.Equal(t, 10*time.Minute, checker.interval)
	assert.NotNil(t, checker.agent)
	assert.NotNil(t, checker.logger)
	assert.False(t, checker.started)
}

func TestCheckerStartStop(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK"}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	// Start checker
	err = checker.Start()
	assert.NoError(t, err)
	assert.True(t, checker.started)

	// Start again should not error
	err = checker.Start()
	assert.NoError(t, err)

	// Wait for ticker to trigger (interval + some margin)
	t.Logf("Waiting for ticker to trigger...")
	time.Sleep(700 * time.Millisecond)
	t.Logf("Wait complete. Agent call count: %d", agent.getCallCount())

	// Stop checker
	err = checker.Stop()
	assert.NoError(t, err)
	assert.False(t, checker.started)

	// Stop again should not error
	err = checker.Stop()
	assert.NoError(t, err)

	// Verify agent was called at least once
	assert.GreaterOrEqual(t, agent.getCallCount(), 1, "Agent should be called at least once")
}

func TestCheckerProcessResponseOK(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK"}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)
	defer checker.Stop()

	// Wait for at least one check to run
	time.Sleep(700 * time.Millisecond)

	// Verify agent was called
	assert.GreaterOrEqual(t, agent.getCallCount(), 1)
}

func TestCheckerProcessResponseAlert(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "ALERT: Something needs attention"}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)
	defer checker.Stop()

	// Wait for at least one check to run
	time.Sleep(700 * time.Millisecond)

	// Verify agent was called
	assert.GreaterOrEqual(t, agent.getCallCount(), 1)
}

func TestCheckerHeartbeatOKToken(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name:     "Exact match",
			response: "HEARTBEAT_OK",
			expected: true,
		},
		{
			name:     "With newline prefix",
			response: "\nHEARTBEAT_OK",
			expected: true,
		},
		{
			name:     "With newline suffix",
			response: "HEARTBEAT_OK\n",
			expected: true,
		},
		{
			name:     "Contains token but not exact match",
			response: "HEARTBEAT_OK something else",
			expected: false,
		},
		{
			name:     "Contains token in middle",
			response: "prefix HEARTBEAT_OK suffix",
			expected: false,
		},
		{
			name:     "Empty response",
			response: "",
			expected: false,
		},
		{
			name:     "Different token",
			response: "NOT_OK",
			expected: false,
		},
		{
			name:     "Alert message",
			response: "ALERT: Task overdue",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsToken(tt.response, heartbeatOKToken)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckerProcessResponseEmpty(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: ""}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)
	defer checker.Stop()

	// Wait for at least one check to run
	time.Sleep(700 * time.Millisecond)

	// Verify agent was called despite empty response
	assert.GreaterOrEqual(t, agent.getCallCount(), 1)
}

func TestCheckerAgentError(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	expectedErr := assert.AnError
	agent := &mockAgent{response: "", err: expectedErr}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)
	defer checker.Stop()

	// Wait for at least one check to run
	time.Sleep(700 * time.Millisecond)

	// Verify agent was called even though it returned an error
	assert.GreaterOrEqual(t, agent.getCallCount(), 1)
}

func TestCheckerConcurrentStartStop(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK"}
	checker := NewChecker(1, agent, log)

	var wg sync.WaitGroup

	// Start multiple times concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = checker.Start()
		}()
	}

	wg.Wait()

	assert.True(t, checker.started)

	// Stop multiple times concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = checker.Stop()
		}()
	}

	wg.Wait()

	assert.False(t, checker.started)
}

func TestCheckerMultipleIntervals(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK", delay: 50 * time.Millisecond}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)

	// Wait for multiple intervals
	time.Sleep(1500 * time.Millisecond)

	err = checker.Stop()
	require.NoError(t, err)

	// Should have been called multiple times
	callCount := agent.getCallCount()
	assert.GreaterOrEqual(t, callCount, 2)
}

func TestCheckerContextCancellation(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	agent := &mockAgent{response: "HEARTBEAT_OK", delay: 5 * time.Second}
	checker := newTestChecker(500*time.Millisecond, agent, log)

	err = checker.Start()
	require.NoError(t, err)

	// Start a check but stop before it completes
	time.Sleep(100 * time.Millisecond)

	startTime := time.Now()
	err = checker.Stop()
	assert.NoError(t, err)

	// Stop should not wait for the long delay to complete
	assert.Less(t, time.Since(startTime), 2*time.Second)
}
