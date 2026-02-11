package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

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

	props, ok := params["properties"].(map[string]any)
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
	pathProp, ok := props["path"].(map[string]any)
	if !ok {
		t.Fatal("Path property should be a map")
	}

	if pathProp["type"] != "string" {
		t.Errorf("Expected path type 'string', got '%v'", pathProp["type"])
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Fatal("Content property should be a map")
	}

	if contentProp["type"] != "string" {
		t.Errorf("Expected content type 'string', got '%v'", contentProp["type"])
	}

	// Check mode property
	modeProp, ok := props["mode"].(map[string]any)
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

func TestWriteFileTool_SkillPathValidation_InvalidLocation(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "tmp/SKILL.md", "content": "---\nname: test\n---\ncontent"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for skill file outside skills/ directory")
	}

	if err != nil && !contains(err.Error(), "skills/") {
		t.Errorf("Expected error to mention 'skills/', got: %v", err)
	}
}

func TestWriteFileTool_SkillPathValidation_InvalidFilename(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "skills/my_skill/my_file.txt", "content": "content"}`
	_, err := tool.Execute(args)

	if err != nil {
		t.Errorf("Expected non-SKILL.md file to succeed, got error: %v", err)
	}
}

func TestWriteFileTool_SkillPathValidation_ValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewWriteFileTool(ws, testConfig())

	args := `{"path": "skills/my_skill/SKILL.md", "content": "---\nname: test\n---\ncontent"}`
	_, err := tool.Execute(args)

	if err != nil {
		t.Errorf("Expected valid skill path to succeed, got error: %v", err)
	}

	filePath := filepath.Join(tmpDir, "skills", "my_skill", "SKILL.md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected skill file to be created")
	}
}

func TestWriteFileTool_SkillContentValidation_Enabled(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:              true,
				ValidateSkillContent: true,
			},
		},
	}

	tool := NewWriteFileTool(ws, cfg)

	args := `{"path": "skills/my_skill/SKILL.md", "content": "invalid content without frontmatter"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for skill content without YAML frontmatter")
	}

	if err != nil && !contains(err.Error(), "validation failed") {
		t.Errorf("Expected error to mention validation failure, got: %v", err)
	}
}

func TestWriteFileTool_SkillContentValidation_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:              true,
				ValidateSkillContent: false,
			},
		},
	}

	tool := NewWriteFileTool(ws, cfg)

	args := `{"path": "skills/my_skill/SKILL.md", "content": "invalid content without frontmatter"}`
	_, err := tool.Execute(args)

	if err != nil {
		t.Errorf("Expected validation disabled to succeed, got error: %v", err)
	}
}

func TestWriteFileTool_SkillContentValidation_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:              true,
				ValidateSkillContent: true,
			},
		},
	}

	tool := NewWriteFileTool(ws, cfg)

	content := `---
name: test-skill
description: A test skill
version: 1.0.0
---

Test content here`

	args := fmt.Sprintf(`{"path": "skills/my_skill/SKILL.md", "content": %s}`, jsonEscape(content))
	_, err := tool.Execute(args)

	if err != nil {
		t.Errorf("Expected valid skill content to succeed, got error: %v", err)
	}
}

func TestWriteFileTool_SkillContentValidation_MissingClosingDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:              true,
				ValidateSkillContent: true,
			},
		},
	}

	tool := NewWriteFileTool(ws, cfg)

	content := `---
name: test-skill
description: A test skill
Test content without closing delimiter`

	args := fmt.Sprintf(`{"path": "skills/my_skill/SKILL.md", "content": %s}`, jsonEscape(content))
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for skill content without closing YAML delimiter")
	}

	if err != nil && !contains(err.Error(), "validation failed") {
		t.Errorf("Expected error to mention validation failure, got: %v", err)
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
