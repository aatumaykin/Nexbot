package session

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

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

func TestTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("timestamp is saved with message", func(t *testing.T) {
		sessionID := "test-timestamp-1"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: "Test message with timestamp",
		}

		if err := session.Append(msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Read file content to verify timestamp
		content, err := os.ReadFile(session.File)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Parse the entry to verify timestamp
		var entry Entry
		lines := strings.Split(string(content), "\n")
		if len(lines) < 1 || strings.TrimSpace(lines[0]) == "" {
			t.Fatal("File is empty")
		}

		if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
			t.Fatalf("Failed to unmarshal entry: %v", err)
		}

		if entry.Timestamp == "" {
			t.Error("Timestamp should not be empty")
		}

		// Verify timestamp is in RFC3339 format
		if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
			t.Errorf("Timestamp is not in RFC3339 format: %v, error: %v", entry.Timestamp, err)
		}
	})
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

		_, created, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		if !created {
			t.Error("First call should create session")
		}

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
			t.Errorf("Message content = %v, want 'Hello, world!'", messages[0].Content)
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

		msg := llm.Message{Role: llm.RoleUser, Content: "Test message"}
		if err := session.Append(msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		messages, err := session.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message before clear, got %d", len(messages))
		}

		if err := session.Clear(); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

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

		if !session.Exists() {
			t.Error("Session should exist before delete")
		}

		if err := session.Delete(); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if session.Exists() {
			t.Error("Session should not exist after delete")
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

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "User message"},
			{Role: llm.RoleAssistant, Content: "Assistant response"},
		}

		for _, msg := range messages {
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		content, err := os.ReadFile(session.File)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

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
