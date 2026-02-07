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
	return "Execute shell commands with security restrictions (whitelist, timeout, logging)."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *ShellExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute. Examples: ls -la, pwd, cat /var/log/syslog, df -h",
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

	// Validate command against deny/ask/allowed lists
	if err := t.validateCommand(shellArgs.Command); err != nil {
		// Check if confirmation is required
		if strings.Contains(err.Error(), "# CONFIRM_REQUIRED:") {
			return err.Error(), nil
		}
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

// validateCommand validates a command against deny/ask/allowed lists in order.
// Validation order: deny → ask → allowed
func (t *ShellExecTool) validateCommand(command string) error {
	denyCommands := t.cfg.Tools.Shell.DenyCommands
	askCommands := t.cfg.Tools.Shell.AskCommands
	allowedCommands := t.cfg.Tools.Shell.AllowedCommands

	// Step 0: Check for path traversal in arguments
	for _, arg := range strings.Fields(command) {
		if strings.Contains(arg, "..") {
			return fmt.Errorf("argument contains path traversal: %s", arg)
		}
	}

	// Step 1: Check deny_commands - if command matches, deny immediately
	for _, denyPattern := range denyCommands {
		if t.matchPattern(command, denyPattern) {
			return fmt.Errorf("denied by deny_commands")
		}
	}

	// Step 2: Check ask_commands - if command matches, require confirmation
	for _, askPattern := range askCommands {
		if t.matchPattern(command, askPattern) {
			return fmt.Errorf("# CONFIRM_REQUIRED: Command '%s' requires confirmation", command)
		}
	}

	// Step 3: Check allowed_commands
	// If allowed_commands is empty and both deny and ask are empty, allow all (fail-open)
	if len(allowedCommands) == 0 && len(denyCommands) == 0 && len(askCommands) == 0 {
		return nil // All commands allowed
	}

	// If allowed_commands is configured, command must match at least one pattern
	if len(allowedCommands) > 0 {
		for _, allowedPattern := range allowedCommands {
			if t.matchPattern(command, allowedPattern) {
				return nil // Command is allowed
			}
		}
		// Command didn't match any allowed pattern
		return fmt.Errorf("command not allowed")
	}

	// allowed_commands is empty, but deny or ask was configured - command is allowed
	return nil
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
		// Validate prefix doesn't contain dangerous characters
		if prefix != "" && strings.ContainsAny(prefix, "|&;<>`$()") {
			// Unsafe pattern - reject to prevent command injection
			return false
		}
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
