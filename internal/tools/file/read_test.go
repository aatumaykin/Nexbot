package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// testConfig creates a test configuration with default values.
func testConfig() *config.Config {
	return &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:       true,
				WhitelistDirs: []string{},
			},
		},
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestReadFileTool_Name(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", tool.Name())
	}
}

func TestReadFileTool_Description(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	// Description should mention file reading
	if !contains(desc, "file") {
		t.Errorf("Description should mention 'file', got: %s", desc)
	}
}

func TestReadFileTool_Parameters(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	// Check type
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	// Check properties
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Check required fields
	required, ok := params["required"].([]interface{})
	if !ok {
		// Try []string if []interface{} fails
		requiredStr, ok := params["required"].([]string)
		if !ok {
			t.Fatal("Required should be a slice")
		}
		if len(requiredStr) != 1 || requiredStr[0] != "path" {
			t.Errorf("Expected required to be ['path'], got %v", requiredStr)
		}
	} else {
		if len(required) != 1 || required[0] != "path" {
			t.Errorf("Expected required to be ['path'], got %v", required)
		}
	}

	// Check path property
	pathProp, ok := props["path"].(map[string]interface{})
	if !ok {
		t.Fatal("Path property should be a map")
	}

	if pathProp["type"] != "string" {
		t.Errorf("Expected path type 'string', got '%v'", pathProp["type"])
	}

	// Check optional parameters exist
	optionalParams := []string{"offset", "limit", "encoding"}
	for _, param := range optionalParams {
		if _, ok := props[param]; !ok {
			t.Errorf("Expected optional parameter '%s' in properties", param)
		}
	}
}

func TestReadFileTool_Execute_Success(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create tool
	tool := NewReadFileTool(ws, testConfig())

	// Execute
	args := `{"path": "test.txt"}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check result contains file content
	if !contains(result, "line1") || !contains(result, "line5") {
		t.Errorf("Expected result to contain file content, got: %s", result)
	}

	// Check result has line numbers
	if !contains(result, "000001|") || !contains(result, "000005|") {
		t.Errorf("Expected result to have line numbers, got: %s", result)
	}
}

func TestReadFileTool_Execute_WithOffset(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	// Read from line 2
	args := `{"path": "test.txt", "offset": 1}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not contain line 1
	if contains(result, "line1") {
		t.Error("Result should not contain line 1 when offset is 1")
	}

	// Should contain line 2
	if !contains(result, "line2") {
		t.Error("Result should contain line 2 when offset is 1")
	}
}

func TestReadFileTool_Execute_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	// Read only 2 lines
	args := `{"path": "test.txt", "limit": 2}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain line 1 and 2
	if !contains(result, "line1") || !contains(result, "line2") {
		t.Error("Result should contain lines 1 and 2")
	}

	// Should not contain line 3
	if contains(result, "line3") {
		t.Error("Result should not contain line 3 when limit is 2")
	}
}

func TestReadFileTool_Execute_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	tool := NewReadFileTool(ws, testConfig())

	args := `{"path": "nonexistent.txt"}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestReadFileTool_Execute_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	args := `{"path": "subdir"}`
	_, err = tool.Execute(args)
	if err == nil {
		t.Error("Expected error for directory path")
	}

	if !contains(err.Error(), "directory") {
		t.Errorf("Expected error to mention 'directory', got: %v", err)
	}
}

func TestReadFileTool_Execute_EscapeAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	tool := NewReadFileTool(ws, testConfig())

	// Try to escape workspace
	args := `{"path": "../etc/passwd"}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for path escape attempt")
	}

	if !contains(err.Error(), "escape") {
		t.Errorf("Expected error to mention 'escape', got: %v", err)
	}
}

func TestReadFileTool_Execute_MissingPath(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())

	args := `{}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for missing path")
	}

	if !contains(err.Error(), "required") {
		t.Errorf("Expected error to mention 'required', got: %v", err)
	}
}

func TestReadFileTool_Execute_UnsupportedEncoding(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	args := `{"path": "test.txt", "encoding": "iso-8859-1"}`
	_, err = tool.Execute(args)
	if err == nil {
		t.Error("Expected error for unsupported encoding")
	}

	if !contains(err.Error(), "unsupported") {
		t.Errorf("Expected error to mention 'unsupported', got: %v", err)
	}
}

func TestReadFileTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())

	args := `{invalid json}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !contains(err.Error(), "parse") {
		t.Errorf("Expected error to mention 'parse', got: %v", err)
	}
}

func TestReadFileTool_Execute_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "absolute path test"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(nil, testConfig())

	args := fmt.Sprintf(`{"path": "%s"}`, testFile)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, content) {
		t.Errorf("Expected result to contain '%s', got: %s", content, result)
	}
}

func TestReadFileTool_Execute_BeyondFileLength(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	// Try to read from beyond file length
	args := `{"path": "test.txt", "offset": 10}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "beyond file length") {
		t.Errorf("Expected result to mention 'beyond file length', got: %s", result)
	}
}

func TestReadFileTool_Execute_CRLF(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	// Windows-style line endings
	content := "line1\r\nline2\r\nline3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	args := `{"path": "test.txt"}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should handle CRLF correctly
	if !contains(result, "line1") || !contains(result, "line2") || !contains(result, "line3") {
		t.Errorf("Expected result to contain all lines, got: %s", result)
	}
}

func TestReadFileTool_Execute_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	args := `{"path": "test.txt"}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should handle empty files gracefully
	if !contains(result, "beyond file length") && !contains(result, "lines 0-0 of 0") {
		t.Errorf("Expected result to mention empty file, got: %s", result)
	}
}

func TestReadFileTool_Execute_LargeOffset(t *testing.T) {
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(ws, testConfig())

	// Negative offset should be treated as 0
	args := `{"path": "test.txt", "offset": -5}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should read from beginning
	if !contains(result, "line1") {
		t.Error("Expected result to start from line 1 when offset is negative")
	}
}

func TestReadFileTool_SchemaGeneration(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())

	schema := tool.Parameters()

	// Verify schema structure
	if schema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got '%v'", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema properties should be a map")
	}

	// Verify all required properties exist
	requiredProps := []string{"path", "offset", "limit", "encoding"}
	for _, prop := range requiredProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("Expected property '%s' in schema", prop)
		}
	}

	// Verify path property
	pathProp, ok := props["path"].(map[string]interface{})
	if !ok {
		t.Fatal("Path property should be a map")
	}

	if pathProp["type"] != "string" {
		t.Errorf("Expected path type 'string', got '%v'", pathProp["type"])
	}

	if _, ok := pathProp["description"]; !ok {
		t.Error("Path property should have description")
	}

	// Verify offset property
	offsetProp, ok := props["offset"].(map[string]interface{})
	if !ok {
		t.Fatal("Offset property should be a map")
	}

	if offsetProp["type"] != "integer" {
		t.Errorf("Expected offset type 'integer', got '%v'", offsetProp["type"])
	}

	if offsetProp["default"] != 0 {
		t.Errorf("Expected offset default 0, got '%v'", offsetProp["default"])
	}

	// Verify limit property
	limitProp, ok := props["limit"].(map[string]interface{})
	if !ok {
		t.Fatal("Limit property should be a map")
	}

	if limitProp["type"] != "integer" {
		t.Errorf("Expected limit type 'integer', got '%v'", limitProp["type"])
	}

	if limitProp["default"] != 2000 {
		t.Errorf("Expected limit default 2000, got '%v'", limitProp["default"])
	}

	// Verify encoding property
	encodingProp, ok := props["encoding"].(map[string]interface{})
	if !ok {
		t.Fatal("Encoding property should be a map")
	}

	if encodingProp["type"] != "string" {
		t.Errorf("Expected encoding type 'string', got '%v'", encodingProp["type"])
	}

	if encodingProp["default"] != "utf-8" {
		t.Errorf("Expected encoding default 'utf-8', got '%v'", encodingProp["default"])
	}

	// Verify required field
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Required should be a []string")
	}

	if len(required) != 1 || required[0] != "path" {
		t.Errorf("Expected required to be ['path'], got %v", required)
	}
}

func TestReadFileTool_SchemaToJSON(t *testing.T) {
	tool := NewReadFileTool(nil, testConfig())

	schema := tool.Parameters()

	// Convert to JSON to ensure it's serializable
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema to JSON: %v", err)
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal schema JSON: %v", err)
	}

	// Verify structure is preserved
	if unmarshaled["type"] != "object" {
		t.Error("Schema type should be preserved after JSON round-trip")
	}
}
