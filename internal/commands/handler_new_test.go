package commands

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/logger"
)

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
