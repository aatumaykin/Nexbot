package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// ShellExecTool implements the Tool interface for executing shell commands.
// It executes shell commands with security restrictions (whitelist, timeout).
type ShellExecTool struct {
	cfg    *config.Config
	logger *logger.Logger
}

// ShellExecArgs represents the arguments for the shell_exec tool.
type ShellExecArgs struct {
	Command string `json:"command"` // Shell command to execute
}

// NewShellExecTool creates a new ShellExecTool instance.
// The config parameter provides the shell tool configuration (whitelist, timeout, etc.).
func NewShellExecTool(cfg *config.Config, log *logger.Logger) *ShellExecTool {
	return &ShellExecTool{
		cfg:    cfg,
		logger: log,
	}
}

// Name returns the tool name.
func (t *ShellExecTool) Name() string {
	return "shell_exec"
}

// Description returns a description of what the tool does.
func (t *ShellExecTool) Description() string {
	return "Executes shell commands with security restrictions. Only whitelisted commands are allowed. Commands have a timeout and are logged."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *ShellExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute.",
			},
		},
		"required": []string{"command"},
	}
}

// Execute executes a shell command.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *ShellExecTool) Execute(args string) (string, error) {
	// Parse arguments
	var shellArgs ShellExecArgs
	if err := parseJSON(args, &shellArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate arguments
	if shellArgs.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Trim whitespace
	shellArgs.Command = strings.TrimSpace(shellArgs.Command)

	// Check if shell tool is enabled
	if !t.cfg.Tools.Shell.Enabled {
		return "", fmt.Errorf("shell_exec tool is disabled in configuration")
	}

	// Validate command against whitelist
	if err := t.validateCommand(shellArgs.Command); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Determine timeout
	timeout := time.Duration(t.cfg.Tools.Shell.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	// Log the command execution
	if t.logger != nil {
		t.logger.Info("Executing shell command", logger.Field{Key: "command", Value: shellArgs.Command})
	}

	// Execute command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute command in workspace
	output, err := t.executeCommand(ctx, shellArgs.Command, t.cfg.Workspace.Path)

	// Log result
	if t.logger != nil {
		if err != nil {
			t.logger.Error("Shell command failed", err, logger.Field{Key: "output", Value: output})
		} else {
			t.logger.Info("Shell command succeeded")
		}
	}

	// Format output
	result := fmt.Sprintf("# Command: %s\n", shellArgs.Command)
	result += fmt.Sprintf("# Exit code: %v\n", getExitCode(err))
	result += "# Output:\n"
	result += output

	if err != nil {
		result += fmt.Sprintf("\n# Error: %v", err)
	}

	return result, nil
}

// validateCommand validates that a command is in the whitelist.
func (t *ShellExecTool) validateCommand(command string) error {
	// If no whitelist is configured, deny all commands (fail-safe)
	if len(t.cfg.Tools.Shell.AllowedCommands) == 0 {
		return fmt.Errorf("no commands are whitelisted in configuration")
	}

	// Check if the command matches any allowed pattern
	for _, allowed := range t.cfg.Tools.Shell.AllowedCommands {
		if t.matchPattern(command, allowed) {
			return nil
		}
	}

	// Extract base command for error message
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("command is empty")
	}
	baseCommand := parts[0]

	// Command not in whitelist
	return fmt.Errorf("command '%s' is not in the allowed commands whitelist: %v",
		baseCommand, t.cfg.Tools.Shell.AllowedCommands)
}

// matchPattern checks if a command matches a given pattern.
// Pattern types:
//   - Exact match: "echo hello" matches "echo hello"
//   - Base command: "echo hello" matches "echo"
//   - Wildcard with one *: "git status" matches "git *"
//   - Full wildcard: "echo hello" matches "*"
func (t *ShellExecTool) matchPattern(command, pattern string) bool {
	// Trim whitespace
	command = strings.TrimSpace(command)
	pattern = strings.TrimSpace(pattern)

	// Full wildcard: allow all commands
	if pattern == "*" {
		return true
	}

	// Both empty is not a match
	if command == "" && pattern == "" {
		return false
	}

	// Exact match
	if command == pattern {
		return true
	}

	// Base command match: pattern contains only the command name
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	baseCommand := parts[0]
	if pattern == baseCommand {
		return true
	}

	// Wildcard with one *: e.g., "git *" matches "git status"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		prefix = strings.TrimSpace(prefix)
		// Command must start with the prefix
		if prefix != "" && strings.HasPrefix(command, prefix) {
			// Ensure the prefix is followed by whitespace or nothing
			remaining := strings.TrimPrefix(command, prefix)
			return remaining == "" || strings.HasPrefix(remaining, " ")
		}
	}

	return false
}

// executeCommand executes a shell command and returns its combined stdout/stderr.
func (t *ShellExecTool) executeCommand(ctx context.Context, command, workingDir string) (string, error) {
	// Execute shell command (using sh -c for shell expansion)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Set working directory to workspace
	cmd.Dir = workingDir

	// Capture stdout and stderr combined
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.String() != "" {
		output += "\n" + stderr.String()
	}

	return output, err
}

// getExitCode extracts the exit code from an error.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Check if it's an exit error
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}

	// For context timeout or other errors, return -1
	return -1
}
