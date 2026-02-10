package tools

import (
	"fmt"
	"strings"

	"github.com/aatumaykin/nexbot/internal/config"
)

// ShellValidator handles validation of shell commands against deny/ask/allowed lists.
type ShellValidator struct {
	denyCommands    []string
	askCommands     []string
	allowedCommands []string
}

// NewShellValidator creates a new ShellValidator with the given command lists.
func NewShellValidator(denyCommands, askCommands, allowedCommands []string) *ShellValidator {
	return &ShellValidator{
		denyCommands:    denyCommands,
		askCommands:     askCommands,
		allowedCommands: allowedCommands,
	}
}

// NewShellValidatorFromConfig creates a ShellValidator from config.ShellToolConfig.
func NewShellValidatorFromConfig(cfg config.ShellToolConfig) *ShellValidator {
	return &ShellValidator{
		denyCommands:    cfg.DenyCommands,
		askCommands:     cfg.AskCommands,
		allowedCommands: cfg.AllowedCommands,
	}
}

// Validate validates a command against deny/ask/allowed lists in order.
// Validation order: deny → ask → allowed
func (v *ShellValidator) Validate(command string) error {
	// Step 0: Check for shell injection vectors (command chaining, substitution)
	if err := v.checkShellInjection(command); err != nil {
		return err
	}

	// Step 1: Check for path traversal in arguments
	cmdName, args, err := parseCommandArgsForValidation(command)
	if err != nil {
		return fmt.Errorf("failed to parse command for validation: %w", err)
	}

	// Check path traversal in all arguments (including command name)
	for _, arg := range append([]string{cmdName}, args...) {
		if strings.Contains(arg, "..") {
			return fmt.Errorf("argument contains path traversal: %s", arg)
		}
	}

	// Step 2: Check deny_commands - if command matches, deny immediately
	for _, denyPattern := range v.denyCommands {
		if v.MatchPattern(command, denyPattern) {
			return fmt.Errorf("denied by deny_commands")
		}
	}

	// Step 3: Check ask_commands - if command matches, require confirmation
	for _, askPattern := range v.askCommands {
		if v.MatchPattern(command, askPattern) {
			return fmt.Errorf("# CONFIRM_REQUIRED: Command '%s' requires confirmation", command)
		}
	}

	// Step 4: Check allowed_commands
	// If allowed_commands is empty and both deny and ask are empty, allow all (fail-open)
	if len(v.allowedCommands) == 0 && len(v.denyCommands) == 0 && len(v.askCommands) == 0 {
		return nil // All commands allowed
	}

	// If allowed_commands is configured, command must match at least one pattern
	if len(v.allowedCommands) > 0 {
		for _, allowedPattern := range v.allowedCommands {
			if v.MatchPattern(command, allowedPattern) {
				return nil // Command is allowed
			}
		}
		// Command didn't match any allowed pattern
		return fmt.Errorf("command not allowed")
	}

	// allowed_commands is empty, but deny or ask was configured - command is allowed
	return nil
}

// MatchPattern checks if a command matches a given pattern.
// Pattern types:
//   - Exact match: "echo hello" matches "echo hello"
//   - Base command: "echo hello" matches "echo"
//   - Wildcard with one *: "git status" matches "git *"
//   - Full wildcard: "echo hello" matches "*"
func (v *ShellValidator) MatchPattern(command, pattern string) bool {
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

// checkShellInjection checks for shell injection vectors in the command.
// It looks for command chaining, substitution, and other shell-specific patterns.
func (v *ShellValidator) checkShellInjection(command string) error {
	// Define dangerous shell characters/patterns
	dangerousPatterns := []string{
		"&&", "||", "|", ";",
		"$(", ")",
		"`",
		">>", ">",
		"<", "<<",
		"&",
	}

	// Check for dangerous patterns (but allow $SECRET_NAME for secret resolution)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			// Special case: $SECRET_NAME is allowed (for secret resolution)
			if pattern == "$(" && strings.Contains(command, "$") {
				// Check if $ is followed by a secret name pattern (letters/numbers/underscore)
				hasSecretRef := false
				pos := 0
				for {
					dollarIdx := strings.Index(command[pos:], "$")
					if dollarIdx == -1 {
						break
					}
					dollarIdx += pos
					if dollarIdx+1 < len(command) {
						nextChar := command[dollarIdx+1]
						if isAlphaNumericForValidation(byte(nextChar)) || nextChar == '_' {
							// This looks like $SECRET_NAME, check if it's followed by (
							if dollarIdx+2 < len(command) && command[dollarIdx+2] != '(' {
								hasSecretRef = true
								break
							}
						}
					}
					pos = dollarIdx + 1
				}
				if hasSecretRef {
					continue // Skip this pattern check
				}
			}
			return fmt.Errorf("command contains shell injection vector: %s", pattern)
		}
	}

	return nil
}

// parseCommandArgsForValidation parses command for validation purposes.
// It's a simplified version that doesn't handle escapes but is sufficient for validation.
func parseCommandArgsForValidation(command string) (string, []string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil, fmt.Errorf("empty command")
	}

	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for _, r := range command {
		switch r {
		case '\'':
			if inSingleQuote {
				inSingleQuote = false
			} else if !inDoubleQuote {
				inSingleQuote = true
			} else {
				current.WriteRune(r)
			}

		case '"':
			if inDoubleQuote {
				inDoubleQuote = false
			} else if !inSingleQuote {
				inDoubleQuote = true
			} else {
				current.WriteRune(r)
			}

		case ' ', '\t':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				continue
			}
			current.WriteRune(r)

		default:
			current.WriteRune(r)
		}
	}

	// Add the last argument
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return "", nil, fmt.Errorf("no command found")
	}

	return args[0], args[1:], nil
}

// isAlphaNumericForValidation checks if a byte is alphanumeric.
func isAlphaNumericForValidation(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}
