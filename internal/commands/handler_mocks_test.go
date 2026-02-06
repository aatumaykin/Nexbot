package commands

import (
	"context"
	"sync"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// MockAgentLoop is a mock implementation of AgentLoopInterface for testing
type MockAgentLoop struct {
	mu               sync.Mutex
	clearSessionErr  error
	getSessionStatus map[string]any
	getStatusErr     error

	clearSessionCalled bool
	clearSessionID     string
	getStatusCalled    bool
	getStatusSessionID string
}

func (m *MockAgentLoop) ClearSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearSessionCalled = true
	m.clearSessionID = sessionID
	return m.clearSessionErr
}

func (m *MockAgentLoop) GetSessionStatus(ctx context.Context, sessionID string) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getStatusCalled = true
	m.getStatusSessionID = sessionID
	return m.getSessionStatus, m.getStatusErr
}

// Reset resets the mock state
func (m *MockAgentLoop) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearSessionCalled = false
	m.clearSessionID = ""
	m.clearSessionErr = nil
	m.getSessionStatus = nil
	m.getStatusErr = nil
	m.getStatusCalled = false
	m.getStatusSessionID = ""
}

// SetClearSessionError sets the error to return from ClearSession
func (m *MockAgentLoop) SetClearSessionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearSessionErr = err
}

// SetSessionStatus sets the status to return from GetSessionStatus
func (m *MockAgentLoop) SetSessionStatus(status map[string]any, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getSessionStatus = status
	m.getStatusErr = err
}

// WasClearSessionCalled returns true if ClearSession was called
func (m *MockAgentLoop) WasClearSessionCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clearSessionCalled
}

// GetClearSessionID returns the session ID passed to ClearSession
func (m *MockAgentLoop) GetClearSessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clearSessionID
}

// WasGetStatusCalled returns true if GetSessionStatus was called
func (m *MockAgentLoop) WasGetStatusCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getStatusCalled
}

// GetStatusSessionID returns the session ID passed to GetSessionStatus
func (m *MockAgentLoop) GetStatusSessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getStatusSessionID
}

// MockMessageBus is a mock implementation of MessageBusInterface for testing
type MockMessageBus struct {
	mu               sync.Mutex
	outboundMessages []bus.OutboundMessage
	publishErr       error

	publishCalled bool
}

func (m *MockMessageBus) PublishOutbound(msg bus.OutboundMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishCalled = true
	if m.publishErr != nil {
		return m.publishErr
	}
	m.outboundMessages = append(m.outboundMessages, msg)
	return nil
}

// Reset resets the mock state
func (m *MockMessageBus) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outboundMessages = nil
	m.publishErr = nil
	m.publishCalled = false
}

// SetPublishError sets the error to return from PublishOutbound
func (m *MockMessageBus) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishErr = err
}

// GetOutboundMessages returns all published outbound messages
func (m *MockMessageBus) GetOutboundMessages() []bus.OutboundMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.outboundMessages
}

// WasPublishCalled returns true if PublishOutbound was called
func (m *MockMessageBus) WasPublishCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishCalled
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}

// Helper function to create a test logger
func createTestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return logger
}
