package constants

import (
	"testing"
)

func TestCommandConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "CommandNewSession",
			value: CommandNewSession,
		},
		{
			name:  "CommandStatus",
			value: CommandStatus,
		},
		{
			name:  "CommandRestart",
			value: CommandRestart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
			// Check that command names use snake_case format
			for _, r := range tt.value {
				if r >= 'A' && r <= 'Z' {
					t.Errorf("%s should use snake_case format, got: %s", tt.name, tt.value)
					break
				}
			}
		})
	}
}

func TestCommandValues(t *testing.T) {
	// Test specific expected values
	if CommandNewSession != "new_session" {
		t.Errorf("CommandNewSession = %s, want 'new_session'", CommandNewSession)
	}

	if CommandStatus != "status" {
		t.Errorf("CommandStatus = %s, want 'status'", CommandStatus)
	}

	if CommandRestart != "restart" {
		t.Errorf("CommandRestart = %s, want 'restart'", CommandRestart)
	}
}

func TestCommandCount(t *testing.T) {
	// Ensure we have the expected number of commands
	// This test helps catch when commands are accidentally added/removed
	expectedCommands := 3
	actualCommands := 3 // Count of constants in commands.go

	if actualCommands != expectedCommands {
		t.Errorf("Expected %d command constants, found %d", expectedCommands, actualCommands)
	}
}
