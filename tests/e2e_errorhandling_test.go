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
)

// TestE2E_ToolErrorHandling tests error handling when tools fail
func TestE2E_ToolErrorHandling(t *testing.T) {
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
		t.Logf("Response: %s", outboundMsg.Content)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

// TestE2E_ShellExecTool tests shell_exec tool execution
func TestE2E_ShellExecTool(t *testing.T) {
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
				return nil
			}

			if msg.Content == "" {
				continue
			}

			response, err := looper.Process(ctx, msg.SessionID, msg.Content)
			if err != nil {
				continue
			}

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
