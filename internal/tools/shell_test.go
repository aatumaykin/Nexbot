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
