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

	// Get working directory
	workingDir := t.cfg.Tools.Shell.WorkingDir

	// Execute command
	output, err := t.executeCommand(ctx, shellArgs.Command, workingDir)

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

	// Extract the base command (first word before any arguments)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("command is empty")
	}

	baseCommand := parts[0]

	// Check if the base command is in the whitelist
	for _, allowed := range t.cfg.Tools.Shell.AllowedCommands {
		// Wildcard: allow all commands
		if allowed == "*" {
			return nil
		}

		// Exact match
		if baseCommand == allowed {
			return nil
		}

		// Check if whitelist entry is a path to the command
		if strings.Contains(allowed, baseCommand) {
			return nil
		}
	}

	// Command not in whitelist
	return fmt.Errorf("command '%s' is not in the allowed commands whitelist: %v",
		baseCommand, t.cfg.Tools.Shell.AllowedCommands)
}

// executeCommand executes a shell command and returns its combined stdout/stderr.
func (t *ShellExecTool) executeCommand(ctx context.Context, command, workingDir string) (string, error) {
	// Execute shell command (using sh -c for shell expansion)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Set working directory if configured
	if workingDir != "" {
		cmd.Dir = workingDir
	}

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
