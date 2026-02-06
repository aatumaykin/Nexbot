package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/constants"
)

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
