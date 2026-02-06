package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/constants"
)

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
