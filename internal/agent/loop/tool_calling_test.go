package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools/file"
	"github.com/aatumaykin/nexbot/internal/workspace"
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

// TestLoop_ToolCalling tests the tool calling integration.
func TestLoop_ToolCalling(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	// Create test file in workspace directory
	testFile := filepath.Join(ws.Path(), "test.txt")
	testContent := "Hello, World!\nThis is a test file."
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock provider with tool call response
	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "",
				FinishReason: llm.FinishReasonToolCalls,
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_123",
						Name: "read_file",
						Arguments: jsonMapToString(map[string]interface{}{
							"path": "test.txt",
						}),
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				Content:      "I found the content of the test file!",
				FinishReason: llm.FinishReasonStop,
				ToolCalls:    nil,
				Usage:        llm.Usage{TotalTokens: 20},
			},
		},
		callIndex: 0,
	}

	// Create loop
	looper, _ := NewLoop(Config{
		Workspace:    ws,
		WorkspaceCfg: config.WorkspaceConfig{Path: workspaceDir},
		SessionDir:   sessionDir,
		LLMProvider:  mockProvider,
		Logger:       log,
	})

	// Register read_file tool
	readFileTool := file.NewReadFileTool(ws, testConfig())
	if err := looper.RegisterTool(readFileTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Process message
	sessionID := "tool-test-session"
	userMessage := "Please read the test.txt file"

	response, err := looper.Process(ctx, sessionID, userMessage)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify final response
	expectedResponse := "I found the content of the test file!"
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}

	// Verify mock provider was called twice (initial + after tool result)
	if mockProvider.GetCallCount() != 2 {
		t.Errorf("Expected 2 calls to provider, got %d", mockProvider.GetCallCount())
	}

	// Verify session history
	history, _ := looper.GetSessionHistory(ctx, sessionID)

	// Expected messages:
	// 1. User: "Please read the test.txt file"
	// 2. Assistant: "" (with tool call)
	// 3. Tool: result of read_file
	// 4. Assistant: "I found the content of the test file!"

	if len(history) != 4 {
		t.Errorf("Expected 4 messages in history, got %d", len(history))
		for i, msg := range history {
			t.Logf("Message %d: Role=%s, Content=%s, ToolCallID=%s",
				i, msg.Role, msg.Content, msg.ToolCallID)
		}
	}

	// Verify tool result is in history
	foundToolResult := false
	for _, msg := range history {
		if msg.Role == llm.RoleTool && msg.ToolCallID == "call_123" {
			foundToolResult = true
			if !contains(msg.Content, "Hello, World!") {
				t.Errorf("Tool result should contain file content, got: %s", msg.Content)
			}
		}
	}

	if !foundToolResult {
		t.Error("Tool result not found in session history")
	}
}

// mockToolCallProvider is a mock provider that simulates tool calling.
type mockToolCallProvider struct {
	responses []llm.ChatResponse
	callIndex int
}

func (m *mockToolCallProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.callIndex >= len(m.responses) {
		return &llm.ChatResponse{
			Content:      "Default response",
			FinishReason: llm.FinishReasonStop,
		}, nil
	}

	resp := m.responses[m.callIndex]
	m.callIndex++
	return &resp, nil
}

func (m *mockToolCallProvider) SupportsToolCalling() bool {
	return true
}

func (m *mockToolCallProvider) GetCallCount() int {
	return m.callIndex
}

// jsonMapToString converts a map to JSON string.
func jsonMapToString(m map[string]interface{}) string {
	data, _ := json.Marshal(m)
	return string(data)
}

// Helper function
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
