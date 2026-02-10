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
	cfg       *config.Config
	logger    *logger.Logger
	validator *ShellValidator
	resolver  *secretResolverWrapper
}

// ShellExecArgs represents the arguments for the shell_exec tool.
type ShellExecArgs struct {
	Command string `json:"command"` // Shell command to execute
}

// NewShellExecTool creates a new ShellExecTool instance.
// The config parameter provides the shell tool configuration (whitelist, timeout, etc.).
func NewShellExecTool(cfg *config.Config, log *logger.Logger) *ShellExecTool {
	return &ShellExecTool{
		cfg:       cfg,
		logger:    log,
		validator: NewShellValidatorFromConfig(cfg.Tools.Shell),
		resolver:  nil,
	}
}

// SetSecretResolver sets the secret resolver function.
// This is called by the tool executor before execution.
func (t *ShellExecTool) SetSecretResolver(resolver func(string, string) string) {
	t.resolver = &secretResolverWrapper{
		resolve: resolver,
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
	return t.ExecuteWithContext(context.Background(), args)
}

// ExecuteWithContext executes a shell command with context support.
// The context is used for cancellation and timeouts.
// It also resolves secret references in the command.
func (t *ShellExecTool) ExecuteWithContext(ctx context.Context, args string) (string, error) {
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

	// Resolve secrets in command
	resolvedCommand := t.resolveSecrets(ctx, shellArgs.Command)

	// Check if shell tool is enabled
	if !t.cfg.Tools.Shell.Enabled {
		return "", fmt.Errorf("shell_exec tool is disabled in configuration")
	}

	// Validate command against deny/ask/allowed lists
	if err := t.validator.Validate(resolvedCommand); err != nil {
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

	// Get sessionID for logging (if available)
	sessionID := getSessionID(ctx)

	// Log the command execution (with masked secrets)
	if t.logger != nil {
		maskedCommand := t.maskSecrets(resolvedCommand)
		t.logger.Info("Executing shell command",
			logger.Field{Key: "command", Value: maskedCommand},
			logger.Field{Key: "session_id", Value: sessionID})
	}

	// Execute command with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command in workspace
	output, err := t.executeCommand(execCtx, resolvedCommand, t.cfg.Workspace.Path)

	// Log result
	if t.logger != nil {
		if err != nil {
			t.logger.Error("Shell command failed", err, logger.Field{Key: "session_id", Value: sessionID})
		} else {
			t.logger.Info("Shell command succeeded", logger.Field{Key: "session_id", Value: sessionID})
		}
	}

	// Format output (mask secrets)
	result := fmt.Sprintf("# Command: %s\n", t.maskSecrets(resolvedCommand))
	result += fmt.Sprintf("# Exit code: %v\n", getExitCode(err))
	result += "# Output:\n"
	result += output

	if err != nil {
		result += fmt.Sprintf("\n# Error: %v", err)
	}

	return result, nil
}

// resolveSecrets resolves secret references in the command.
func (t *ShellExecTool) resolveSecrets(ctx context.Context, command string) string {
	if t.resolver == nil {
		return command
	}

	sessionID := getSessionID(ctx)
	return t.resolver.Resolve(sessionID, command)
}

// maskSecrets masks secret values in the command for logging.
func (t *ShellExecTool) maskSecrets(command string) string {
	if t.resolver == nil {
		return command
	}
	return t.resolver.MaskSecrets(command)
}

// getSessionID extracts sessionID from context.
func getSessionID(ctx context.Context) string {
	if sessionID, ok := ctx.Value(sessionIDKey).(string); ok {
		return sessionID
	}
	return ""
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

// secretResolverWrapper wraps a secret resolver function.
type secretResolverWrapper struct {
	resolve func(string, string) string
	// Track resolved secrets for masking purposes
	resolvedSecrets map[string]string
}

// Resolve resolves secret references in text.
func (w *secretResolverWrapper) Resolve(sessionID, text string) string {
	if w.resolve == nil {
		return text
	}

	// Track which secrets were resolved for masking
	w.resolvedSecrets = make(map[string]string)

	// Simple regex-like pattern matching for $SECRET_NAME
	result := text
	pos := 0

	for pos < len(result) {
		dollarPos := strings.Index(result[pos:], "$")
		if dollarPos == -1 {
			break
		}
		dollarPos += pos

		// Extract secret name
		secretName, endPos := extractSecretName(result, dollarPos+1)
		if secretName == "" {
			pos = dollarPos + 1
			continue
		}

		// Resolve the secret
		secretValue := w.resolve(sessionID, secretName)

		// Track for masking
		w.resolvedSecrets[secretName] = secretValue

		// Replace
		result = result[:dollarPos] + secretValue + result[endPos:]
		pos = dollarPos + len(secretValue)
	}

	return result
}

// MaskSecrets masks resolved secret values in text.
func (w *secretResolverWrapper) MaskSecrets(text string) string {
	if len(w.resolvedSecrets) == 0 {
		return text
	}

	result := text
	for secretName := range w.resolvedSecrets {
		reference := "$" + secretName
		result = strings.ReplaceAll(result, reference, "***")
	}
	return result
}

// extractSecretName extracts a secret name from text starting at given position.
func extractSecretName(text string, startPos int) (string, int) {
	if startPos >= len(text) {
		return "", startPos
	}

	endPos := startPos
	for endPos < len(text) {
		c := text[endPos]
		if !isAlphaNumeric(c) && c != '_' {
			break
		}
		endPos++
	}

	if endPos == startPos {
		return "", startPos
	}

	return text[startPos:endPos], endPos
}

// isAlphaNumeric checks if a byte is alphanumeric.
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}
