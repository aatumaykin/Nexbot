package commands

import (
	"context"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/constants"
)

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
