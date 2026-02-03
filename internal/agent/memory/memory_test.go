package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

func TestNewStore(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid JSONL store",
			config: Config{
				BaseDir: "/tmp/test-memory-jsonl",
				Format:  FormatJSONL,
			},
			wantErr: false,
		},
		{
			name: "valid Markdown store",
			config: Config{
				BaseDir: "/tmp/test-memory-markdown",
				Format:  FormatMarkdown,
			},
			wantErr: false,
		},
		{
			name: "empty base directory",
			config: Config{
				BaseDir: "",
				Format:  FormatJSONL,
			},
			wantErr: true,
		},
		{
			name: "default format when not specified",
			config: Config{
				BaseDir: "/tmp/test-memory-default",
				Format:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				defer os.RemoveAll(tt.config.BaseDir)
			}

			store, err := NewStore(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if store == nil {
					t.Error("NewStore() returned nil store")
				}

				// Verify default format is set
				if tt.config.Format == "" && store.format != FormatJSONL {
					t.Errorf("Expected default format JSONL, got %v", store.format)
				}

				// Verify directory was created
				if _, err := os.Stat(tt.config.BaseDir); os.IsNotExist(err) {
					t.Error("Base directory should be created")
				}
			}
		})
	}
}

func TestWriteAndReadJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("write and read single message", func(t *testing.T) {
		sessionID := "test-session-1"
		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: "Hello, world!",
		}

		// Write
		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Read
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if messages[0].Role != llm.RoleUser {
			t.Errorf("Message role = %v, want %v", messages[0].Role, llm.RoleUser)
		}

		if messages[0].Content != "Hello, world!" {
			t.Errorf("Message content = %v, want 'Hello, world!'", messages[0].Content)
		}
	})

	t.Run("write and read multiple messages", func(t *testing.T) {
		sessionID := "test-session-2"
		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are helpful assistant"},
			{Role: llm.RoleUser, Content: "What is 2+2?"},
			{Role: llm.RoleAssistant, Content: "2+2=4"},
		}

		for _, msg := range messages {
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		readMessages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(readMessages) != len(messages) {
			t.Fatalf("Read() returned %d messages, want %d", len(readMessages), len(messages))
		}

		for i, want := range messages {
			got := readMessages[i]
			if got.Role != want.Role {
				t.Errorf("Message %d role = %v, want %v", i, got.Role, want.Role)
			}
			if got.Content != want.Content {
				t.Errorf("Message %d content = %v, want %v", i, got.Content, want.Content)
			}
		}
	})

	t.Run("write with tool call", func(t *testing.T) {
		sessionID := "test-session-3"
		msg := llm.Message{
			Role:       llm.RoleTool,
			Content:    "Tool result",
			ToolCallID: "call_123",
		}

		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if messages[0].ToolCallID != "call_123" {
			t.Errorf("ToolCallID = %v, want 'call_123'", messages[0].ToolCallID)
		}
	})
}

func TestWriteAndReadMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatMarkdown,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("write and read single message", func(t *testing.T) {
		sessionID := "test-md-session-1"
		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: "Hello, world!",
		}

		// Write
		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Read
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if messages[0].Role != llm.RoleUser {
			t.Errorf("Message role = %v, want %v", messages[0].Role, llm.RoleUser)
		}

		// Content may contain formatting, just check it contains the original text
		if !strings.Contains(messages[0].Content, "Hello, world!") {
			t.Errorf("Message content should contain 'Hello, world!', got %v", messages[0].Content)
		}
	})

	t.Run("write and read multiple messages", func(t *testing.T) {
		sessionID := "test-md-session-2"
		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "System message"},
			{Role: llm.RoleUser, Content: "User message"},
			{Role: llm.RoleAssistant, Content: "Assistant message"},
		}

		for _, msg := range messages {
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		readMessages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(readMessages) != len(messages) {
			t.Fatalf("Read() returned %d messages, want %d", len(readMessages), len(messages))
		}

		for i, want := range messages {
			got := readMessages[i]
			if got.Role != want.Role {
				t.Errorf("Message %d role = %v, want %v", i, got.Role, want.Role)
			}
			if !strings.Contains(got.Content, want.Content) {
				t.Errorf("Message %d content should contain %v, got %v", i, want.Content, got.Content)
			}
		}
	})

	t.Run("write with tool call", func(t *testing.T) {
		sessionID := "test-md-session-3"
		msg := llm.Message{
			Role:       llm.RoleTool,
			Content:    "Tool result",
			ToolCallID: "call_123",
		}

		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if messages[0].ToolCallID != "call_123" {
			t.Errorf("ToolCallID = %v, want 'call_123'", messages[0].ToolCallID)
		}
	})
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name   string
		format Format
	}{
		{"JSONL format", FormatJSONL},
		{"Markdown format", FormatMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store, err := NewStore(Config{
				BaseDir: tmpDir,
				Format:  tt.format,
			})
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}

			sessionID := "test-append-" + string(tt.format)

			// Append multiple messages at once
			messages := []llm.Message{
				{Role: llm.RoleUser, Content: "Message 1"},
				{Role: llm.RoleUser, Content: "Message 2"},
				{Role: llm.RoleUser, Content: "Message 3"},
			}

			if err := store.Append(sessionID, messages); err != nil {
				t.Fatalf("Append() error = %v", err)
			}

			// Read and verify
			readMessages, err := store.Read(sessionID)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}

			if len(readMessages) != len(messages) {
				t.Fatalf("Read() returned %d messages, want %d", len(readMessages), len(messages))
			}
		})
	}
}

func TestGetLastN(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("get last N messages", func(t *testing.T) {
		sessionID := "test-lastn-1"

		// Add 10 messages
		for i := 0; i < 10; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: fmt.Sprintf("Message %d", i),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Get last 5
		lastMessages, err := store.GetLastN(sessionID, 5)
		if err != nil {
			t.Fatalf("GetLastN() error = %v", err)
		}

		if len(lastMessages) != 5 {
			t.Fatalf("GetLastN() returned %d messages, want 5", len(lastMessages))
		}

		// Verify they are the last 5
		for i, msg := range lastMessages {
			expectedContent := fmt.Sprintf("Message %d", 5+i)
			if msg.Content != expectedContent {
				t.Errorf("Message %d content = %v, want %v", i, msg.Content, expectedContent)
			}
		}
	})

	t.Run("get last N when N > total", func(t *testing.T) {
		sessionID := "test-lastn-2"

		// Add 3 messages
		for i := 0; i < 3; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: fmt.Sprintf("Message %d", i),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Get last 10 (should return all 3)
		lastMessages, err := store.GetLastN(sessionID, 10)
		if err != nil {
			t.Fatalf("GetLastN() error = %v", err)
		}

		if len(lastMessages) != 3 {
			t.Fatalf("GetLastN() returned %d messages, want 3", len(lastMessages))
		}
	})
}

func TestClear(t *testing.T) {
	tests := []struct {
		name   string
		format Format
	}{
		{"JSONL format", FormatJSONL},
		{"Markdown format", FormatMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store, err := NewStore(Config{
				BaseDir: tmpDir,
				Format:  tt.format,
			})
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}

			sessionID := "test-clear-" + string(tt.format)

			// Add messages
			msg := llm.Message{Role: llm.RoleUser, Content: "Test message"}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify message exists
			messages, err := store.Read(sessionID)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if len(messages) != 1 {
				t.Fatalf("Expected 1 message before clear, got %d", len(messages))
			}

			// Clear
			if err := store.Clear(sessionID); err != nil {
				t.Fatalf("Clear() error = %v", err)
			}

			// Verify session is empty
			messages, err = store.Read(sessionID)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if len(messages) != 0 {
				t.Errorf("After Clear(), Read() returned %d messages, want 0", len(messages))
			}
		})
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("exists after write", func(t *testing.T) {
		sessionID := "test-exists-1"
		msg := llm.Message{Role: llm.RoleUser, Content: "Test"}

		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		if !store.Exists(sessionID) {
			t.Error("Session should exist after write")
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		sessionID := "test-nonexistent"

		if store.Exists(sessionID) {
			t.Error("Non-existent session should not exist")
		}
	})
}

func TestGetSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("get all sessions", func(t *testing.T) {
		// Create multiple sessions
		sessionIDs := []string{"session-1", "session-2", "session-3"}
		for _, id := range sessionIDs {
			msg := llm.Message{Role: llm.RoleUser, Content: "Test"}
			if err := store.Write(id, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		sessions, err := store.GetSessions()
		if err != nil {
			t.Fatalf("GetSessions() error = %v", err)
		}

		if len(sessions) != len(sessionIDs) {
			t.Fatalf("GetSessions() returned %d sessions, want %d", len(sessions), len(sessionIDs))
		}

		// Verify all session IDs are present
		sessionMap := make(map[string]bool)
		for _, id := range sessionIDs {
			sessionMap[id] = false
		}

		for _, id := range sessions {
			if _, exists := sessionMap[id]; !exists {
				t.Errorf("Unexpected session ID: %s", id)
			}
			sessionMap[id] = true
		}

		for id, found := range sessionMap {
			if !found {
				t.Errorf("Session ID not found: %s", id)
			}
		}
	})

	t.Run("empty when no sessions", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir: tmpDir,
			Format:  FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessions, err := store.GetSessions()
		if err != nil {
			t.Fatalf("GetSessions() error = %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("GetSessions() returned %d sessions, want 0", len(sessions))
		}
	})
}

func TestFormatValidation(t *testing.T) {
	t.Run("verify JSONL file format", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir: tmpDir,
			Format:  FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-jsonl-format"
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Message 1"},
			{Role: llm.RoleAssistant, Content: "Message 2"},
		}

		for _, msg := range messages {
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Read file directly
		filePath := filepath.Join(tmpDir, sessionID+".jsonl")
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Verify JSONL format
		lines := strings.Split(string(content), "\n")
		jsonLines := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				if !strings.HasPrefix(strings.TrimSpace(line), "{") {
					t.Errorf("Line is not JSON: %s", line)
				}
				jsonLines++
			}
		}

		if jsonLines != 2 {
			t.Errorf("Expected 2 JSON lines, got %d", jsonLines)
		}
	})

	t.Run("verify Markdown file format", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir: tmpDir,
			Format:  FormatMarkdown,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-md-format"
		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: "Test message",
		}

		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Read file directly
		filePath := filepath.Join(tmpDir, sessionID+".markdown")
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		fileContent := string(content)

		// Verify Markdown format has headers
		if !strings.Contains(fileContent, "#") {
			t.Error("Markdown file should contain headers")
		}

		if !strings.Contains(fileContent, "User") {
			t.Error("Markdown file should contain role information")
		}
	})
}

func TestMultipleSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("different sessions are isolated", func(t *testing.T) {
		session1 := "session-isolated-1"
		session2 := "session-isolated-2"

		// Write to session 1
		msg1 := llm.Message{Role: llm.RoleUser, Content: "Session 1 message"}
		if err := store.Write(session1, msg1); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Write to session 2
		msg2 := llm.Message{Role: llm.RoleUser, Content: "Session 2 message"}
		if err := store.Write(session2, msg2); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Read session 1
		messages1, err := store.Read(session1)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages1) != 1 || messages1[0].Content != "Session 1 message" {
			t.Error("Session 1 should only contain its own messages")
		}

		// Read session 2
		messages2, err := store.Read(session2)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages2) != 1 || messages2[0].Content != "Session 2 message" {
			t.Error("Session 2 should only contain its own messages")
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(Config{
		BaseDir: tmpDir,
		Format:  FormatJSONL,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("concurrent writes to same session", func(t *testing.T) {
		sessionID := "test-concurrent-1"

		// Concurrently write messages
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				msg := llm.Message{
					Role:    llm.RoleUser,
					Content: fmt.Sprintf("Message %d", idx),
				}
				store.Write(sessionID, msg)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all messages were added
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 10 {
			t.Errorf("Expected 10 messages, got %d", len(messages))
		}
	})
}
