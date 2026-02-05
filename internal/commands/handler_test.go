package commands

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/constants"
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
	bus              *bus.MessageBus // wraps real bus for channel subscription

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

// TestNewHandler tests creating a new handler
func TestNewHandler(t *testing.T) {
	tests := []struct {
		name      string
		agentLoop AgentLoopInterface
		bus       MessageBusInterface
		logger    *logger.Logger
		onRestart func() error
	}{
		{
			name:      "valid handler with all parameters",
			agentLoop: &MockAgentLoop{},
			bus:       &MockMessageBus{},
			logger:    createTestLogger(t),
			onRestart: func() error { return nil },
		},
		{
			name:      "handler with nil onRestart",
			agentLoop: &MockAgentLoop{},
			bus:       &MockMessageBus{},
			logger:    createTestLogger(t),
			onRestart: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.agentLoop, tt.bus, tt.logger, tt.onRestart)

			if handler == nil {
				t.Fatal("NewHandler() returned nil handler")
			}

			if handler.agentLoop != tt.agentLoop {
				t.Error("NewHandler() agentLoop not set correctly")
			}

			if handler.messageBus != tt.bus {
				t.Error("NewHandler() messageBus not set correctly")
			}

			if handler.logger != tt.logger {
				t.Error("NewHandler() logger not set correctly")
			}

			if tt.onRestart == nil && handler.onRestart != nil {
				t.Error("NewHandler() onRestart should be nil")
			}
		})
	}
}

// TestHandleCommand tests the HandleCommand function
func TestHandleCommand(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		sessionID    string
		userID       string
		channelType  bus.ChannelType
		clearErr     error
		statusErr    error
		publishErr   error
		wantErr      bool
		expectedMsg  string
		onRestartErr error
	}{
		{
			name:        "handle new_session command successfully",
			cmd:         constants.CommandNewSession,
			sessionID:   "session-123",
			userID:      "user-123",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false,
			expectedMsg: constants.MsgSessionCleared,
		},
		{
			name:        "handle status command successfully",
			cmd:         constants.CommandStatus,
			sessionID:   "session-456",
			userID:      "user-456",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false,
			expectedMsg: "📊 **Session Status**",
		},
		{
			name:        "handle restart command successfully",
			cmd:         constants.CommandRestart,
			sessionID:   "session-789",
			userID:      "user-789",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false,
			expectedMsg: constants.MsgRestarting,
		},
		{
			name:        "unknown command returns error",
			cmd:         "unknown",
			sessionID:   "session-999",
			userID:      "user-999",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     true,
		},
		{
			name:        "new_session with clear error",
			cmd:         constants.CommandNewSession,
			sessionID:   "session-error",
			userID:      "user-error",
			channelType: bus.ChannelTypeTelegram,
			clearErr:    errors.New("clear session failed"),
			wantErr:     true,
		},
		{
			name:        "new_session with publish error",
			cmd:         constants.CommandNewSession,
			sessionID:   "session-pub-err",
			userID:      "user-pub-err",
			channelType: bus.ChannelTypeTelegram,
			publishErr:  errors.New("publish failed"),
			wantErr:     true,
		},
		{
			name:        "status with get status error",
			cmd:         constants.CommandStatus,
			sessionID:   "session-status-err",
			userID:      "user-status-err",
			channelType: bus.ChannelTypeTelegram,
			statusErr:   errors.New("get status failed"),
			wantErr:     true,
		},
		{
			name:        "restart with publish error",
			cmd:         constants.CommandRestart,
			sessionID:   "session-restart-err",
			userID:      "user-restart-err",
			channelType: bus.ChannelTypeTelegram,
			publishErr:  errors.New("publish failed"),
			wantErr:     true,
		},
		{
			name:         "restart with callback error",
			cmd:          constants.CommandRestart,
			sessionID:    "session-cb-err",
			userID:       "user-cb-err",
			channelType:  bus.ChannelTypeTelegram,
			onRestartErr: errors.New("callback failed"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			// Setup mocks
			agentLoop.SetClearSessionError(tt.clearErr)

			if tt.statusErr != nil {
				agentLoop.SetSessionStatus(nil, tt.statusErr)
			} else if tt.cmd == constants.CommandStatus {
				agentLoop.SetSessionStatus(map[string]any{
					"session_id":      tt.sessionID,
					"message_count":   5,
					"file_size_human": "1.2 KB",
					"model":           "gpt-4",
					"temperature":     0.7,
					"max_tokens":      4096,
				}, nil)
			}

			messageBus.SetPublishError(tt.publishErr)

			var onRestart func() error
			if tt.onRestartErr != nil {
				onRestart = func() error { return tt.onRestartErr }
			} else {
				onRestart = func() error { return nil }
			}

			handler := NewHandler(agentLoop, messageBus, log, onRestart)

			msg := bus.NewInboundMessage(tt.channelType, tt.userID, tt.sessionID, "test", nil)

			err := handler.HandleCommand(context.Background(), tt.cmd, *msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expectedMsg != "" {
				messages := messageBus.GetOutboundMessages()
				if len(messages) == 0 {
					t.Error("Expected at least one outbound message")
					return
				}
				if !contains(messages[len(messages)-1].Content, tt.expectedMsg) {
					t.Errorf("Expected message to contain %q, got %q", tt.expectedMsg, messages[len(messages)-1].Content)
				}
			}
		})
	}
}

// TestHandleNewSession tests the handleNewSession function
func TestHandleNewSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		userID      string
		channelType bus.ChannelType
		clearErr    error
		publishErr  error
		wantErr     bool
	}{
		{
			name:        "successful session clear",
			sessionID:   "test-session-1",
			userID:      "user-1",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false,
		},
		{
			name:        "session clear with error",
			sessionID:   "test-session-2",
			userID:      "user-2",
			channelType: bus.ChannelTypeTelegram,
			clearErr:    errors.New("clear failed"),
			wantErr:     true,
		},
		{
			name:        "session clear with publish error",
			sessionID:   "test-session-3",
			userID:      "user-3",
			channelType: bus.ChannelTypeTelegram,
			publishErr:  errors.New("publish failed"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			agentLoop.SetClearSessionError(tt.clearErr)
			messageBus.SetPublishError(tt.publishErr)

			handler := NewHandler(agentLoop, messageBus, log, nil)

			msg := bus.NewInboundMessage(tt.channelType, tt.userID, tt.sessionID, "test", nil)

			err := handler.HandleCommand(context.Background(), constants.CommandNewSession, *msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify ClearSession was called
			if !agentLoop.WasClearSessionCalled() && !tt.wantErr {
				t.Error("Expected ClearSession to be called")
			}

			if agentLoop.WasClearSessionCalled() && agentLoop.GetClearSessionID() != tt.sessionID {
				t.Errorf("Expected ClearSession to be called with session ID %q, got %q",
					tt.sessionID, agentLoop.GetClearSessionID())
			}

			// Verify confirmation message was published (unless there was an error)
			if !tt.wantErr {
				messages := messageBus.GetOutboundMessages()
				if len(messages) != 1 {
					t.Errorf("Expected 1 outbound message, got %d", len(messages))
					return
				}
				if messages[0].Content != constants.MsgSessionCleared {
					t.Errorf("Expected message %q, got %q", constants.MsgSessionCleared, messages[0].Content)
				}
			}
		})
	}
}

// TestHandleStatus tests the handleStatus function
func TestHandleStatus(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		userID      string
		channelType bus.ChannelType
		status      map[string]any
		statusErr   error
		publishErr  error
		wantErr     bool
	}{
		{
			name:        "successful status retrieval",
			sessionID:   "test-session-1",
			userID:      "user-1",
			channelType: bus.ChannelTypeTelegram,
			status: map[string]any{
				"session_id":      "test-session-1",
				"message_count":   10,
				"file_size_human": "2.5 KB",
				"model":           "gpt-4",
				"temperature":     0.8,
				"max_tokens":      8192,
			},
			wantErr: false,
		},
		{
			name:        "status retrieval with error",
			sessionID:   "test-session-2",
			userID:      "user-2",
			channelType: bus.ChannelTypeTelegram,
			statusErr:   errors.New("get status failed"),
			wantErr:     true,
		},
		{
			name:        "status retrieval with publish error",
			sessionID:   "test-session-3",
			userID:      "user-3",
			channelType: bus.ChannelTypeTelegram,
			status: map[string]any{
				"session_id":      "test-session-3",
				"message_count":   5,
				"file_size_human": "1.0 KB",
				"model":           "gpt-3.5",
				"temperature":     0.5,
				"max_tokens":      4096,
			},
			publishErr: errors.New("publish failed"),
			wantErr:    true,
		},
		{
			name:        "status with partial data",
			sessionID:   "test-session-4",
			userID:      "user-4",
			channelType: bus.ChannelTypeTelegram,
			status: map[string]any{
				"session_id": "test-session-4",
				// Missing some fields - should still work
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			agentLoop.SetSessionStatus(tt.status, tt.statusErr)
			messageBus.SetPublishError(tt.publishErr)

			handler := NewHandler(agentLoop, messageBus, log, nil)

			msg := bus.NewInboundMessage(tt.channelType, tt.userID, tt.sessionID, "test", nil)

			err := handler.HandleCommand(context.Background(), constants.CommandStatus, *msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify GetSessionStatus was called
			if !agentLoop.WasGetStatusCalled() {
				t.Error("Expected GetSessionStatus to be called")
			}

			if agentLoop.GetStatusSessionID() != tt.sessionID {
				t.Errorf("Expected GetSessionStatus to be called with session ID %q, got %q",
					tt.sessionID, agentLoop.GetStatusSessionID())
			}

			// Verify status message was published
			if tt.statusErr == nil && tt.publishErr == nil {
				messages := messageBus.GetOutboundMessages()
				if len(messages) != 1 {
					t.Errorf("Expected 1 outbound message, got %d", len(messages))
					return
				}
				if !contains(messages[0].Content, "📊 **Session Status**") {
					t.Errorf("Expected status message to contain header, got %q", messages[0].Content)
				}
			}
		})
	}
}

// TestHandleRestart tests the handleRestart function
func TestHandleRestart(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		userID       string
		channelType  bus.ChannelType
		publishErr   error
		callbackErr  error
		onRestartNil bool
		wantErr      bool
	}{
		{
			name:        "successful restart",
			sessionID:   "test-session-1",
			userID:      "user-1",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false,
		},
		{
			name:        "restart with publish error",
			sessionID:   "test-session-2",
			userID:      "user-2",
			channelType: bus.ChannelTypeTelegram,
			publishErr:  errors.New("publish failed"),
			wantErr:     true,
		},
		{
			name:        "restart with callback error",
			sessionID:   "test-session-3",
			userID:      "user-3",
			channelType: bus.ChannelTypeTelegram,
			callbackErr: errors.New("callback failed"),
			wantErr:     true,
		},
		{
			name:         "restart with nil callback",
			sessionID:    "test-session-4",
			userID:       "user-4",
			channelType:  bus.ChannelTypeTelegram,
			onRestartNil: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			messageBus.SetPublishError(tt.publishErr)

			var onRestart func() error
			if tt.onRestartNil {
				onRestart = nil
			} else if tt.callbackErr != nil {
				onRestart = func() error { return tt.callbackErr }
			} else {
				onRestart = func() error { return nil }
			}

			handler := NewHandler(agentLoop, messageBus, log, onRestart)

			msg := bus.NewInboundMessage(tt.channelType, tt.userID, tt.sessionID, "test", nil)

			err := handler.HandleCommand(context.Background(), constants.CommandRestart, *msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify restart notification was published (unless publish failed)
			if tt.publishErr == nil {
				messages := messageBus.GetOutboundMessages()
				if len(messages) != 1 {
					t.Errorf("Expected 1 outbound message, got %d", len(messages))
					return
				}
				if messages[0].Content != constants.MsgRestarting {
					t.Errorf("Expected message %q, got %q", constants.MsgRestarting, messages[0].Content)
				}
			}
		})
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		sessionID   string
		userID      string
		channelType bus.ChannelType
		setupFunc   func(*MockAgentLoop, *MockMessageBus)
		wantErr     bool
	}{
		{
			name:        "empty session ID",
			cmd:         constants.CommandNewSession,
			sessionID:   "",
			userID:      "user-1",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false, // Should still work with empty session ID
		},
		{
			name:        "empty user ID",
			cmd:         constants.CommandStatus,
			sessionID:   "session-1",
			userID:      "",
			channelType: bus.ChannelTypeTelegram,
			wantErr:     false, // Should still work with empty user ID
		},
		{
			name:        "discord channel type",
			cmd:         constants.CommandNewSession,
			sessionID:   "session-1",
			userID:      "user-1",
			channelType: bus.ChannelTypeDiscord,
			wantErr:     false,
		},
		{
			name:        "multiple new_session calls",
			cmd:         constants.CommandNewSession,
			sessionID:   "session-1",
			userID:      "user-1",
			channelType: bus.ChannelTypeTelegram,
			setupFunc: func(al *MockAgentLoop, mb *MockMessageBus) {
				// Set status that will be used
				al.SetSessionStatus(map[string]any{
					"session_id":      "session-1",
					"message_count":   5,
					"file_size_human": "1.0 KB",
					"model":           "gpt-4",
					"temperature":     0.7,
					"max_tokens":      4096,
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			if tt.setupFunc != nil {
				tt.setupFunc(agentLoop, messageBus)
			}

			handler := NewHandler(agentLoop, messageBus, log, nil)

			msg := bus.NewInboundMessage(tt.channelType, tt.userID, tt.sessionID, "test", nil)

			err := handler.HandleCommand(context.Background(), tt.cmd, *msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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

// TestMessageChannelTypes tests handling messages from different channel types
func TestMessageChannelTypes(t *testing.T) {
	channelTypes := []bus.ChannelType{
		bus.ChannelTypeTelegram,
		bus.ChannelTypeDiscord,
		bus.ChannelTypeSlack,
		bus.ChannelTypeWeb,
		bus.ChannelTypeAPI,
	}

	for _, channelType := range channelTypes {
		t.Run(string(channelType), func(t *testing.T) {
			agentLoop := &MockAgentLoop{}
			messageBus := &MockMessageBus{}
			log := createTestLogger(t)

			agentLoop.SetSessionStatus(map[string]any{
				"session_id":      "test-session",
				"message_count":   1,
				"file_size_human": "100 B",
				"model":           "gpt-4",
				"temperature":     0.7,
				"max_tokens":      4096,
			}, nil)

			handler := NewHandler(agentLoop, messageBus, log, nil)

			msg := bus.NewInboundMessage(channelType, "user-1", "session-1", "test", nil)

			err := handler.HandleCommand(context.Background(), constants.CommandStatus, *msg)

			if err != nil {
				t.Errorf("HandleCommand() error = %v", err)
			}

			messages := messageBus.GetOutboundMessages()
			if len(messages) != 1 {
				t.Errorf("Expected 1 outbound message, got %d", len(messages))
			}

			// Verify the channel type is preserved
			if messages[0].ChannelType != channelType {
				t.Errorf("Expected channel type %s, got %s", channelType, messages[0].ChannelType)
			}
		})
	}
}
