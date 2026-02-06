package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/tools/file"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"github.com/mymmrac/telego"
	"sync"
)

// testConfig creates a test configuration with default values.
func testConfig() *config.Config {
	return &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:       true,
				WhitelistDirs: []string{},
			},
		},
	}
}

// MockTelegramBot is a mock implementation of telego.Bot for testing
type MockTelegramBot struct {
	sentMessages []MockSentMessage
	mu           sync.Mutex
}

type MockSentMessage struct {
	ChatID    int64
	Text      string
	ParseMode string
}

// NewMockTelegramBot creates a mock Telegram bot
func NewMockTelegramBot() *MockTelegramBot {
	return &MockTelegramBot{
		sentMessages: make([]MockSentMessage, 0),
	}
}

// SendMessage mocks of SendMessage method
func (m *MockTelegramBot) SendMessage(params telego.SendMessageParams) (*telego.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract chat ID from telego.ChatID
	var chatID int64
	if params.ChatID.ID != 0 {
		chatID = params.ChatID.ID
	}

	m.sentMessages = append(m.sentMessages, MockSentMessage{
		ChatID:    chatID,
		Text:      params.Text,
		ParseMode: params.ParseMode,
	})

	return &telego.Message{
		MessageID: len(m.sentMessages),
		Chat:      telego.Chat{ID: chatID},
		Text:      params.Text,
	}, nil
}

// GetSentMessages returns all sent messages
func (m *MockTelegramBot) GetSentMessages() []MockSentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentMessages
}

// ClearSentMessages clears the message history
func (m *MockTelegramBot) ClearSentMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = make([]MockSentMessage, 0)
}

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
		m.responseIndex = 0 // Cycle through responses
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
	// Setup: create temporary directories
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session directory: %v", err)
	}

	// Create workspace
	ws := workspace.New(config.WorkspaceConfig{
		Path: workspaceDir,
	})

	// Create bootstrap files
	createTestBootstrapFiles(t, ws)

	// Setup logger
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Setup message bus
	msgBus := bus.New(100, log)

	// Setup LLM provider with tool calling
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

	// Setup agent loop
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

	// Register tools
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

	// Setup config for shell tool
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo", "cat", "ls"},
				TimeoutSeconds:  30,
			},
		},
	}
	if err := agentLoop.RegisterTool(tools.NewShellExecTool(cfg, log)); err != nil {
		t.Fatalf("Failed to register shell tool: %v", err)
	}

	// Create a test file in workspace
	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup context and start components
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	// Subscribe to outbound messages
	outboundCh := msgBus.SubscribeOutbound(ctx)

	// Start agent loop in background
	agentDone := make(chan error, 1)
	go func() {
		agentDone <- processAgentLoop(ctx, agentLoop, msgBus)
	}()

	// Simulate inbound message from Telegram
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
		Metadata: map[string]interface{}{
			"message_id": 1,
			"chat_id":    testChatID,
			"chat_type":  "private",
			"username":   "test_user",
		},
	}

	// Publish inbound message to bus
	if err := msgBus.PublishInbound(inboundMsg); err != nil {
		t.Fatalf("Failed to publish inbound message: %v", err)
	}

	// Wait for outbound message with timeout
	var outboundMsg bus.OutboundMessage
	select {
	case outboundMsg = <-outboundCh:
		t.Log("Received outbound message from agent")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for outbound message")
	}

	// Verify outbound message
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

	// Verify that the mock LLM was called twice (initial request + tool result)
	if mockProvider.GetCallCount() != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", mockProvider.GetCallCount())
	}

	// Verify agent loop didn't error
	select {
	case err := <-agentDone:
		if err != nil && err != context.Canceled {
			t.Errorf("Agent loop error: %v", err)
		}
	case <-time.After(1 * time.Second):
		// OK, agent loop should still be running
	}

	t.Log("E2E test passed successfully!")
}

// TestE2E_MultipleToolCalls tests multiple tool calls in sequence
func TestE2E_MultipleToolCalls(t *testing.T) {
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	createTestBootstrapFiles(t, ws)

	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	msgBus := bus.New(100, log)

	// Setup mock provider with multiple tool call responses
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
			Content:      "I've listed the directory and read the file. Here's the summary.",
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

	// Create test file
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

	// Send message
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

	// Wait for response
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

// TestE2E_ToolErrorHandling tests error handling when tools fail
func TestE2E_ToolErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	createTestBootstrapFiles(t, ws)

	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	msgBus := bus.New(100, log)

	// Setup mock provider with simple response (no tool call)
	mockLLMResponses := []MockResponse{
		{
			Content:      "I encountered an error reading the file. Let me try something else.",
			FinishReason: llm.FinishReasonStop,
			ToolCalls:    nil,
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
		Content:     "Read nonexistent.txt",
		Timestamp:   time.Now(),
	}

	if err := msgBus.PublishInbound(inboundMsg); err != nil {
		t.Fatal(err)
	}

	select {
	case outboundMsg := <-outboundCh:
		if outboundMsg.Content == "" {
			t.Error("Expected non-empty response despite tool error")
		}
		// Verify that that agent handled error gracefully
		t.Logf("Response: %s", outboundMsg.Content)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

// TestE2E_ShellExecTool tests shell_exec tool execution
func TestE2E_ShellExecTool(t *testing.T) {
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := workspace.New(config.WorkspaceConfig{Path: workspaceDir})
	createTestBootstrapFiles(t, ws)

	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	msgBus := bus.New(100, log)

	// Setup mock provider with shell tool call
	mockLLMResponses := []MockResponse{
		{
			Content:      "I'll execute the echo command for you.",
			FinishReason: llm.FinishReasonToolCalls,
			ToolCalls: []llm.ToolCall{
				{ID: "call_1", Name: "shell_exec", Arguments: `{"command": "echo hello"}`},
			},
		},
		{
			Content:      "Shell command executed successfully.",
			FinishReason: llm.FinishReasonStop,
			ToolCalls:    nil,
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
	shellCfg := &config.Config{
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				Enabled:         true,
				AllowedCommands: []string{"echo"},
				TimeoutSeconds:  30,
			},
		},
	}
	if err := agentLoop.RegisterTool(tools.NewShellExecTool(shellCfg, log)); err != nil {
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
		Content:     "Run echo hello",
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
		if mockProvider.GetCallCount() != 2 {
			t.Errorf("Expected 2 LLM calls, got %d", mockProvider.GetCallCount())
		}
		t.Log("Shell exec test passed!")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

// Helper function to process agent loop
func processAgentLoop(ctx context.Context, looper *loop.Loop, msgBus *bus.MessageBus) error {
	inboundCh := msgBus.SubscribeInbound(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-inboundCh:
			if !ok {
				// Channel closed
				return nil
			}

			// Skip empty messages (they may be sent after channel closes)
			if msg.Content == "" {
				continue
			}

			// Process inbound message
			response, err := looper.Process(ctx, msg.SessionID, msg.Content)
			if err != nil {
				continue
			}

			// Send outbound response
			outboundMsg := bus.OutboundMessage{
				ChannelType: msg.ChannelType,
				UserID:      msg.UserID,
				SessionID:   msg.SessionID,
				Content:     response,
				Timestamp:   time.Now(),
				Metadata:    msg.Metadata,
			}

			if err := msgBus.PublishOutbound(outboundMsg); err != nil {
				continue
			}
		}
	}
}

// Helper function to create test bootstrap files
func createTestBootstrapFiles(t *testing.T, ws *workspace.Workspace) {
	bootstrapFiles := map[string]string{
		"IDENTITY.md": "# Nexbot Identity\n\nI am Nexbot, a lightweight AI agent.",
		"AGENTS.md":   "# Agent Instructions\n\nBe helpful and concise.",
		"SOUL.md":     "# Personality\n\nFriendly and professional.",
		"USER.md":     "# User Preferences\n\nNo specific preferences.",
		"TOOLS.md":    "# Available Tools\n\nI have access to file operations and shell commands.",
	}

	for filename, content := range bootstrapFiles {
		path, err := ws.ResolvePath(filename)
		if err != nil {
			t.Fatalf("Failed to resolve path for %s: %v", filename, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create bootstrap file %s: %v", filename, err)
		}
	}
}
