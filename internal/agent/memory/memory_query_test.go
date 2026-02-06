package memory

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

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
