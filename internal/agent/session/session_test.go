package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "valid directory",
			baseDir: "/tmp/test-session-manager",
			wantErr: false,
		},
		{
			name:    "empty base directory",
			baseDir: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid directory" {
				defer os.RemoveAll(tt.baseDir)
			}

			mgr, err := NewManager(tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && mgr == nil {
				t.Error("NewManager() returned nil manager")
			}

			if !tt.wantErr && mgr.baseDir != tt.baseDir {
				t.Errorf("NewManager() baseDir = %v, want %v", mgr.baseDir, tt.baseDir)
			}
		})
	}
}

func TestGetOrCreate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("create new session", func(t *testing.T) {
		sessionID := "test-session-1"
		session, created, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("GetOrCreate() error = %v", err)
		}

		if !created {
			t.Error("GetOrCreate() should have created new session")
		}

		if session.ID != sessionID {
			t.Errorf("Session.ID = %v, want %v", session.ID, sessionID)
		}

		if !session.Exists() {
			t.Error("Session file should exist after creation")
		}
	})

	t.Run("get existing session", func(t *testing.T) {
		sessionID := "test-session-2"

		// Create session first
		_, created, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		if !created {
			t.Error("First call should create session")
		}

		// Get existing session
		session, created, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("GetOrCreate() error = %v", err)
		}

		if created {
			t.Error("GetOrCreate() should not have created new session")
		}

		if session.ID != sessionID {
			t.Errorf("Session.ID = %v, want %v", session.ID, sessionID)
		}
	})

	t.Run("multiple different sessions", func(t *testing.T) {
		sessionID1 := "test-session-3a"
		sessionID2 := "test-session-3b"

		session1, created1, err := mgr.GetOrCreate(sessionID1)
		if err != nil {
			t.Fatalf("Failed to create session1: %v", err)
		}

		session2, created2, err := mgr.GetOrCreate(sessionID2)
		if err != nil {
			t.Fatalf("Failed to create session2: %v", err)
		}

		if !created1 || !created2 {
			t.Error("Both sessions should be created")
		}

		if session1.File == session2.File {
			t.Error("Different sessions should have different files")
		}
	})
}

func TestAppend(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("append single message", func(t *testing.T) {
		sessionID := "test-append-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: "Hello, world!",
		}

		if err := session.Append(msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		messages, err := session.Read()
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
			t.Errorf("Message content = %v, want %v", messages[0].Content, "Hello, world!")
		}
	})

	t.Run("append multiple messages", func(t *testing.T) {
		sessionID := "test-append-2"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are helpful assistant"},
			{Role: llm.RoleUser, Content: "What is 2+2?"},
			{Role: llm.RoleAssistant, Content: "2+2=4"},
		}

		for _, msg := range messages {
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		readMessages, err := session.Read()
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

	t.Run("append with tool call", func(t *testing.T) {
		sessionID := "test-append-3"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		msg := llm.Message{
			Role:       llm.RoleTool,
			Content:    "Tool result",
			ToolCallID: "call_123",
		}

		if err := session.Append(msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		messages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if messages[0].Role != llm.RoleTool {
			t.Errorf("Message role = %v, want %v", messages[0].Role, llm.RoleTool)
		}

		if messages[0].ToolCallID != "call_123" {
			t.Errorf("ToolCallID = %v, want %v", messages[0].ToolCallID, "call_123")
		}
	})
}

func TestRead(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("read empty session", func(t *testing.T) {
		sessionID := "test-read-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		messages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 0 {
			t.Errorf("Read() returned %d messages, want 0", len(messages))
		}
	})

	t.Run("read non-existent session", func(t *testing.T) {
		sessionID := "test-read-nonexistent"
		session := &Session{
			ID:   sessionID,
			File: filepath.Join(tmpDir, sessionID+".jsonl"),
		}

		_, err := session.Read()
		if err == nil {
			t.Error("Read() should return error for non-existent file")
		}
	})

	t.Run("read after multiple appends", func(t *testing.T) {
		sessionID := "test-read-2"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add messages
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Message 1"},
			{Role: llm.RoleUser, Content: "Message 2"},
			{Role: llm.RoleUser, Content: "Message 3"},
		}

		for _, msg := range messages {
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Read messages
		readMessages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(readMessages) != len(messages) {
			t.Fatalf("Read() returned %d messages, want %d", len(readMessages), len(messages))
		}
	})
}

func TestMessageCount(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("empty session count", func(t *testing.T) {
		sessionID := "test-count-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		count, err := session.MessageCount()
		if err != nil {
			t.Fatalf("MessageCount() error = %v", err)
		}

		if count != 0 {
			t.Errorf("MessageCount() = %d, want 0", count)
		}
	})

	t.Run("session with messages count", func(t *testing.T) {
		sessionID := "test-count-2"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add 5 messages
		for i := 0; i < 5; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: fmt.Sprintf("Message %d", i+1),
			}
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		count, err := session.MessageCount()
		if err != nil {
			t.Fatalf("MessageCount() error = %v", err)
		}

		if count != 5 {
			t.Errorf("MessageCount() = %d, want 5", count)
		}
	})
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("clear session with messages", func(t *testing.T) {
		sessionID := "test-clear-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add messages
		msg := llm.Message{Role: llm.RoleUser, Content: "Test message"}
		if err := session.Append(msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Verify message exists
		messages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message before clear, got %d", len(messages))
		}

		// Clear session
		if err := session.Clear(); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		// Verify session is empty
		messages, err = session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("After Clear(), Read() returned %d messages, want 0", len(messages))
		}
	})
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("delete existing session", func(t *testing.T) {
		sessionID := "test-delete-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Verify session exists
		if !session.Exists() {
			t.Error("Session should exist before delete")
		}

		// Delete session
		if err := session.Delete(); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify session is deleted
		if session.Exists() {
			t.Error("Session should not exist after delete")
		}
	})

	t.Run("delete already deleted session", func(t *testing.T) {
		sessionID := "test-delete-2"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Delete session
		if err := session.Delete(); err != nil {
			t.Fatalf("First Delete() error = %v", err)
		}

		// Delete again should not error
		if err := session.Delete(); err != nil {
			t.Errorf("Second Delete() error = %v", err)
		}
	})
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("session exists after creation", func(t *testing.T) {
		sessionID := "test-exists-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if !session.Exists() {
			t.Error("Session should exist after creation")
		}
	})

	t.Run("session does not exist", func(t *testing.T) {
		session := &Session{
			ID:   "nonexistent-session",
			File: filepath.Join(tmpDir, "nonexistent-session.jsonl"),
		}

		if session.Exists() {
			t.Error("Non-existent session should not exist")
		}
	})
}

func TestJSONLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("verify JSONL format", func(t *testing.T) {
		sessionID := "test-jsonl-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add messages
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "User message"},
			{Role: llm.RoleAssistant, Content: "Assistant response"},
		}

		for _, msg := range messages {
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Read file directly
		content, err := os.ReadFile(session.File)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Verify JSONL format (one JSON object per line)
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
}

func TestConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("concurrent appends", func(t *testing.T) {
		sessionID := "test-concurrent-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Concurrently append messages
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				msg := llm.Message{
					Role:    llm.RoleUser,
					Content: fmt.Sprintf("Message %d", idx),
				}
				session.Append(msg)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all messages were added
		messages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 10 {
			t.Errorf("Expected 10 messages, got %d", len(messages))
		}
	})
}
