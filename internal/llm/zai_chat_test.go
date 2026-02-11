package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aatumaykin/nexbot/internal/logger"
)

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
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Test response" {
		t.Errorf("Content = %q, want Test response", resp.Content)
	}

	if resp.Model != "glm-4.7" {
		t.Errorf("Model = %q, want glm-4.7", resp.Model)
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
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	resp, err := p.Chat(ctx, req)
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
