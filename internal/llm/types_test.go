package llm

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRole_Constants(t *testing.T) {
	tests := []struct {
		name  string
		role  Role
		value string
	}{
		{"RoleSystem", RoleSystem, "system"},
		{"RoleUser", RoleUser, "user"},
		{"RoleAssistant", RoleAssistant, "assistant"},
		{"RoleTool", RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.value {
				t.Errorf("%s = %q, want %q", tt.name, tt.role, tt.value)
			}
		})
	}
}

func TestFinishReason_Constants(t *testing.T) {
	tests := []struct {
		name   string
		reason FinishReason
		value  string
	}{
		{"FinishReasonStop", FinishReasonStop, "stop"},
		{"FinishReasonLength", FinishReasonLength, "length"},
		{"FinishReasonToolCalls", FinishReasonToolCalls, "tool_calls"},
		{"FinishReasonError", FinishReasonError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.reason) != tt.value {
				t.Errorf("%s = %q, want %q", tt.name, tt.reason, tt.value)
			}
		})
	}
}

func TestMockMode_Constants(t *testing.T) {
	tests := []struct {
		name string
		mode MockMode
		want int
	}{
		{"MockModeEcho", MockModeEcho, 0},
		{"MockModeFixed", MockModeFixed, 1},
		{"MockModeFixtures", MockModeFixtures, 2},
		{"MockModeError", MockModeError, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.mode) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.mode, tt.want)
			}
		})
	}
}

func TestMessage_JSONTags(t *testing.T) {
	msg := Message{
		Role:       RoleUser,
		Content:    "test",
		ToolCallID: "call_123",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}

	// Check JSON structure
	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Message: %v", err)
	}

	if unmarshaled["role"] != "user" {
		t.Errorf("role in JSON = %v, want user", unmarshaled["role"])
	}

	if unmarshaled["content"] != "test" {
		t.Errorf("content in JSON = %v, want test", unmarshaled["content"])
	}

	if unmarshaled["tool_call_id"] != "call_123" {
		t.Errorf("tool_call_id in JSON = %v, want call_123", unmarshaled["tool_call_id"])
	}
}

func TestChatRequest_JSONTags(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "test"},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []ToolDefinition{
			{
				Name:        "test_tool",
				Description: "Test tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal ChatRequest: %v", err)
	}

	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ChatRequest: %v", err)
	}

	if unmarshaled["model"] != "glm-4.7" {
		t.Errorf("model in JSON = %v, want glm-4.7", unmarshaled["model"])
	}

	if unmarshaled["temperature"] != 0.7 {
		t.Errorf("temperature in JSON = %v, want 0.7", unmarshaled["temperature"])
	}

	if unmarshaled["max_tokens"] != 100.0 {
		t.Errorf("max_tokens in JSON = %v, want 100.0", unmarshaled["max_tokens"])
	}
}

func TestChatResponse_JSONTags(t *testing.T) {
	resp := ChatResponse{
		Content:      "test response",
		FinishReason: FinishReasonStop,
		ToolCalls: []ToolCall{
			{
				ID:        "call_1",
				Name:      "test_tool",
				Arguments: `{"arg":"value"}`,
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Model: "glm-4.7",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ChatResponse: %v", err)
	}

	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ChatResponse: %v", err)
	}

	if unmarshaled["content"] != "test response" {
		t.Errorf("content in JSON = %v, want test response", unmarshaled["content"])
	}

	if unmarshaled["finish_reason"] != "stop" {
		t.Errorf("finish_reason in JSON = %v, want stop", unmarshaled["finish_reason"])
	}

	if unmarshaled["model"] != "glm-4.7" {
		t.Errorf("model in JSON = %v, want glm-4.7", unmarshaled["model"])
	}
}

func TestZAIConfig_JSONTags(t *testing.T) {
	cfg := ZAIConfig{
		APIKey:         "test-key",
		Model:          "glm-4.7",
		TimeoutSeconds: 30,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal ZAIConfig: %v", err)
	}

	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ZAIConfig: %v", err)
	}

	if unmarshaled["api_key"] != "test-key" {
		t.Errorf("api_key in JSON = %v, want test-key", unmarshaled["api_key"])
	}

	if unmarshaled["model"] != "glm-4.7" {
		t.Errorf("model in JSON = %v, want glm-4.7", unmarshaled["model"])
	}

	if unmarshaled["timeout_seconds"] != 30.0 {
		t.Errorf("timeout_seconds in JSON = %v, want 30.0", unmarshaled["timeout_seconds"])
	}
}

func TestMockConfig_DefaultValues(t *testing.T) {
	cfg := MockConfig{
		Mode:       MockModeEcho,
		Responses:  []string{"resp1", "resp2"},
		Delay:      100,
		ErrorAfter: 5,
	}

	if cfg.Mode != MockModeEcho {
		t.Errorf("Mode = %v, want MockModeEcho", cfg.Mode)
	}

	if len(cfg.Responses) != 2 {
		t.Errorf("Responses len = %d, want 2", len(cfg.Responses))
	}

	if cfg.Delay != 100 {
		t.Errorf("Delay = %d, want 100", cfg.Delay)
	}

	if cfg.ErrorAfter != 5 {
		t.Errorf("ErrorAfter = %d, want 5", cfg.ErrorAfter)
	}
}

func TestConstants(t *testing.T) {
	if ZAIEndpoint != "https://api.z.ai/api/coding/paas/v4/chat/completions" {
		t.Errorf("ZAIEndpoint = %q, want correct endpoint", ZAIEndpoint)
	}

	if ZAIRequestTimeout != 60*time.Second {
		t.Errorf("ZAIRequestTimeout = %v, want 60s", ZAIRequestTimeout)
	}

	if ZAIMaxRetries != 3 {
		t.Errorf("ZAIMaxRetries = %d, want 3", ZAIMaxRetries)
	}

	if ZAIRetryDelay != 1*time.Second {
		t.Errorf("ZAIRetryDelay = %v, want 1s", ZAIRetryDelay)
	}
}
