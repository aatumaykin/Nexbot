package tools

import (
	"fmt"
	"testing"
)

// mockTool is a simple tool implementation for testing.
type mockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	executeFunc func(args string) (string, error)
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Parameters() map[string]interface{} {
	return m.parameters
}

func (m *mockTool) Execute(args string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(args)
	}
	return "mock result", nil
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	registry.Register(tool)

	retrieved, ok := registry.Get("test_tool")
	if !ok {
		t.Fatal("Tool not found after registration")
	}

	if retrieved.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", retrieved.Name())
	}
}

func TestRegistry_Register_Nil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering nil tool")
		}
	}()

	registry := NewRegistry()
	registry.Register(nil)
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering tool with empty name")
		}
	}()

	registry := NewRegistry()
	registry.Register(&mockTool{name: ""})
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "get_test",
		description: "Test get method",
		parameters:  map[string]interface{}{},
	}
	registry.Register(tool)

	// Test existing tool
	retrieved, ok := registry.Get("get_test")
	if !ok {
		t.Error("Expected to find existing tool")
	}
	if retrieved.Name() != "get_test" {
		t.Errorf("Expected name 'get_test', got '%s'", retrieved.Name())
	}

	// Test non-existing tool
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	tools := []*mockTool{
		{name: "tool1", description: "First tool", parameters: map[string]interface{}{}},
		{name: "tool2", description: "Second tool", parameters: map[string]interface{}{}},
		{name: "tool3", description: "Third tool", parameters: map[string]interface{}{}},
	}

	for _, tool := range tools {
		registry.Register(tool)
	}

	listed := registry.List()
	if len(listed) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(listed))
	}

	// Verify all tools are in the list
	names := make(map[string]bool)
	for _, tool := range listed {
		names[tool.Name()] = true
	}

	expectedNames := []string{"tool1", "tool2", "tool3"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected tool '%s' not found in list", name)
		}
	}
}

func TestRegistry_ToSchema(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "schema_tool",
		description: "Tool for schema testing",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
				"param2": map[string]interface{}{
					"type":        "integer",
					"description": "Second parameter",
				},
			},
			"required": []string{"param1"},
		},
	}
	registry.Register(tool)

	schemas := registry.ToSchema()
	if len(schemas) != 1 {
		t.Fatalf("Expected 1 schema, got %d", len(schemas))
	}

	schema := schemas[0]
	if schema.Name != "schema_tool" {
		t.Errorf("Expected name 'schema_tool', got '%s'", schema.Name)
	}

	if schema.Description != "Tool for schema testing" {
		t.Errorf("Expected description 'Tool for schema testing', got '%s'", schema.Description)
	}

	// Verify parameters
	props, ok := schema.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	if _, ok := props["param1"]; !ok {
		t.Error("Expected param1 in properties")
	}

	if _, ok := props["param2"]; !ok {
		t.Error("Expected param2 in properties")
	}

	required, ok := schema.Parameters["required"].([]interface{})
	if !ok {
		// Try string slice if interface slice fails
		requiredStr, ok := schema.Parameters["required"].([]string)
		if !ok || len(requiredStr) != 1 || requiredStr[0] != "param1" {
			t.Fatalf("Expected required to be ['param1'], got %v", schema.Parameters["required"])
		}
	} else {
		if len(required) != 1 || required[0] != "param1" {
			t.Errorf("Expected required to be ['param1'], got %v", required)
		}
	}
}

func TestExecuteToolCall(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "execute_test",
		description: "Tool for execute testing",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
		},
		executeFunc: func(args string) (string, error) {
			return "executed: " + args, nil
		},
	}
	registry.Register(tool)

	tc := ToolCall{
		ID:        "call_123",
		Name:      "execute_test",
		Arguments: `{"value": "test"}`,
	}

	result, err := ExecuteToolCall(registry, tc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ToolCallID != "call_123" {
		t.Errorf("Expected ToolCallID 'call_123', got '%s'", result.ToolCallID)
	}

	if result.Content != "executed: {\"value\": \"test\"}" {
		t.Errorf("Expected content 'executed: {\"value\": \"test\"}', got '%s'", result.Content)
	}

	if result.Error != "" {
		t.Errorf("Expected no error, got '%s'", result.Error)
	}
}

func TestExecuteToolCall_NotFound(t *testing.T) {
	registry := NewRegistry()

	tc := ToolCall{
		ID:        "call_123",
		Name:      "nonexistent",
		Arguments: "{}",
	}

	result, err := ExecuteToolCall(registry, tc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Error != "tool not found: nonexistent" {
		t.Errorf("Expected error 'tool not found: nonexistent', got '%s'", result.Error)
	}
}

func TestExecuteToolCall_ExecutionError(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "error_tool",
		description: "Tool that returns error",
		parameters:  map[string]interface{}{},
		executeFunc: func(args string) (string, error) {
			return "", fmt.Errorf("execution failed")
		},
	}
	registry.Register(tool)

	tc := ToolCall{
		ID:        "call_123",
		Name:      "error_tool",
		Arguments: "{}",
	}

	result, err := ExecuteToolCall(registry, tc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Error != "execution failed" {
		t.Errorf("Expected error 'execution failed', got '%s'", result.Error)
	}
}

func TestRegistry_ToJSON(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name:        "json_tool",
		description: "Tool for JSON testing",
		parameters: map[string]interface{}{
			"type": "object",
		},
	}
	registry.Register(tool)

	jsonStr, err := registry.ToJSON()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Basic JSON validation
	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it contains the tool name
	if !contains(jsonStr, "json_tool") {
		t.Error("Expected JSON to contain tool name 'json_tool'")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// Register tools concurrently
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			tool := &mockTool{
				name:        fmt.Sprintf("tool_%d", n),
				description: fmt.Sprintf("Tool %d", n),
				parameters:  map[string]interface{}{},
			}
			registry.Register(tool)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify all tools are registered
	listed := registry.List()
	if len(listed) != 100 {
		t.Errorf("Expected 100 tools, got %d", len(listed))
	}

	// Test concurrent reads
	for i := 0; i < 100; i++ {
		go func(n int) {
			name := fmt.Sprintf("tool_%d", n)
			_, ok := registry.Get(name)
			if !ok {
				t.Errorf("Tool %s not found", name)
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 100; i++ {
		<-done
	}
}

// Helper function for string matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
