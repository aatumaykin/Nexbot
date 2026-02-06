package llm

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestMapChatRequest(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful"},
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there!"},
			{Role: RoleTool, Content: "result", ToolCallID: "call_123"},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   500,
	}

	zaiReq := p.mapChatRequest(req)

	if len(zaiReq.Messages) != 4 {
		t.Errorf("Messages len = %d, want 4", len(zaiReq.Messages))
	}

	if zaiReq.Model != "glm-4.7" {
		t.Errorf("Model = %q, want glm-4.7", zaiReq.Model)
	}

	if zaiReq.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", zaiReq.Temperature)
	}

	if zaiReq.MaxTokens != 500 {
		t.Errorf("MaxTokens = %d, want 500", zaiReq.MaxTokens)
	}

	// Check message mapping
	if zaiReq.Messages[0].Role != "system" {
		t.Errorf("First message role = %q, want system", zaiReq.Messages[0].Role)
	}

	if zaiReq.Messages[3].ToolCallID != "call_123" {
		t.Errorf("Tool message ToolCallID = %q, want call_123", zaiReq.Messages[3].ToolCallID)
	}
}

func TestMapChatRequest_WithTools(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Get weather"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters: map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	zaiReq := p.mapChatRequest(req)

	if len(zaiReq.Tools) != 1 {
		t.Errorf("Tools len = %d, want 1", len(zaiReq.Tools))
	}

	if zaiReq.Tools[0].Type != "function" {
		t.Errorf("Tool type = %q, want function", zaiReq.Tools[0].Type)
	}

	if zaiReq.Tools[0].Function["name"] != "get_weather" {
		t.Errorf("Tool function name = %q, want get_weather", zaiReq.Tools[0].Function["name"])
	}

	if zaiReq.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want auto", zaiReq.ToolChoice)
	}
}

func TestMapChatResponse(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	zaiResp := &zaiResponse{
		ID:      "resp-123",
		Model:   "glm-4.7",
		Created: 1234567890,
		Choices: []zaiChoice{
			{
				Index: 0,
				Message: zaiMessage{
					Role:             "assistant",
					Content:          "Hello!",
					ReasoningContent: "Reasoning here",
					ToolCalls: []zaiToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							}{
								Name:      "get_weather",
								Arguments: `{"city":"Tokyo"}`,
							},
						},
					},
				},
				FinishReason: "stop",
			},
		},
		Usage: zaiUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	resp := p.mapChatResponse(zaiResp)

	if resp.Content != "Hello!" {
		t.Errorf("Content = %q, want Hello!", resp.Content)
	}

	if resp.Model != "glm-4.7" {
		t.Errorf("Model = %q, want glm-4.7", resp.Model)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", resp.FinishReason)
	}

	if len(resp.ToolCalls) != 1 {
		t.Errorf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCall ID = %q, want call_1", resp.ToolCalls[0].ID)
	}

	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want get_weather", resp.ToolCalls[0].Name)
	}

	if resp.ToolCalls[0].Arguments != `{"city":"Tokyo"}` {
		t.Errorf("ToolCall Arguments = %q, want {\"city\":\"Tokyo\"}", resp.ToolCalls[0].Arguments)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Usage.PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("Usage.CompletionTokens = %d, want 20", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Usage.TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
}

func TestMapChatResponse_NoChoices(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	zaiResp := &zaiResponse{
		ID:      "resp-123",
		Model:   "glm-4.7",
		Choices: []zaiChoice{},
		Usage: zaiUsage{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
	}

	resp := p.mapChatResponse(zaiResp)

	if resp.Content != "" {
		t.Errorf("Content should be empty, got %q", resp.Content)
	}

	if resp.FinishReason != FinishReasonError {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonError)
	}

	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty, got %d", len(resp.ToolCalls))
	}

	if resp.Model != "glm-4.7" {
		t.Errorf("Model = %q, want glm-4.7", resp.Model)
	}
}

func TestMapChatResponse_UseReasoningContent(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	zaiResp := &zaiResponse{
		ID:    "resp-123",
		Model: "glm-4.7",
		Choices: []zaiChoice{
			{
				Index: 0,
				Message: zaiMessage{
					Role:             "assistant",
					Content:          "", // Empty content
					ReasoningContent: "This is the reasoning",
				},
				FinishReason: "stop",
			},
		},
		Usage: zaiUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	resp := p.mapChatResponse(zaiResp)

	if resp.Content != "This is the reasoning" {
		t.Errorf("Content should use reasoning_content, got %q", resp.Content)
	}
}
