package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// ============================================================================
// Tests for MockProvider
// ============================================================================

func TestNewMockProvider(t *testing.T) {
	tests := []struct {
		name  string
		cfg   MockConfig
		want  MockMode
		repos []string
	}{
		{
			name: "echo mode",
			cfg:  MockConfig{Mode: MockModeEcho},
			want: MockModeEcho,
		},
		{
			name: "fixed mode",
			cfg: MockConfig{
				Mode:      MockModeFixed,
				Responses: []string{"test response"},
			},
			want:  MockModeFixed,
			repos: []string{"test response"},
		},
		{
			name: "fixtures mode",
			cfg: MockConfig{
				Mode:      MockModeFixtures,
				Responses: []string{"resp1", "resp2"},
			},
			want:  MockModeFixtures,
			repos: []string{"resp1", "resp2"},
		},
		{
			name: "error mode",
			cfg:  MockConfig{Mode: MockModeError},
			want: MockModeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMockProvider(tt.cfg)
			if p.mode != tt.want {
				t.Errorf("NewMockProvider() mode = %v, want %v", p.mode, tt.want)
			}
			if tt.repos != nil {
				if len(p.responses) != len(tt.repos) {
					t.Errorf("NewMockProvider() responses len = %d, want %d", len(p.responses), len(tt.repos))
				}
			}
		})
	}
}

func TestNewEchoProvider(t *testing.T) {
	p := NewEchoProvider()
	if p.mode != MockModeEcho {
		t.Errorf("NewEchoProvider() mode = %v, want %v", p.mode, MockModeEcho)
	}
}

func TestNewFixedProvider(t *testing.T) {
	response := "fixed response"
	p := NewFixedProvider(response)

	if p.mode != MockModeFixed {
		t.Errorf("NewFixedProvider() mode = %v, want %v", p.mode, MockModeFixed)
	}
	if len(p.responses) != 1 || p.responses[0] != response {
		t.Errorf("NewFixedProvider() responses = %v, want [%s]", p.responses, response)
	}
}

func TestNewFixturesProvider(t *testing.T) {
	responses := []string{"resp1", "resp2", "resp3"}
	p := NewFixturesProvider(responses)

	if p.mode != MockModeFixtures {
		t.Errorf("NewFixturesProvider() mode = %v, want %v", p.mode, MockModeFixtures)
	}
	if len(p.responses) != len(responses) {
		t.Errorf("NewFixturesProvider() responses len = %d, want %d", len(p.responses), len(responses))
	}
}

func TestNewErrorProvider(t *testing.T) {
	p := NewErrorProvider()
	if p.mode != MockModeError {
		t.Errorf("NewErrorProvider() mode = %v, want %v", p.mode, MockModeError)
	}
}

func TestMockProvider_Chat_EchoMode(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Model: "test-model",
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Echo: Hello" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Echo: Hello")
	}

	if resp.Model != "test-model" {
		t.Errorf("Chat() model = %q, want %q", resp.Model, "test-model")
	}

	if resp.FinishReason != "stop" {
		t.Errorf("Chat() finishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestMockProvider_Chat_FixedMode(t *testing.T) {
	p := NewFixedProvider("Always the same")
	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Test1"},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Always the same" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Always the same")
	}

	// Test multiple requests
	resp, err = p.Chat(ctx, ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Test2"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Always the same" {
		t.Errorf("Chat() second call content = %q, want %q", resp.Content, "Always the same")
	}
}

func TestMockProvider_Chat_FixturesMode(t *testing.T) {
	responses := []string{"resp1", "resp2", "resp3"}
	p := NewFixturesProvider(responses)
	ctx := context.Background()

	for i, wantResp := range responses {
		req := ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "Test"}},
		}
		resp, err := p.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		if err != nil {
			t.Fatalf("Chat() iteration %d error = %v", i, err)
		}

		if resp.Content != wantResp {
			t.Errorf("Chat() iteration %d content = %q, want %q", i, resp.Content, wantResp)
		}
	}

	// Test rotation back to first response
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() rotation error = %v", err)
	}

	if resp.Content != "resp1" {
		t.Errorf("Chat() after rotation content = %q, want %q", resp.Content, "resp1")
	}
}

func TestMockProvider_Chat_ErrorMode(t *testing.T) {
	p := NewErrorProvider()
	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}

	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "mock provider error") {
		t.Errorf("Chat() error = %v, want 'mock provider error'", err)
	}
}

func TestMockProvider_Chat_ErrorAfter(t *testing.T) {
	p := NewMockProvider(MockConfig{
		Mode:       MockModeEcho,
		ErrorAfter: 2,
	})
	ctx := context.Background()

	// First two calls should succeed
	for i := 0; i < 2; i++ {
		req := ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "Test"}},
		}
		_, err := p.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		if err != nil {
			t.Fatalf("Chat() call %d expected success, got error: %v", i+1, err)
		}
	}

	// Third call should fail
	_, err := p.Chat(ctx, ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	})
	if err == nil {
		t.Error("Chat() expected error after 2 calls, got nil")
	}
}

func TestMockProvider_Chat_NoUserMessage(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful"},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Echo: (no user message)" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Echo: (no user message)")
	}
}

func TestMockProvider_Chat_EmptyMessages(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Echo: (no user message)" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Echo: (no user message)")
	}
}

func TestMockProvider_SupportsToolCalling(t *testing.T) {
	p := NewEchoProvider()
	if p.SupportsToolCalling() != false {
		t.Errorf("SupportsToolCalling() = %v, want false", p.SupportsToolCalling())
	}
}

func TestMockProvider_GetCallCount(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	if p.GetCallCount() != 0 {
		t.Errorf("GetCallCount() = %d, want 0", p.GetCallCount())
	}

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if p.GetCallCount() != 1 {
		t.Errorf("GetCallCount() after 1 call = %d, want 1", p.GetCallCount())
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if p.GetCallCount() != 2 {
		t.Errorf("GetCallCount() after 2 calls = %d, want 2", p.GetCallCount())
	}
}

func TestMockProvider_ResetCallCount(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	p.ResetCallCount()

	if p.GetCallCount() != 0 {
		t.Errorf("After ResetCallCount(), GetCallCount() = %d, want 0", p.GetCallCount())
	}
}

func TestMockProvider_SetErrorAfter(t *testing.T) {
	p := NewMockProvider(MockConfig{
		Mode:       MockModeEcho,
		ErrorAfter: 0,
	})
	ctx := context.Background()

	p.SetErrorAfter(1)

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Test"}},
	}
	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Errorf("First call with ErrorAfter=1 should succeed, got error: %v", err)
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Second call with ErrorAfter=1 should fail, got nil")
	}
}

func TestMockProvider_GetResponses(t *testing.T) {
	responses := []string{"resp1", "resp2"}
	p := NewFixturesProvider(responses)

	got := p.GetResponses()
	if len(got) != len(responses) {
		t.Errorf("GetResponses() len = %d, want %d", len(got), len(responses))
	}

	for i, r := range got {
		if r != responses[i] {
			t.Errorf("GetResponses()[%d] = %q, want %q", i, r, responses[i])
		}
	}
}

func TestMockProvider_SetResponses(t *testing.T) {
	p := NewEchoProvider()

	responses := []string{"new1", "new2"}
	p.SetResponses(responses)

	got := p.GetResponses()
	if len(got) != len(responses) {
		t.Errorf("After SetResponses(), len = %d, want %d", len(got), len(responses))
	}

	// Response index should be reset
	if p.responseIndex != 0 {
		t.Errorf("After SetResponses(), responseIndex = %d, want 0", p.responseIndex)
	}

	// Verify responses are actually set
	for i, r := range got {
		if r != responses[i] {
			t.Errorf("After SetResponses(), response[%d] = %q, want %q", i, r, responses[i])
		}
	}
}

func TestMockProvider_Chat_UsageTracking(t *testing.T) {
	p := NewEchoProvider()
	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello, world!"},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Usage.PromptTokens != len("Hello, world!") {
		t.Errorf("Usage.PromptTokens = %d, want %d", resp.Usage.PromptTokens, len("Hello, world!"))
	}

	expectedCompletionTokens := len("Echo: Hello, world!")
	if resp.Usage.CompletionTokens != expectedCompletionTokens {
		t.Errorf("Usage.CompletionTokens = %d, want %d", resp.Usage.CompletionTokens, expectedCompletionTokens)
	}

	expectedTotal := len("Hello, world!") + expectedCompletionTokens
	if resp.Usage.TotalTokens != expectedTotal {
		t.Errorf("Usage.TotalTokens = %d, want %d", resp.Usage.TotalTokens, expectedTotal)
	}
}

// ============================================================================
// Tests for ZAIProvider
// ============================================================================

func TestNewZAIProvider(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := ZAIConfig{
		APIKey:         "test-key",
		Model:          "glm-4.7",
		TimeoutSeconds: 30,
	}

	p := NewZAIProvider(cfg, log)

	if p == nil {
		t.Fatal("NewZAIProvider() returned nil")
	}

	if p.config.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", p.config.APIKey, "test-key")
	}

	if p.config.Model != "glm-4.7" {
		t.Errorf("Model = %q, want %q", p.config.Model, "glm-4.7")
	}

	if p.client.Timeout != 30*time.Second {
		t.Errorf("client.Timeout = %v, want 30s", p.client.Timeout)
	}

	if p.apiURL != ZAIEndpoint {
		t.Errorf("apiURL = %q, want %q", p.apiURL, ZAIEndpoint)
	}
}

func TestNewZAIProvider_Defaults(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := ZAIConfig{
		APIKey: "test-key",
		// Model not set
		// TimeoutSeconds not set
	}

	p := NewZAIProvider(cfg, log)

	if p.config.Model != "glm-4.7" {
		t.Errorf("Default Model = %q, want %q", p.config.Model, "glm-4.7")
	}

	if p.client.Timeout != ZAIRequestTimeout {
		t.Errorf("Default Timeout = %v, want %v", p.client.Timeout, ZAIRequestTimeout)
	}
}

func TestTruncateResponse(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		maxLen int
		want   string
	}{
		{
			name:   "short body",
			body:   []byte("short"),
			maxLen: 100,
			want:   "short",
		},
		{
			name:   "exact length",
			body:   []byte("exact"),
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "long body",
			body:   []byte("this is a long string"),
			maxLen: 10,
			want:   "this is a ...",
		},
		{
			name:   "empty body",
			body:   []byte(""),
			maxLen: 100,
			want:   "",
		},
		{
			name:   "zero max len",
			body:   []byte("test"),
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateResponse(tt.body, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestZAIHTTPError_Error(t *testing.T) {
	err := &zaiHTTPError{
		StatusCode: 404,
		Body:       `{"error": "not found"}`,
	}

	got := err.Error()
	want := "HTTP error: status=404, body={\"error\": \"not found\"}"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

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

func TestZAIProvider_SupportsToolCalling(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test"}, log)

	if !p.SupportsToolCalling() {
		t.Error("ZAIProvider should support tool calling")
	}
}

func TestZAIProvider_Chat_Success(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		resp := zaiResponse{
			ID:    "test-123",
			Model: "glm-4.7",
			Choices: []zaiChoice{
				{
					Index: 0,
					Message: zaiMessage{
						Role:    "assistant",
						Content: "Test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: zaiUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Test response" {
		t.Errorf("Content = %q, want Test response", resp.Content)
	}

	if resp.Model != "glm-4.7" {
		t.Errorf("Model = %q, want glm-4.7", resp.Model)
	}
}

func TestZAIProvider_Chat_Timeout(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{
		APIKey:         "test-key",
		TimeoutSeconds: 0, // Very short timeout
	}, log)
	p.apiURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected timeout error, got nil")
	}
}

func TestZAIProvider_Chat_NetworkError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = "http://invalid-host-that-does-not-exist.local:9999"

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected network error, got nil")
	}
}

func TestZAIProvider_Chat_HTTPError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error": "invalid api key"}`)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "invalid-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected HTTP error, got nil")
	}

	var httpErr *zaiHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("HTTP error status = %d, want 401", httpErr.StatusCode)
		}
	}
}

func TestZAIProvider_Chat_APIError(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := zaiResponse{
			Error: &zaiAPIError{
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
				Code:    "rate_limit",
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected API error, got nil")
	}

	if !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Errorf("Error = %v, should contain 'Rate limit exceeded'", err)
	}
}

func TestZAIProvider_Chat_InvalidJSON(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`invalid json`)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	_, err = p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err == nil {
		t.Error("Chat() expected JSON error, got nil")
	}
}

func TestZAIProvider_Chat_ToolCalls(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := zaiResponse{
			ID:    "test-123",
			Model: "glm-4.7",
			Choices: []zaiChoice{
				{
					Index: 0,
					Message: zaiMessage{
						Role:    "assistant",
						Content: "",
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
							{
								ID:   "call_2",
								Type: "function",
								Function: struct {
									Name      string `json:"name"`
									Arguments string `json:"arguments"`
								}{
									Name:      "get_time",
									Arguments: `{}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: zaiUsage{
				PromptTokens:     20,
				CompletionTokens: 10,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
	}))
	defer server.Close()

	p := NewZAIProvider(ZAIConfig{APIKey: "test-key"}, log)
	p.apiURL = server.URL

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "What's the weather in Tokyo?"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Errorf("ToolCalls len = %d, want 2", len(resp.ToolCalls))
	}

	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}

	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("First tool name = %q, want get_weather", resp.ToolCalls[0].Name)
	}
}

// ============================================================================
// Tests for Provider types and constants
// ============================================================================

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
	var unmarshaled map[string]interface{}
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
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal ChatRequest: %v", err)
	}

	var unmarshaled map[string]interface{}
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

	var unmarshaled map[string]interface{}
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

	var unmarshaled map[string]interface{}
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
