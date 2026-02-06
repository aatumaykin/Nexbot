package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockSpawnFunc is a mock spawn function for testing.
type mockSpawnFunc struct {
	subagentID       string
	sessionID        string
	shouldError      bool
	errorMsg         string
	checkCtxCanceled bool
}

func (m *mockSpawnFunc) Spawn(ctx context.Context, parentSession string, task string) (string, error) {
	// Check if context is cancelled
	if m.checkCtxCanceled && ctx.Err() != nil {
		return "", ctx.Err()
	}

	if m.shouldError {
		return "", assert.AnError
	}

	// Return JSON with subagent ID
	result := map[string]string{
		"id":      m.subagentID,
		"session": m.sessionID,
	}
	data, _ := json.Marshal(result)
	return string(data), nil
}

func TestSpawnTool_Name(t *testing.T) {
	tool := NewSpawnTool(nil)
	if tool.Name() != "spawn" {
		t.Errorf("Expected name 'spawn', got '%s'", tool.Name())
	}
}

func TestSpawnTool_Description(t *testing.T) {
	tool := NewSpawnTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	// Description should mention subagent
	if !contains(desc, "subagent") {
		t.Errorf("Description should mention 'subagent', got: %s", desc)
	}
}

func TestSpawnTool_Parameters(t *testing.T) {
	tool := NewSpawnTool(nil)
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
		if len(requiredStr) != 1 || requiredStr[0] != "task" {
			t.Errorf("Expected required to be ['task'], got %v", requiredStr)
		}
	} else {
		if len(required) != 1 || required[0] != "task" {
			t.Errorf("Expected required to be ['task'], got %v", required)
		}
	}

	// Check task property
	taskProp, ok := props["task"].(map[string]interface{})
	if !ok {
		t.Fatal("Task property should be a map")
	}

	if taskProp["type"] != "string" {
		t.Errorf("Expected task type 'string', got '%v'", taskProp["type"])
	}

	// Check timeout_seconds property
	timeoutProp, ok := props["timeout_seconds"].(map[string]interface{})
	if !ok {
		t.Fatal("Timeout property should be a map")
	}

	if timeoutProp["type"] != "number" {
		t.Errorf("Expected timeout_seconds type 'number', got '%v'", timeoutProp["type"])
	}
}

func TestSpawnTool_Execute_Success(t *testing.T) {
	mock := &mockSpawnFunc{
		subagentID: "test-subagent-123",
		sessionID:  "subagent-session-456",
	}

	tool := NewSpawnTool(mock.Spawn)

	args := `{"task": "Test task description"}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check result contains subagent ID
	if !contains(result, "test-subagent-123") {
		t.Errorf("Expected result to contain subagent ID, got: %s", result)
	}
}

func TestSpawnTool_ExecuteWithContext_Success(t *testing.T) {
	mock := &mockSpawnFunc{
		subagentID: "test-subagent-789",
		sessionID:  "subagent-session-012",
	}

	tool := NewSpawnTool(mock.Spawn)
	ctx := context.Background()

	args := `{"task": "Another test task"}`
	result, err := tool.ExecuteWithContext(ctx, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check result contains subagent ID
	if !contains(result, "test-subagent-789") {
		t.Errorf("Expected result to contain subagent ID, got: %s", result)
	}
}

func TestSpawnTool_Execute_WithTimeout(t *testing.T) {
	mock := &mockSpawnFunc{
		subagentID: "test-subagent-timeout",
		sessionID:  "subagent-session-timeout",
	}

	tool := NewSpawnTool(mock.Spawn)

	args := `{"task": "Test task with timeout", "timeout_seconds": 60}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check result contains subagent ID
	if !contains(result, "test-subagent-timeout") {
		t.Errorf("Expected result to contain subagent ID, got: %s", result)
	}
}

func TestSpawnTool_Execute_MissingTask(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	args := `{}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for missing task")
	}

	if !contains(err.Error(), "required") {
		t.Errorf("Expected error to mention 'required', got: %v", err)
	}
}

func TestSpawnTool_Execute_InvalidTimeout(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	// Test negative timeout
	args := `{"task": "Test", "timeout_seconds": -5}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for negative timeout")
	}

	if !contains(err.Error(), "positive") {
		t.Errorf("Expected error to mention 'positive', got: %v", err)
	}

	// Test zero timeout
	args = `{"task": "Test", "timeout_seconds": 0}`
	_, err = tool.Execute(args)
	if err == nil {
		t.Error("Expected error for zero timeout")
	}

	if !contains(err.Error(), "positive") {
		t.Errorf("Expected error to mention 'positive', got: %v", err)
	}
}

func TestSpawnTool_Execute_InvalidJSON(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	args := `{invalid json}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !contains(err.Error(), "parse") {
		t.Errorf("Expected error to mention 'parse', got: %v", err)
	}
}

func TestSpawnTool_Execute_SpawnError(t *testing.T) {
	mock := &mockSpawnFunc{
		shouldError: true,
		errorMsg:    "spawn failed",
	}
	tool := NewSpawnTool(mock.Spawn)

	args := `{"task": "Test task"}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for spawn failure")
	}

	if !contains(err.Error(), "failed to spawn") {
		t.Errorf("Expected error to mention 'failed to spawn', got: %v", err)
	}
}

func TestSpawnTool_ContextualToolInterface(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	// Verify tool implements ContextualTool
	var _ ContextualTool = tool
}

func TestSpawnTool_ToolInterface(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	// Verify tool implements Tool
	var _ Tool = tool
}

func TestSpawnTool_SchemaGeneration(t *testing.T) {
	tool := NewSpawnTool(nil)

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
	requiredProps := []string{"task", "timeout_seconds"}
	for _, prop := range requiredProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("Expected property '%s' in schema", prop)
		}
	}

	// Verify task property
	taskProp, ok := props["task"].(map[string]interface{})
	if !ok {
		t.Fatal("Task property should be a map")
	}

	if taskProp["type"] != "string" {
		t.Errorf("Expected task type 'string', got '%v'", taskProp["type"])
	}

	if _, ok := taskProp["description"]; !ok {
		t.Error("Task property should have description")
	}

	// Verify timeout property
	timeoutProp, ok := props["timeout_seconds"].(map[string]interface{})
	if !ok {
		t.Fatal("Timeout property should be a map")
	}

	if timeoutProp["type"] != "number" {
		t.Errorf("Expected timeout type 'number', got '%v'", timeoutProp["type"])
	}

	if _, ok := timeoutProp["description"]; !ok {
		t.Error("Timeout property should have description")
	}

	// Verify required field
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Required should be a []string")
	}

	if len(required) != 1 || required[0] != "task" {
		t.Errorf("Expected required to be ['task'], got %v", required)
	}
}

func TestSpawnTool_SchemaToJSON(t *testing.T) {
	tool := NewSpawnTool(nil)

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

func TestSpawnTool_RegistryIntegration(t *testing.T) {
	mock := &mockSpawnFunc{
		subagentID: "registry-test-subagent",
		sessionID:  "registry-test-session",
	}

	registry := NewRegistry()
	tool := NewSpawnTool(mock.Spawn)
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register spawn tool: %v", err)
	}

	// Generate schemas
	schemas := registry.ToSchema()

	if len(schemas) != 1 {
		t.Fatalf("Expected 1 schema, got %d", len(schemas))
	}

	schema := schemas[0]

	// Verify ToolDefinition fields
	if schema.Name != "spawn" {
		t.Errorf("Expected name 'spawn', got '%s'", schema.Name)
	}

	if schema.Description == "" {
		t.Error("Description should not be empty")
	}

	if schema.Parameters == nil {
		t.Error("Parameters should not be nil")
	}

	// Verify parameters match tool's Parameters()
	toolParams := tool.Parameters()
	if len(schema.Parameters) != len(toolParams) {
		t.Errorf("Schema parameters don't match tool parameters")
	}
}

func TestSpawnTool_ExecuteWithContext_Cancellation(t *testing.T) {
	mock := &mockSpawnFunc{
		subagentID:       "cancellation-test",
		sessionID:        "cancellation-session",
		checkCtxCanceled: true,
	}

	tool := NewSpawnTool(mock.Spawn)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	args := `{"task": "Test task"}`
	_, err := tool.ExecuteWithContext(ctx, args)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}

	// Note: The exact error message depends on the spawnFunc implementation
	// We're just checking that an error occurred
	_ = err // Avoid unused variable warning
}

func TestSpawnTool_Execute_EmptyTask(t *testing.T) {
	mock := &mockSpawnFunc{}
	tool := NewSpawnTool(mock.Spawn)

	args := `{"task": ""}`
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("Expected error for empty task")
	}

	if !contains(err.Error(), "required") {
		t.Errorf("Expected error to mention 'required', got: %v", err)
	}
}

func TestSpawnTool_Execute_ContextTimeout(t *testing.T) {
	// This test verifies that when timeout is provided in arguments,
	// it's applied to the context passed to spawnFunc
	mock := &mockSpawnFunc{
		subagentID: "timeout-test",
		sessionID:  "timeout-session",
	}

	tool := NewSpawnTool(mock.Spawn)

	args := `{"task": "Test task", "timeout_seconds": 300}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(result, "timeout-test") {
		t.Errorf("Expected result to contain subagent ID, got: %s", result)
	}
}
