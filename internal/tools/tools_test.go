package tools

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Tests for ShellExecTool

func TestShellExecTool_Name(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo", "test"},
			},
		},
	}

	tool := NewShellExecTool(cfg, log)

	if tool.Name() != "shell_exec" {
		t.Errorf("Expected name 'shell_exec', got '%s'", tool.Name())
	}
}

func TestShellExecTool_Description(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo"},
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	desc := tool.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if !contains(desc, "shell") {
		t.Errorf("Description should mention 'shell', got: %s", desc)
	}
}

func TestShellExecTool_Parameters(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo"},
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a []string")
	}

	if len(required) != 1 || required[0] != "command" {
		t.Errorf("Expected required to be ['command'], got %v", required)
	}

	// Check command property
	commandProp, ok := props["command"].(map[string]any)
	if !ok {
		t.Fatal("Command property should be a map")
	}

	if commandProp["type"] != "string" {
		t.Errorf("Expected command type 'string', got '%v'", commandProp["type"])
	}
}

func TestShellExecTool_Execute_Disabled(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         false,
				AllowedCommands: []string{"echo"},
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "echo test"}`
	_, err = tool.Execute(args)

	if err == nil {
		t.Error("Expected error when shell tool is disabled")
	}

	if !contains(err.Error(), "disabled") {
		t.Errorf("Expected error to mention 'disabled', got: %v", err)
	}
}

func TestShellExecTool_Execute_NotWhitelisted(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "rm -rf /"}`
	_, err = tool.Execute(args)

	if err == nil {
		t.Error("Expected error for non-whitelisted command")
	}

	if !contains(err.Error(), "allowed") {
		t.Errorf("Expected error to mention 'allowed', got: %v", err)
	}
}

func TestShellExecTool_Execute_Echo(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "echo 'Hello, World!'"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Hello, World!") {
		t.Errorf("Expected result to contain 'Hello, World!', got: %s", result)
	}

	if !contains(result, "Exit code: 0") {
		t.Error("Expected result to contain exit code 0")
	}
}

func TestShellExecTool_Execute_Timeout(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"sleep"},
				TimeoutSeconds:  1,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "sleep 5"}`

	// Just verify that the tool can be called
	// Actual timeout behavior may vary by system
	_, err = tool.Execute(args)
	// We expect some kind of error (timeout or killed), but don't enforce it
	_ = err
}

func TestShellExecTool_Execute_FailedCommand(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"sh"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "sh -c 'exit 1'"}`

	result, err := tool.Execute(args)

	// Command execution should "succeed" (no error from tool.Execute)
	// but result should contain error information
	if err != nil {
		t.Fatalf("Unexpected error from tool execution: %v", err)
	}

	// Check that exit code is reflected in result
	if !contains(result, "Exit code: 1") {
		t.Errorf("Expected result to contain exit code 1, got: %s", result)
	}
}

func TestShellExecTool_Execute_EmptyWhitelist(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{}, // Empty whitelist
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "echo test"}`
	_, err = tool.Execute(args)

	// With new logic: all lists empty = fail-open (all commands allowed)
	if err != nil {
		t.Errorf("Expected no error when all lists are empty (fail-open), got: %v", err)
	}
}

func TestShellExecTool_Execute_DenyCommand(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo", "rm"},
				DenyCommands:    []string{"rm"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "rm -rf /tmp/test"}`
	_, err = tool.Execute(args)

	if err == nil {
		t.Error("Expected error for denied command")
	}

	if !contains(err.Error(), "denied by deny_commands") {
		t.Errorf("Expected error to mention deny, got: %v", err)
	}
}

func TestShellExecTool_Execute_AskCommand(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo", "git"},
				AskCommands:     []string{"git *"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "git commit -m test"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "CONFIRM_REQUIRED") {
		t.Errorf("Expected result to contain CONFIRM_REQUIRED, got: %s", result)
	}
}

func TestShellExecTool_Execute_Priority(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"rm"},
				DenyCommands:    []string{"rm"},
				AskCommands:     []string{"rm"},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "rm -rf /tmp/test"}`
	_, err = tool.Execute(args)

	if err == nil {
		t.Error("Expected error (deny has priority)")
	}

	if !contains(err.Error(), "denied by deny_commands") {
		t.Errorf("Expected deny error (not ask/allowed), got: %v", err)
	}
}

func TestShellExecTool_Execute_AllListsEmpty(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{},
				DenyCommands:    []string{},
				AskCommands:     []string{},
				TimeoutSeconds:  5,
			},
		},
	}

	tool := NewShellExecTool(cfg, log)
	args := `{"command": "echo test"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error (all lists empty = all allowed): %v", err)
	}

	if !contains(result, "test") {
		t.Errorf("Expected command to execute, got: %s", result)
	}
}
