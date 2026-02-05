package tools

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestMatchPattern(t *testing.T) {
	cfg := &config.Config{
		Workspace: config.WorkspaceConfig{
			Path: "/tmp/test",
		},
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled: true,
			},
		},
	}
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	tool := NewShellExecTool(cfg, log)

	tests := []struct {
		name     string
		command  string
		pattern  string
		expected bool
	}{
		// Exact match
		{
			name:     "exact match",
			command:  "echo hello",
			pattern:  "echo hello",
			expected: true,
		},
		{
			name:     "exact match with multiple arguments",
			command:  "git commit -m 'test'",
			pattern:  "git commit -m 'test'",
			expected: true,
		},

		// Base command match
		{
			name:     "base command match - echo",
			command:  "echo hello",
			pattern:  "echo",
			expected: true,
		},
		{
			name:     "base command match - git commit",
			command:  "git commit",
			pattern:  "git",
			expected: true,
		},
		{
			name:     "base command mismatch",
			command:  "echo hello",
			pattern:  "git",
			expected: false,
		},

		// Wildcard with one *
		{
			name:     "wildcard - git status matches 'git *'",
			command:  "git status",
			pattern:  "git *",
			expected: true,
		},
		{
			name:     "wildcard - docker run matches 'docker *'",
			command:  "docker run",
			pattern:  "docker *",
			expected: true,
		},
		{
			name:     "wildcard - git commit matches 'git *'",
			command:  "git commit",
			pattern:  "git *",
			expected: true,
		},
		{
			name:     "wildcard - command doesn't match 'git *'",
			command:  "docker run",
			pattern:  "git *",
			expected: false,
		},
		{
			name:     "wildcard - no space after command",
			command:  "gitstatus",
			pattern:  "git *",
			expected: false,
		},

		// Full wildcard *
		{
			name:     "full wildcard - any command",
			command:  "echo hello",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "full wildcard - complex command",
			command:  "docker run --rm -v /tmp:/tmp alpine ls",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "full wildcard - empty pattern",
			command:  "echo hello",
			pattern:  "",
			expected: false,
		},

		// Whitespace handling
		{
			name:     "trim whitespace - command",
			command:  "  echo hello  ",
			pattern:  "echo",
			expected: true,
		},
		{
			name:     "trim whitespace - pattern",
			command:  "echo hello",
			pattern:  "  echo  ",
			expected: true,
		},
		{
			name:     "trim whitespace - wildcard pattern",
			command:  "git status",
			pattern:  "  git *  ",
			expected: true,
		},

		// Edge cases
		{
			name:     "empty command",
			command:  "",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "empty pattern, non-empty command",
			command:  "echo hello",
			pattern:  "",
			expected: false,
		},
		{
			name:     "both empty",
			command:  "",
			pattern:  "",
			expected: false,
		},
		{
			name:     "wildcard prefix with no space",
			command:  "gitstatus",
			pattern:  "git*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.matchPattern(tt.command, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v",
					tt.command, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})

	tests := []struct {
		name              string
		denyCommands      []string
		askCommands       []string
		allowedCommands   []string
		command           string
		expectedError     bool
		errorContains     string
		isConfirmRequired bool
	}{
		{
			name:            "all lists empty - fail-open, command allowed",
			denyCommands:    []string{},
			askCommands:     []string{},
			allowedCommands: []string{},
			command:         "rm -rf /",
			expectedError:   false,
		},
		{
			name:            "deny command - denied",
			denyCommands:    []string{"rm *", "dd *"},
			askCommands:     []string{},
			allowedCommands: []string{},
			command:         "rm -rf /",
			expectedError:   true,
			errorContains:   "denied by deny_commands",
		},
		{
			name:            "deny by wildcard - denied",
			denyCommands:    []string{"rm *"},
			askCommands:     []string{},
			allowedCommands: []string{},
			command:         "rm -rf test.txt",
			expectedError:   true,
			errorContains:   "denied by deny_commands",
		},
		{
			name:              "ask command - requires confirmation",
			denyCommands:      []string{},
			askCommands:       []string{"docker *"},
			allowedCommands:   []string{},
			command:           "docker run alpine",
			expectedError:     true,
			errorContains:     "# CONFIRM_REQUIRED:",
			isConfirmRequired: true,
		},
		{
			name:            "allowed command - allowed",
			denyCommands:    []string{},
			askCommands:     []string{},
			allowedCommands: []string{"git *", "ls", "echo"},
			command:         "git status",
			expectedError:   false,
		},
		{
			name:            "allowed command not in list - denied",
			denyCommands:    []string{},
			askCommands:     []string{},
			allowedCommands: []string{"git *", "ls"},
			command:         "rm file.txt",
			expectedError:   true,
			errorContains:   "command not allowed",
		},
		{
			name:            "deny + ask + allowed - deny takes precedence",
			denyCommands:    []string{"rm *"},
			askCommands:     []string{"docker *"},
			allowedCommands: []string{"git *"},
			command:         "rm -rf /",
			expectedError:   true,
			errorContains:   "denied by deny_commands",
		},
		{
			name:              "deny + ask + allowed - ask takes second precedence",
			denyCommands:      []string{"rm *"},
			askCommands:       []string{"docker *"},
			allowedCommands:   []string{"git *"},
			command:           "docker run alpine",
			expectedError:     true,
			errorContains:     "# CONFIRM_REQUIRED:",
			isConfirmRequired: true,
		},
		{
			name:            "deny + ask - command not in deny or ask, but no allowed list - allowed",
			denyCommands:    []string{"rm *"},
			askCommands:     []string{"docker *"},
			allowedCommands: []string{},
			command:         "ls -la",
			expectedError:   false,
		},
		{
			name:            "deny + allowed - command not in deny or allowed - denied",
			denyCommands:    []string{"rm *"},
			askCommands:     []string{},
			allowedCommands: []string{"git *"},
			command:         "ls -la",
			expectedError:   true,
			errorContains:   "command not allowed",
		},
		{
			name:            "exact match deny",
			denyCommands:    []string{"git push origin main"},
			askCommands:     []string{},
			allowedCommands: []string{},
			command:         "git push origin main",
			expectedError:   true,
			errorContains:   "denied by deny_commands",
		},
		{
			name:            "base command deny",
			denyCommands:    []string{"dd"},
			askCommands:     []string{},
			allowedCommands: []string{},
			command:         "dd if=/dev/zero of=/dev/sda",
			expectedError:   true,
			errorContains:   "denied by deny_commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Workspace: config.WorkspaceConfig{
					Path: "/tmp/test",
				},
				Tools: config.ToolsConfig{
					Shell: config.ShellToolConfig{
						Enabled:         true,
						DenyCommands:    tt.denyCommands,
						AskCommands:     tt.askCommands,
						AllowedCommands: tt.allowedCommands,
					},
				},
			}
			tool := NewShellExecTool(cfg, log)

			err := tool.validateCommand(tt.command)

			if tt.expectedError {
				if err == nil {
					t.Errorf("validateCommand(%q) expected error, got nil", tt.command)
				} else if tt.errorContains != "" {
					errStr := err.Error()
					if !containsSubstring(errStr, tt.errorContains) {
						t.Errorf("validateCommand(%q) error = %q, expected to contain %q",
							tt.command, errStr, tt.errorContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("validateCommand(%q) expected no error, got: %v", tt.command, err)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
