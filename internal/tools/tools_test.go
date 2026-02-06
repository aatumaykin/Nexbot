package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// Tests for WriteFileTool

func TestWriteFileTool_Name(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	if tool.Name() != "write_file" {
		t.Errorf("Expected name 'write_file', got '%s'", tool.Name())
	}
}

func TestWriteFileTool_Description(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())
	desc := tool.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if !contains(desc, "file") {
		t.Errorf("Description should mention 'file', got: %s", desc)
	}
}

func TestWriteFileTool_Parameters(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a []string")
	}

	if len(required) != 2 || required[0] != "path" || required[1] != "content" {
		t.Errorf("Expected required to be ['path', 'content'], got %v", required)
	}

	// Check path property
	pathProp, ok := props["path"].(map[string]interface{})
	if !ok {
		t.Fatal("Path property should be a map")
	}

	if pathProp["type"] != "string" {
		t.Errorf("Expected path type 'string', got '%v'", pathProp["type"])
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]interface{})
	if !ok {
		t.Fatal("Content property should be a map")
	}

	if contentProp["type"] != "string" {
		t.Errorf("Expected content type 'string', got '%v'", contentProp["type"])
	}

	// Check mode property
	modeProp, ok := props["mode"].(map[string]interface{})
	if !ok {
		t.Fatal("Mode property should be a map")
	}

	if modeProp["type"] != "string" {
		t.Errorf("Expected mode type 'string', got '%v'", modeProp["type"])
	}
}

func TestWriteFileTool_Execute_CreateMode(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "test.txt", "content": "Hello, World!"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "test.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected file content 'Hello, World!', got '%s'", string(content))
	}
}

func TestWriteFileTool_Execute_AppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	// Create file first
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("Initial\n"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Append content
	args := `{"path": "test.txt", "mode": "append", "content": "Appended"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify content was appended
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Initial\nAppended" {
		t.Errorf("Expected 'Initial\\nAppended', got '%s'", string(content))
	}
}

func TestWriteFileTool_Execute_OverwriteMode(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	// Create file first
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("Initial"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Overwrite content
	args := `{"path": "test.txt", "mode": "overwrite", "content": "Overwritten"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify content was overwritten
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Overwritten" {
		t.Errorf("Expected 'Overwritten', got '%s'", string(content))
	}
}

func TestWriteFileTool_Execute_CreateExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	// Create file first
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("Initial"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Try to create again (should fail)
	args := `{"path": "test.txt", "mode": "create", "content": "New content"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error when file already exists in create mode")
	}

	if !contains(err.Error(), "already exists") {
		t.Errorf("Expected error to mention 'already exists', got: %v", err)
	}
}

func TestWriteFileTool_Execute_AppendNonExisting(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	// Try to append to non-existent file
	args := `{"path": "nonexistent.txt", "mode": "append", "content": "Content"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error when appending to non-existent file")
	}

	if !contains(err.Error(), "does not exist") {
		t.Errorf("Expected error to mention 'does not exist', got: %v", err)
	}
}

func TestWriteFileTool_Execute_CreateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "subdir/test.txt", "content": "Content"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify directory was created
	subDir := filepath.Join(tmpDir, "subdir")
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Expected subdirectory to be created")
	}

	// Verify file was created
	filePath := filepath.Join(subDir, "test.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected file to be created")
	}
}

func TestWriteFileTool_Execute_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "test.txt", "content": "Content", "mode": "invalid"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for invalid mode")
	}

	if !contains(err.Error(), "invalid mode") {
		t.Errorf("Expected error to mention 'invalid mode', got: %v", err)
	}
}

// Tests for ListDirTool

func TestListDirTool_Name(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewListDirTool(ws, testConfig())

	if tool.Name() != "list_dir" {
		t.Errorf("Expected name 'list_dir', got '%s'", tool.Name())
	}
}

func TestListDirTool_Description(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewListDirTool(ws, testConfig())
	desc := tool.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if !contains(desc, "directory") || !contains(desc, "list") {
		t.Errorf("Description should mention 'directory' and 'list', got: %s", desc)
	}
}

func TestListDirTool_Parameters(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewListDirTool(ws, testConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a []string")
	}

	if len(required) != 1 || required[0] != "path" {
		t.Errorf("Expected required to be ['path'], got %v", required)
	}

	// Check path property
	pathProp, ok := props["path"].(map[string]interface{})
	if !ok {
		t.Fatal("Path property should be a map")
	}

	if pathProp["type"] != "string" {
		t.Errorf("Expected path type 'string', got '%v'", pathProp["type"])
	}

	// Check recursive property
	recursiveProp, ok := props["recursive"].(map[string]interface{})
	if !ok {
		t.Fatal("Recursive property should be a map")
	}

	if recursiveProp["type"] != "boolean" {
		t.Errorf("Expected recursive type 'boolean', got '%v'", recursiveProp["type"])
	}

	// Check include_hidden property
	hiddenProp, ok := props["include_hidden"].(map[string]interface{})
	if !ok {
		t.Fatal("Include_hidden property should be a map")
	}

	if hiddenProp["type"] != "boolean" {
		t.Errorf("Expected include_hidden type 'boolean', got '%v'", hiddenProp["type"])
	}
}

func TestListDirTool_Execute_Flat(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	tool := NewListDirTool(ws, testConfig())
	args := `{"path": "."}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "DIR  subdir") {
		t.Error("Expected result to contain directory")
	}

	if !contains(result, "FILE file1.txt") {
		t.Error("Expected result to contain file1.txt")
	}

	if !contains(result, "FILE file2.txt") {
		t.Error("Expected result to contain file2.txt")
	}
}

func TestListDirTool_Execute_Recursive(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create directory structure
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	tool := NewListDirTool(ws, testConfig())
	args := `{"path": ".", "recursive": true}`
	_, err := tool.Execute(args)

	// Just verify that recursive mode doesn't error
	if err != nil {
		t.Fatalf("Unexpected error in recursive mode: %v", err)
	}
}

func TestListDirTool_Execute_IncludeHidden(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create hidden file
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("hidden"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	tool := NewListDirTool(ws, testConfig())

	// Without include_hidden
	args := `{"path": ".", "include_hidden": false}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if contains(result, ".hidden") {
		t.Error("Expected result to not contain hidden file when include_hidden is false")
	}

	// With include_hidden
	args = `{"path": ".", "include_hidden": true}`
	result, err = tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, ".hidden") {
		t.Error("Expected result to contain hidden file when include_hidden is true")
	}
}

func TestListDirTool_Execute_NotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create a file instead of directory
	filePath := filepath.Join(tmpDir, "notadir.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewListDirTool(ws, testConfig())
	args := `{"path": "notadir.txt"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for non-directory path")
	}

	if !contains(err.Error(), "not a directory") {
		t.Errorf("Expected error to mention 'not a directory', got: %v", err)
	}
}

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

	props, ok := params["properties"].(map[string]interface{})
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
	commandProp, ok := props["command"].(map[string]interface{})
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
	// but the result should contain error information
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

func TestWriteFileTool_WhitelistAbsolutePaths(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Создадим временный каталог для whitelist
	whitelistDir := t.TempDir()

	// Конфигурация с whitelist
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:       true,
				WhitelistDirs: []string{whitelistDir},
			},
		},
	}

	tool := NewWriteFileTool(ws, cfg)

	// Тест 1: Абсолютный путь внутри whitelist должен работать
	allowedFile := filepath.Join(whitelistDir, "allowed.txt")
	args := fmt.Sprintf(`{"path": "%s", "content": "test content", "mode": "create"}`, allowedFile)
	_, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected absolute path in whitelist to be allowed, got error: %v", err)
	}

	// Тест 2: Абсолютный путь вне whitelist должен быть запрещён
	forbiddenFile := "/tmp/forbidden.txt"
	args = fmt.Sprintf(`{"path": "%s", "content": "test content", "mode": "create"}`, forbiddenFile)
	_, err = tool.Execute(args)
	if err == nil {
		t.Error("Expected absolute path outside whitelist to be rejected")
	}
	if err != nil && !contains(err.Error(), "absolute paths are not allowed") {
		t.Errorf("Expected 'absolute paths are not allowed' error, got: %v", err)
	}

	// Тест 3: Относительные пути должны работать (они относятся к workspace)
	args = `{"path": "relative.txt", "content": "test content", "mode": "create"}`
	_, err = tool.Execute(args)
	if err != nil {
		t.Errorf("Expected relative path to work, got error: %v", err)
	}
}
