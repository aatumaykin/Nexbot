package tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools/file"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// ToolCallingMockProvider is a mock provider that supports tool calling
type ToolCallingMockProvider struct {
	responses     []MockResponse
	responseIndex int
	callCount     int
	mu            sync.Mutex
}

type MockResponse struct {
	Content      string
	FinishReason llm.FinishReason
	ToolCalls    []llm.ToolCall
}

// NewToolCallingMockProvider creates a mock provider with tool calling support
func NewToolCallingMockProvider(responses []MockResponse) *ToolCallingMockProvider {
	return &ToolCallingMockProvider{
		responses: responses,
	}
}

// Chat implements the Provider interface
func (m *ToolCallingMockProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.responseIndex >= len(m.responses) {
		m.responseIndex = 0
	}

	resp := m.responses[m.responseIndex]
	m.responseIndex++

	return &llm.ChatResponse{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
		ToolCalls:    resp.ToolCalls,
		Usage: llm.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Model: "mock-tool-model",
	}, nil
}

// SupportsToolCalling returns true
func (m *ToolCallingMockProvider) SupportsToolCalling() bool {
	return true
}

// GetCallCount returns number of Chat() calls
func (m *ToolCallingMockProvider) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// TestE2E_TelegramToAgentWithToolCalls tests the full E2E flow:
// Telegram message -> bus -> agent -> LLM with tool calls -> tools execution -> response
func TestE2E_TelegramToAgentWithToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session directory: %v", err)
	}

	ws := workspace.New(config.WorkspaceConfig{
		Path: workspaceDir,
	})
	createTestBootstrapFiles(t, ws)

	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	msgBus := bus.New(100, 10, log)

	mockLLMResponses := []MockResponse{
		{
			Content:      "I'll read the file for you.",
			FinishReason: llm.FinishReasonToolCalls,
			ToolCalls: []llm.ToolCall{
				{
					ID:        "call_1",
					Name:      "read_file",
					Arguments: `{"path": "test.txt"}`,
				},
			},
		},
		{
			Content:      "Here's the content of the file: Hello, World!",
			FinishReason: llm.FinishReasonStop,
			ToolCalls:    nil,
		},
	}
	mockProvider := NewToolCallingMockProvider(mockLLMResponses)

	agentLoop, err := loop.NewLoop(loop.Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
		Model:       "mock-tool-model",
		MaxTokens:   1024,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("Failed to create agent loop: %v", err)
	}

	wsForTools := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	if err := agentLoop.RegisterTool(file.NewReadFileTool(wsForTools, testConfig())); err != nil {
		t.Fatalf("Failed to register read file tool: %v", err)
	}
	if err := agentLoop.RegisterTool(file.NewWriteFileTool(wsForTools, testConfig())); err != nil {
		t.Fatalf("Failed to register write file tool: %v", err)
	}
	if err := agentLoop.RegisterTool(file.NewListDirTool(wsForTools, testConfig())); err != nil {
		t.Fatalf("Failed to register list dir tool: %v", err)
	}

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	outboundCh := msgBus.SubscribeOutbound(ctx)

	agentDone := make(chan error, 1)
	go func() {
		agentDone <- processAgentLoop(ctx, agentLoop, msgBus)
	}()

	testUserID := "123456789"
	testChatID := int64(987654321)
	testSessionID := "987654321"
	testMessage := "Read test.txt file"

	inboundMsg := bus.InboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      testUserID,
		SessionID:   testSessionID,
		Content:     testMessage,
		Timestamp:   time.Now(),
		Metadata: map[string]any{
			"message_id": 1,
			"chat_id":    testChatID,
			"chat_type":  "private",
			"username":   "test_user",
		},
	}

	if err := msgBus.PublishInbound(inboundMsg); err != nil {
		t.Fatalf("Failed to publish inbound message: %v", err)
	}

	var outboundMsg bus.OutboundMessage
	select {
	case outboundMsg = <-outboundCh:
		t.Log("Received outbound message from agent")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for outbound message")
	}

	if outboundMsg.ChannelType != bus.ChannelTypeTelegram {
		t.Errorf("Expected channel type telegram, got %s", outboundMsg.ChannelType)
	}
	if outboundMsg.UserID != testUserID {
		t.Errorf("Expected user ID %s, got %s", testUserID, outboundMsg.UserID)
	}
	if outboundMsg.SessionID != testSessionID {
		t.Errorf("Expected session ID %s, got %s", testSessionID, outboundMsg.SessionID)
	}
	if outboundMsg.Content == "" {
		t.Error("Expected non-empty response content")
	}
	if outboundMsg.Content != "Here's the content of the file: Hello, World!" {
		t.Errorf("Unexpected response content: %s", outboundMsg.Content)
	}

	if mockProvider.GetCallCount() != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", mockProvider.GetCallCount())
	}

	select {
	case err := <-agentDone:
		if err != nil && err != context.Canceled {
			t.Errorf("Agent loop error: %v", err)
		}
	case <-time.After(1 * time.Second):
	}

	t.Log("E2E test passed successfully!")
}

// TestE2E_MultipleToolCalls tests multiple tool calls in sequence
func TestE2E_MultipleToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	createTestBootstrapFiles(t, ws)

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}
	msgBus := bus.New(100, 10, log)

	mockLLMResponses := []MockResponse{
		{
			FinishReason: llm.FinishReasonToolCalls,
			ToolCalls: []llm.ToolCall{
				{ID: "call_1", Name: "list_dir", Arguments: `{"path": ".", "recursive": false}`},
			},
		},
		{
			FinishReason: llm.FinishReasonToolCalls,
			ToolCalls: []llm.ToolCall{
				{ID: "call_2", Name: "read_file", Arguments: `{"path": "test.txt"}`},
			},
		},
		{
			Content:      "I've listed directory and read the file. Here's the summary.",
			FinishReason: llm.FinishReasonStop,
		},
	}
	mockProvider := NewToolCallingMockProvider(mockLLMResponses)

	agentLoop, _ := loop.NewLoop(loop.Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	wsForTools := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	if err := agentLoop.RegisterTool(file.NewReadFileTool(wsForTools, testConfig())); err != nil {
		t.Fatal(err)
	}
	if err := agentLoop.RegisterTool(file.NewListDirTool(wsForTools, testConfig())); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(workspaceDir, "test.txt"), []byte("Test content"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatal(err)
	}

	outboundCh := msgBus.SubscribeOutbound(ctx)

	agentDone := make(chan error, 1)
	go func() {
		agentDone <- processAgentLoop(ctx, agentLoop, msgBus)
	}()

	inboundMsg := bus.InboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123",
		SessionID:   "456",
		Content:     "List directory and read test.txt",
		Timestamp:   time.Now(),
	}

	if err := msgBus.PublishInbound(inboundMsg); err != nil {
		t.Fatal(err)
	}

	select {
	case outboundMsg := <-outboundCh:
		if outboundMsg.Content == "" {
			t.Error("Expected non-empty response")
		}
		if mockProvider.GetCallCount() != 3 {
			t.Errorf("Expected 3 LLM calls, got %d", mockProvider.GetCallCount())
		}
		t.Log("Multiple tool calls test passed!")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}
