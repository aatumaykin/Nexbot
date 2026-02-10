package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestLoop_AddMessageToSession tests adding messages to session.
func TestLoop_AddMessageToSession(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Hello!",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	tests := []struct {
		name      string
		sessionID string
		message   llm.Message
		wantErr   bool
	}{
		{
			name:      "add user message",
			sessionID: "test-session-1",
			message: llm.Message{
				Role:    llm.RoleUser,
				Content: "Hello, bot!",
			},
			wantErr: false,
		},
		{
			name:      "add assistant message",
			sessionID: "test-session-2",
			message: llm.Message{
				Role:    llm.RoleAssistant,
				Content: "Hi there!",
			},
			wantErr: false,
		},
		{
			name:      "add system message",
			sessionID: "test-session-3",
			message: llm.Message{
				Role:    llm.RoleSystem,
				Content: "You are helpful assistant",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := looper.sessionOps.AddMessageToSession(ctx, tt.sessionID, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddMessageToSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify message was added
				history, err := looper.sessionOps.GetSessionHistory(ctx, tt.sessionID)
				if err != nil {
					t.Errorf("Failed to get session history: %v", err)
					return
				}
				if len(history) != 1 {
					t.Errorf("Expected 1 message, got %d", len(history))
					return
				}
				if history[0].Role != tt.message.Role || history[0].Content != tt.message.Content {
					t.Errorf("Message mismatch: got %+v, want %+v", history[0], tt.message)
				}
			}
		})
	}
}

// TestLoop_ClearSession tests clearing sessions.
func TestLoop_ClearSession(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Response",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session"

	// Add some messages
	if err := looper.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 1"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := looper.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 2"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Verify messages exist
	history, _ := looper.sessionOps.GetSessionHistory(ctx, sessionID)
	if len(history) != 2 {
		t.Fatalf("Expected 2 messages before clear, got %d", len(history))
	}

	// Clear session
	err := looper.sessionOps.ClearSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}

	// Verify session is cleared
	history, _ = looper.sessionOps.GetSessionHistory(ctx, sessionID)
	if len(history) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(history))
	}
}

// TestLoop_DeleteSession tests deleting sessions.
func TestLoop_DeleteSession(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Response",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
	})

	sessionID := "test-session-delete"

	// Add a message to create session
	if err := looper.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Verify session exists
	history, _ := looper.sessionOps.GetSessionHistory(ctx, sessionID)
	if len(history) != 1 {
		t.Fatalf("Expected 1 message before delete, got %d", len(history))
	}

	// Delete session
	err := looper.sessionOps.DeleteSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session no longer exists (should be recreated as empty)
	history, _ = looper.sessionOps.GetSessionHistory(ctx, sessionID)
	if len(history) != 0 {
		t.Errorf("Expected empty session after delete, got %d messages", len(history))
	}
}

// TestLoop_Getters tests getter methods.
func TestLoop_Getters(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Response",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
		Model:       "test-model",
		MaxTokens:   2048,
		Temperature: 0.5,
	})

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "GetContextBuilder",
			test: func(t *testing.T) {
				cb := looper.GetContextBuilder()
				if cb == nil {
					t.Error("GetContextBuilder returned nil")
				}
			},
		},
		{
			name: "GetSessionManager",
			test: func(t *testing.T) {
				sm := looper.GetSessionManager()
				if sm == nil {
					t.Error("GetSessionManager returned nil")
				}
			},
		},
		{
			name: "GetLLMProvider",
			test: func(t *testing.T) {
				provider := looper.GetLLMProvider()
				if provider == nil {
					t.Error("GetLLMProvider returned nil")
				}
				if provider != mockProvider {
					t.Error("GetLLMProvider returned wrong provider")
				}
			},
		},
		{
			name: "GetSessionModel",
			test: func(t *testing.T) {
				ctx := context.Background()
				model := looper.GetSessionModel(ctx, "any-session")
				if model != "test-model" {
					t.Errorf("GetSessionModel returned %s, want test-model", model)
				}
			},
		},
		{
			name: "GetSessionMaxTokens",
			test: func(t *testing.T) {
				maxTokens := looper.GetSessionMaxTokens("any-session")
				if maxTokens != 2048 {
					t.Errorf("GetSessionMaxTokens returned %d, want 2048", maxTokens)
				}
			},
		},
		{
			name: "GetTools",
			test: func(t *testing.T) {
				tools := looper.GetTools()
				if tools == nil {
					t.Error("GetTools returned nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestLoop_ProcessHeartbeatCheck tests heartbeat processing.
func TestLoop_ProcessHeartbeatCheck(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	tests := []struct {
		name         string
		response     string
		finishReason llm.FinishReason
		wantErr      bool
	}{
		{
			name:         "successful heartbeat",
			response:     "HEARTBEAT_OK",
			finishReason: llm.FinishReasonStop,
			wantErr:      false,
		},
		{
			name:         "heartbeat with tasks",
			response:     "Tasks: task1, task2",
			finishReason: llm.FinishReasonStop,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockToolCallProvider{
				responses: []llm.ChatResponse{
					{
						Content:      tt.response,
						FinishReason: tt.finishReason,
						Usage:        llm.Usage{TotalTokens: 10},
					},
				},
				callIndex: 0,
			}

			looper, _ := NewLoop(Config{
				Workspace:   workspaceDir,
				SessionDir:  sessionDir,
				LLMProvider: mockProvider,
				Logger:      log,
			})

			result, err := looper.ProcessHeartbeatCheck(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessHeartbeatCheck() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.response {
				t.Errorf("ProcessHeartbeatCheck() returned %s, want %s", result, tt.response)
			}
		})
	}
}

// TestLoop_GetSessionStatus tests getting session status.
func TestLoop_GetSessionStatus(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Response",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}

	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
		Model:       "test-model",
		MaxTokens:   2048,
		Temperature: 0.5,
	})

	sessionID := "status-test-session"

	// Add messages
	if err := looper.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 1"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := looper.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 2"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Get status
	status, err := looper.GetSessionStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionStatus failed: %v", err)
	}

	// Verify status fields
	if status["session_id"] != sessionID {
		t.Errorf("session_id = %v, want %v", status["session_id"], sessionID)
	}

	if status["message_count"] != 2 {
		t.Errorf("message_count = %v, want 2", status["message_count"])
	}

	if status["model"] != "test-model" {
		t.Errorf("model = %v, want test-model", status["model"])
	}

	if status["max_tokens"] != 2048 {
		t.Errorf("max_tokens = %v, want 2048", status["max_tokens"])
	}

	if status["temperature"] != 0.5 {
		t.Errorf("temperature = %v, want 0.5", status["temperature"])
	}

	if _, ok := status["file_size"]; !ok {
		t.Error("status should contain file_size")
	}

	if _, ok := status["file_size_human"]; !ok {
		t.Error("status should contain file_size_human")
	}
}

// TestAgentMessageSender tests the message sender.
func TestAgentMessageSender(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		channelType string
		sessionID   string
		message     string
		wantErr     bool
	}{
		{
			name:        "send valid message",
			userID:      "user123",
			channelType: "telegram",
			sessionID:   "session456",
			message:     "Hello!",
			wantErr:     false,
		},
		{
			name:        "send empty message",
			userID:      "user123",
			channelType: "telegram",
			sessionID:   "session456",
			message:     "",
			wantErr:     false,
		},
		{
			name:        "send with special characters",
			userID:      "user123",
			channelType: "telegram",
			sessionID:   "session456",
			message:     "Hello, World! 🚀",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
			messageBus := bus.New(100, log)
			if err := messageBus.Start(context.Background()); err != nil {
				t.Fatalf("Failed to start message bus: %v", err)
			}
			defer func() {
				if err := messageBus.Stop(); err != nil {
					t.Logf("Warning: messageBus.Stop() error: %v", err)
				}
			}()

			sender := NewAgentMessageSender(messageBus, log)

			// Subscribe to outbound messages to simulate channel sending result
			outboundCh := messageBus.SubscribeOutbound(context.Background())
			go func() {
				for msg := range outboundCh {
					// Simulate successful send
					result := bus.MessageSendResult{
						CorrelationID: msg.CorrelationID,
						ChannelType:   msg.ChannelType,
						Success:       true,
						Timestamp:     time.Now(),
					}
					_ = messageBus.PublishSendResult(result)
				}
			}()

			_, err := sender.SendMessage(tt.userID, tt.channelType, tt.sessionID, tt.message, time.Second*30)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSessionOperations tests session operations.
func TestSessionOperations(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})

	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	sessionMgr, err := session.NewManager(sessionDir)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	sessionOps := NewSessionOperations(sessionMgr)

	// Create a loop for status testing
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}
	mockProvider := &mockToolCallProvider{
		responses: []llm.ChatResponse{
			{
				Content:      "Response",
				FinishReason: llm.FinishReasonStop,
				Usage:        llm.Usage{TotalTokens: 10},
			},
		},
		callIndex: 0,
	}
	looper, _ := NewLoop(Config{
		Workspace:   workspaceDir,
		SessionDir:  sessionDir,
		LLMProvider: mockProvider,
		Logger:      log,
		Model:       "test-model",
		MaxTokens:   2048,
		Temperature: 0.5,
	})

	t.Run("ClearSession", func(t *testing.T) {
		sessionID := "clear-test-session"

		// Add messages
		if err := sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 1"}); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
		if err := sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 2"}); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}

		// Verify messages exist
		history, _ := sessionOps.GetSessionHistory(ctx, sessionID)
		if len(history) != 2 {
			t.Fatalf("Expected 2 messages before clear, got %d", len(history))
		}

		// Clear
		err := sessionOps.ClearSession(ctx, sessionID)
		if err != nil {
			t.Fatalf("ClearSession failed: %v", err)
		}

		// Verify cleared
		history, _ = sessionOps.GetSessionHistory(ctx, sessionID)
		if len(history) != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", len(history))
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		sessionID := "delete-test-session"

		// Add a message
		if err := sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message"}); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}

		// Verify session exists
		history, _ := sessionOps.GetSessionHistory(ctx, sessionID)
		if len(history) != 1 {
			t.Fatalf("Expected 1 message before delete, got %d", len(history))
		}

		// Delete
		err := sessionOps.DeleteSession(ctx, sessionID)
		if err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}

		// Verify session is empty (recreated)
		history, _ = sessionOps.GetSessionHistory(ctx, sessionID)
		if len(history) != 0 {
			t.Errorf("Expected 0 messages after delete, got %d", len(history))
		}
	})

	t.Run("GetSessionStatus", func(t *testing.T) {
		sessionID := "status-test-session"

		// Add messages
		if err := sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 1"}); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
		if err := sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{Role: llm.RoleUser, Content: "Message 2"}); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}

		// Get status
		status, err := sessionOps.GetSessionStatus(ctx, sessionID, looper)
		if err != nil {
			t.Fatalf("GetSessionStatus failed: %v", err)
		}

		// Verify status
		if status["session_id"] != sessionID {
			t.Errorf("session_id = %v, want %v", status["session_id"], sessionID)
		}

		if status["message_count"] != 2 {
			t.Errorf("message_count = %v, want 2", status["message_count"])
		}
	})
}
