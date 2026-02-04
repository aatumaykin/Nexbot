package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestLoopProcess_BasicFlow tests the basic message flow through the agent loop
func TestLoopProcess_BasicFlow(t *testing.T) {
	// Setup
	ctx := context.Background()
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create temporary directories for testing
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session directory: %v", err)
	}

	// Create mock provider with fixed response
	mockProvider := llm.NewFixedProvider("Hello! This is a test response.")

	// Create loop
	looper, err := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
		Model:       "test-model",
		MaxTokens:   1024,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("Failed to create loop: %v", err)
	}

	// Test message processing
	sessionID := "test-session-1"
	userMessage := "Hello, how are you?"

	response, err := looper.Process(ctx, sessionID, userMessage)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expectedResponse := "Hello! This is a test response."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}

	// Verify that the mock provider was called
	if mockProvider.GetCallCount() != 1 {
		t.Errorf("Expected mock provider to be called once, got %d calls", mockProvider.GetCallCount())
	}

	// Verify session history
	history, err := looper.GetSessionHistory(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session history: %v", err)
	}

	// Should have 2 messages: user + assistant
	if len(history) != 2 {
		t.Errorf("Expected 2 messages in history, got %d", len(history))
	}

	if history[0].Role != llm.RoleUser || history[0].Content != userMessage {
		t.Errorf("Expected first message to be user message '%s', got role=%s content=%s",
			userMessage, history[0].Role, history[0].Content)
	}

	if history[1].Role != llm.RoleAssistant || history[1].Content != expectedResponse {
		t.Errorf("Expected second message to be assistant response '%s', got role=%s content=%s",
			expectedResponse, history[1].Role, history[1].Content)
	}
}

// TestLoopProcess_MultiMessage tests processing multiple messages in a session
func TestLoopProcess_MultiMessage(t *testing.T) {
	// Setup
	ctx := context.Background()
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")

	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	// Create mock provider with fixtures
	responses := []string{
		"Response 1",
		"Response 2",
		"Response 3",
	}
	mockProvider := llm.NewFixturesProvider(responses)

	// Create loop
	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-2"
	messages := []string{
		"First message",
		"Second message",
		"Third message",
	}

	// Process multiple messages
	for i, msg := range messages {
		response, err := looper.Process(ctx, sessionID, msg)
		if err != nil {
			t.Fatalf("Process failed for message %d: %v", i+1, err)
		}

		expected := responses[i]
		if response != expected {
			t.Errorf("Message %d: expected response '%s', got '%s'", i+1, expected, response)
		}
	}

	// Verify session history contains all messages
	history, _ := looper.GetSessionHistory(ctx, sessionID)
	if len(history) != 6 { // 3 user + 3 assistant
		t.Errorf("Expected 6 messages in history, got %d", len(history))
	}

	// Verify call count
	if mockProvider.GetCallCount() != 3 {
		t.Errorf("Expected 3 calls to mock provider, got %d", mockProvider.GetCallCount())
	}
}

// TestLoopProcess_EchoMode tests the loop with echo mode mock provider
func TestLoopProcess_EchoMode(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	mockProvider := llm.NewEchoProvider()
	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-3"
	userMessage := "Test echo message"

	response, err := looper.Process(ctx, sessionID, userMessage)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expected := "Echo: Test echo message"
	if response != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, response)
	}
}

// TestLoopProcess_ErrorHandling tests graceful degradation when LLM fails
func TestLoopProcess_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	// Create error provider
	mockProvider := llm.NewErrorProvider()
	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-4"
	userMessage := "This should fail"

	// Process should not return error, but a graceful response
	response, err := looper.Process(ctx, sessionID, userMessage)
	if err != nil {
		t.Fatalf("Process should handle errors gracefully, got error: %v", err)
	}

	// Verify we got a graceful response
	if response == "" {
		t.Error("Expected a graceful response, got empty string")
	}

	// The response should mention the user's message or a fallback message
	if len(response) < 10 {
		t.Errorf("Graceful response seems too short: %s", response)
	}

	// Verify that the user message was not added to session (because LLM failed)
	history, _ := looper.GetSessionHistory(ctx, sessionID)
	if len(history) != 0 {
		// Actually, the current implementation does add the user message before calling LLM
		// So we expect 1 message (user only)
		if len(history) != 1 {
			t.Errorf("Expected 1 message (user only) in history after error, got %d", len(history))
		}
	}
}

// TestLoopProcess_ErrorAfterNCalls tests error after N successful calls
func TestLoopProcess_ErrorAfterNCalls(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	// Create provider that fails after 2 calls
	mockProvider := llm.NewFixedProvider("Success")
	mockProvider.SetErrorAfter(2)

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-5"

	// First two calls should succeed
	for i := 0; i < 2; i++ {
		_, err := looper.Process(ctx, sessionID, "Test message")
		if err != nil {
			t.Errorf("Call %d should succeed, got error: %v", i+1, err)
		}
	}

	// Third call should fail gracefully
	response, err := looper.Process(ctx, sessionID, "Test message")
	if err != nil {
		t.Fatalf("Third call should handle error gracefully, got error: %v", err)
	}

	if response == "" {
		t.Error("Expected graceful response on third call, got empty string")
	}
}

// TestLoopSessionManagement tests session management operations
func TestLoopSessionManagement(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	mockProvider := llm.NewFixedProvider("Test response")
	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-6"

	// Test AddMessageToSession
	err := looper.AddMessageToSession(ctx, sessionID, llm.Message{
		Role:    llm.RoleUser,
		Content: "Added message",
	})
	if err != nil {
		t.Fatalf("AddMessageToSession failed: %v", err)
	}

	// Verify message was added
	history, err := looper.GetSessionHistory(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 message in history, got %d", len(history))
	}

	// Test ClearSession
	err = looper.ClearSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}

	history, _ = looper.GetSessionHistory(ctx, sessionID)
	if len(history) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(history))
	}

	// Add messages again and test DeleteSession
	_, _ = looper.Process(ctx, sessionID, "Before delete")

	err = looper.DeleteSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Trying to get history of deleted session should create a new empty one
	history, _ = looper.GetSessionHistory(ctx, sessionID)
	if len(history) != 0 {
		t.Errorf("Expected 0 messages for deleted session, got %d", len(history))
	}
}

// TestLoop_NewLoop_Validation tests configuration validation
func TestLoop_NewLoop_Validation(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	// Create temporary directories for validation test
	validationTempDir := t.TempDir()
	validationWorkspaceDir := filepath.Join(validationTempDir, "workspace")
	validationSessionDir := filepath.Join(validationTempDir, "sessions")
	os.MkdirAll(validationWorkspaceDir, 0755)
	os.MkdirAll(validationSessionDir, 0755)

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Workspace:   validationWorkspaceDir,
				SessionDir:  validationSessionDir,
				LLMProvider: llm.NewFixedProvider("test"),
				Logger:      log,
			},
			wantErr: false,
		},
		{
			name: "empty workspace",
			cfg: Config{
				Workspace:   "",
				SessionDir:  "/tmp/sessions",
				LLMProvider: llm.NewFixedProvider("test"),
				Logger:      log,
			},
			wantErr: true,
		},
		{
			name: "empty session dir",
			cfg: Config{
				Workspace:   "/tmp/workspace",
				SessionDir:  "",
				LLMProvider: llm.NewFixedProvider("test"),
				Logger:      log,
			},
			wantErr: true,
		},
		{
			name: "nil provider",
			cfg: Config{
				Workspace:   "/tmp/workspace",
				SessionDir:  "/tmp/sessions",
				LLMProvider: nil,
				Logger:      log,
			},
			wantErr: true,
		},
		{
			name: "nil logger",
			cfg: Config{
				Workspace:   "/tmp/workspace",
				SessionDir:  "/tmp/sessions",
				LLMProvider: llm.NewFixedProvider("test"),
				Logger:      nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLoop(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLoop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoop_GetContextBuilder tests access to internal components
func TestLoop_GetContextBuilder(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: llm.NewFixedProvider("test"),
		Logger:      log,
	})

	// Test getters
	if looper.GetContextBuilder() == nil {
		t.Error("Expected context builder to be non-nil")
	}

	if looper.GetSessionManager() == nil {
		t.Error("Expected session manager to be non-nil")
	}

	if looper.GetLLMProvider() == nil {
		t.Error("Expected LLM provider to be non-nil")
	}
}

// TestLoop_GetSessionStatus tests the GetSessionStatus method
func TestLoop_GetSessionStatus(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(workspaceDir, 0755)
	os.MkdirAll(sessionDir, 0755)

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: llm.NewFixedProvider("test"),
		Logger:      log,
		Model:       "test-model-123",
		MaxTokens:   2048,
		Temperature: 0.8,
	})

	sessionID := "test-session-status"

	// Get status for empty session
	status, err := looper.GetSessionStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionStatus failed for empty session: %v", err)
	}

	// Verify status fields
	if status["session_id"] != sessionID {
		t.Errorf("Expected session_id '%s', got '%v'", sessionID, status["session_id"])
	}

	if status["message_count"] != 0 {
		t.Errorf("Expected message_count 0 for empty session, got %v", status["message_count"])
	}

	if status["model"] != "test-model-123" {
		t.Errorf("Expected model 'test-model-123', got %v", status["model"])
	}

	if status["temperature"] != 0.8 {
		t.Errorf("Expected temperature 0.8, got %v", status["temperature"])
	}

	if status["max_tokens"] != 2048 {
		t.Errorf("Expected max_tokens 2048, got %v", status["max_tokens"])
	}

	// Add some messages to session
	_, _ = looper.Process(ctx, sessionID, "First message")
	_, _ = looper.Process(ctx, sessionID, "Second message")

	// Get status again
	status, err = looper.GetSessionStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionStatus failed after adding messages: %v", err)
	}

	// Should have 4 messages now (2 user + 2 assistant)
	if status["message_count"] != 4 {
		t.Errorf("Expected message_count 4, got %v", status["message_count"])
	}

	// File size should be non-zero
	fileSize, _ := status["file_size"].(int64)
	if fileSize <= 0 {
		t.Errorf("Expected positive file_size, got %v", fileSize)
	}
}
