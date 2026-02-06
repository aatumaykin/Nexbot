package llm

import (
	"context"
	"strings"
	"testing"
)

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
	if err == nil {
		t.Fatal("Chat() expected error, got nil")
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
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Echo: (no user message)" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Echo: (no user message)")
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
