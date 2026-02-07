package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

func TestDeleteFileTool_Name(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	if tool.Name() != "delete_file" {
		t.Errorf("Expected name 'delete_file', got '%s'", tool.Name())
	}
}

func TestDeleteFileTool_Description(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())
	desc := tool.Description()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if !contains(desc, "Delete") {
		t.Errorf("Description should mention 'Delete', got: %s", desc)
	}
}

func TestDeleteFileTool_Parameters(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())
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
}

func TestDeleteFileTool_Execute_File(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Delete the file
	args := `{"path": "test.txt"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted")
	}
}

func TestDeleteFileTool_Execute_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Create an empty directory
	subDir := filepath.Join(tmpDir, "emptydir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Delete the directory
	args := `{"path": "emptydir"}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify directory was deleted
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Error("Expected directory to be deleted")
	}
}

func TestDeleteFileTool_Execute_NonEmptyDirectory_Recursive(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Create a non-empty directory
	subDir := filepath.Join(tmpDir, "nonemptydir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Delete the directory recursively
	args := `{"path": "nonemptydir", "recursive": true}`
	result, err := tool.Execute(args)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify directory was deleted
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Error("Expected directory to be deleted")
	}
}

func TestDeleteFileTool_Execute_NonEmptyDirectory_NonRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Create a non-empty directory
	subDir := filepath.Join(tmpDir, "nonemptydir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to delete without recursive flag (should fail)
	args := `{"path": "nonemptydir", "recursive": false}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error when trying to delete non-empty directory without recursive flag")
	}

	if !contains(err.Error(), "not empty") {
		t.Errorf("Expected error to mention 'not empty', got: %v", err)
	}
}

func TestDeleteFileTool_Execute_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Try to delete non-existent file
	args := `{"path": "nonexistent.txt"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestDeleteFileTool_Execute_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Try to delete with missing path
	args := `{}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for missing path")
	}

	if !contains(err.Error(), "required") {
		t.Errorf("Expected error to mention 'required', got: %v", err)
	}
}

func TestDeleteFileTool_Execute_DirectoryTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	tool := NewDeleteFileTool(ws, testConfig())

	// Try to escape workspace using directory traversal
	args := `{"path": "../etc/passwd"}`
	_, err := tool.Execute(args)

	if err == nil {
		t.Error("Expected error for directory traversal attempt")
	}

	// Workspace.ResolvePath returns error for path escape attempts
	if !contains(err.Error(), "escape") && !contains(err.Error(), "escape workspace") {
		t.Errorf("Expected error to mention 'escape', got: %v", err)
	}
}
