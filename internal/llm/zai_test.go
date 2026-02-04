package llm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestZAIClient_SendRequest is an integration test for the Z.ai API client.
// It sends a real request to Z.ai API and validates the response.
//
// To run this test, set the ZAI_API_KEY environment variable:
//
//	ZAI_API_KEY=your_api_key go test -v -run TestZAIClient_SendRequest ./internal/llm
//
// If ZAI_API_KEY is not set, the test will be skipped.
func TestZAIClient_SendRequest(t *testing.T) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: ZAI_API_KEY environment variable not set")
	}

	// Create logger for testing
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create ZAI provider
	provider := NewZAIProvider(ZAIConfig{
		APIKey: apiKey,
		Model:  "glm-4.7", // Use the default model
	}, log)

	ctx := context.Background()

	// Create a simple test request
	req := ChatRequest{
		Messages: []Message{
			{
				Role:    RoleUser,
				Content: "Hello! Please respond with just the word 'OK' and nothing else.",
			},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	t.Log("Sending test request to Z.ai API...")

	// Measure latency
	startTime := time.Now()

	// Send request
	resp, err := provider.Chat(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Log latency
	t.Logf("Request completed in %v", latency)

	// Validate response structure
	if resp == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("Response Content: %q", resp.Content)
	t.Logf("Response FinishReason: %s", resp.FinishReason)
	t.Logf("Response Model: %s", resp.Model)

	// Check Content field
	if resp.Content == "" {
		t.Error("Response Content is empty")
	}

	// Check FinishReason field
	if resp.FinishReason == "" {
		t.Error("Response FinishReason is empty")
	}

	// Check that FinishReason is valid
	validFinishReasons := map[FinishReason]bool{
		FinishReasonStop:      true,
		FinishReasonLength:    true,
		FinishReasonToolCalls: true,
		FinishReasonError:     true,
	}

	if !validFinishReasons[resp.FinishReason] {
		t.Errorf("Invalid FinishReason: %s", resp.FinishReason)
	}

	// Check Usage field
	if resp.Usage.TotalTokens <= 0 {
		t.Error("Usage.TotalTokens should be greater than 0")
	}

	if resp.Usage.PromptTokens <= 0 {
		t.Error("Usage.PromptTokens should be greater than 0")
	}

	if resp.Usage.CompletionTokens <= 0 {
		t.Error("Usage.CompletionTokens should be greater than 0")
	}

	t.Logf("Token Usage - Prompt: %d, Completion: %d, Total: %d",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens)

	// Check Model field
	if resp.Model == "" {
		t.Error("Response Model is empty")
	}

	// Log typical errors to watch for
	t.Log("Typical errors to watch for:")
	t.Log("  - Timeout errors: request took too long (check network/API status)")
	t.Log("  - 401 Unauthorized: invalid or expired API key")
	t.Log("  - 429 Rate Limit: too many requests (implement backoff)")
	t.Log("  - 500+ Server errors: Z.ai API issues (retry with backoff)")
}

// ExampleZAIProvider_Chat demonstrates how to use the ZAIProvider
// for sending chat completion requests.
func ExampleZAIProvider_Chat() {
	// Create a logger
	log, err := logger.New(logger.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		panic(err)
	}

	// Initialize the ZAI provider with your API key
	// Note: In production, load API key from config or environment
	provider := NewZAIProvider(ZAIConfig{
		APIKey: "your-api-key-here",
		Model:  "glm-4.7", // or use the default
	}, log)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare the chat request
	req := ChatRequest{
		Messages: []Message{
			{
				Role:    RoleSystem,
				Content: "You are a helpful assistant.",
			},
			{
				Role:    RoleUser,
				Content: "What is the capital of France?",
			},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   500,
	}

	// Send the request
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		// Handle error
		// Common errors:
		// - context.DeadlineExceeded: request timed out
		// - HTTP 401/403: invalid API key or permissions
		// - HTTP 429: rate limit exceeded
		// - HTTP 5xx: server error
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Process the response
	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Finish Reason: %s\n", resp.FinishReason)
	fmt.Printf("Tokens Used: %d\n", resp.Usage.TotalTokens)

	// Check for tool calls (if tool calling is enabled)
	if len(resp.ToolCalls) > 0 {
		fmt.Printf("Tool Calls: %d\n", len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			fmt.Printf("  - %s(%s)\n", tc.Name, tc.Arguments)
		}
	}
}

// TestZAIClient_WithToolCalling tests sending a request with tool definitions.
// This test verifies that the provider correctly handles tool definitions.
//
// To run this test, set the ZAI_API_KEY environment variable:
//
//	ZAI_API_KEY=your_api_key go test -v -run TestZAIClient_WithToolCalling ./internal/llm
func TestZAIClient_WithToolCalling(t *testing.T) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: ZAI_API_KEY environment variable not set")
	}

	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewZAIProvider(ZAIConfig{
		APIKey: apiKey,
	}, log)

	ctx := context.Background()

	// Create a request with tool definitions
	req := ChatRequest{
		Messages: []Message{
			{
				Role:    RoleUser,
				Content: "What's the weather in Tokyo? Use the weather tool.",
			},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   200,
		Tools: []ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	t.Log("Sending request with tool definitions...")

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	t.Logf("Response: %s", resp.Content)
	t.Logf("Tool Calls: %d", len(resp.ToolCalls))

	// Verify that the provider supports tool calling
	if !provider.SupportsToolCalling() {
		t.Error("Provider should support tool calling")
	}

	// If the model requested tool calls, log them
	if len(resp.ToolCalls) > 0 {
		for i, tc := range resp.ToolCalls {
			t.Logf("  Tool Call %d: %s(%s)", i, tc.Name, tc.Arguments)
		}
	}
}

// BenchmarkZAIClient_Latency measures typical latency for requests.
// This benchmark helps understand expected response times.
//
// To run this benchmark, set the ZAI_API_KEY environment variable:
//
//	ZAI_API_KEY=your_api_key go test -bench=BenchmarkZAIClient_Latency -benchmem ./internal/llm
func BenchmarkZAIClient_Latency(b *testing.B) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		b.Skip("Skipping benchmark: ZAI_API_KEY environment variable not set")
	}

	log, err := logger.New(logger.Config{
		Level:  "error", // Minimal logging for benchmarks
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewZAIProvider(ZAIConfig{
		APIKey: apiKey,
	}, log)

	ctx := context.Background()

	req := ChatRequest{
		Messages: []Message{
			{
				Role:    RoleUser,
				Content: "Say 'Hello' in one word.",
			},
		},
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   50,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := provider.Chat(ctx, req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}
