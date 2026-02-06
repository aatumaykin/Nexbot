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
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	schemas := registry.ToSchema()
	if len(schemas) != 1 {
		t.Fatalf("Expected 1 schema, got %d", len(schemas))
	}

	schema := schemas[0]
	if schema.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", schema.Name)
	}

	if schema.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", schema.Description)
	}

	// Verify parameters
	props, ok := schema.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	if _, ok := props["input"]; !ok {
		t.Error("Expected input in properties")
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
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

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
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

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
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

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
	errChan := make(chan error, 100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			tool := &mockTool{
				name:        fmt.Sprintf("tool_%d", n),
				description: fmt.Sprintf("Tool %d", n),
				parameters:  map[string]interface{}{},
			}
			if err := registry.Register(tool); err != nil {
				errChan <- err
			} else {
				done <- true
			}
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 100; i++ {
		select {
		case <-done:
			continue
		case err := <-errChan:
			t.Fatalf("Failed to register tool: %v", err)
		}
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
