package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

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
