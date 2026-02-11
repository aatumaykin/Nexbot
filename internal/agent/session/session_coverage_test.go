package session

import (
	"os"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

// TestManagerExists tests the Manager.Exists method
func TestManagerExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("non-existing session", func(t *testing.T) {
		exists, err := mgr.Exists("non-existing-session")
		if err != nil {
			t.Errorf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() should return false for non-existing session")
		}
	})

	t.Run("existing session", func(t *testing.T) {
		sessionID := "test-exists-session"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Verify session file exists
		if _, err := os.Stat(session.File); err != nil {
			t.Fatalf("Session file should exist: %v", err)
		}

		exists, err := mgr.Exists(sessionID)
		if err != nil {
			t.Errorf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() should return true for existing session")
		}
	})
}

// TestSessionMessageCount tests the Session.MessageCount method
func TestSessionMessageCount(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("empty session", func(t *testing.T) {
		sessionID := "test-count-empty"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		count, err := session.MessageCount()
		if err != nil {
			t.Errorf("MessageCount() error = %v", err)
		}
		if count != 0 {
			t.Errorf("MessageCount() = %d, want 0", count)
		}
	})

	t.Run("session with one message", func(t *testing.T) {
		sessionID := "test-count-one"
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

		count, err := session.MessageCount()
		if err != nil {
			t.Errorf("MessageCount() error = %v", err)
		}
		if count != 1 {
			t.Errorf("MessageCount() = %d, want 1", count)
		}
	})

	t.Run("session with multiple messages", func(t *testing.T) {
		sessionID := "test-count-multiple"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Message 1"},
			{Role: llm.RoleAssistant, Content: "Response 1"},
			{Role: llm.RoleUser, Content: "Message 2"},
			{Role: llm.RoleAssistant, Content: "Response 2"},
			{Role: llm.RoleUser, Content: "Message 3"},
		}

		for _, msg := range messages {
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		count, err := session.MessageCount()
		if err != nil {
			t.Errorf("MessageCount() error = %v", err)
		}
		if count != 5 {
			t.Errorf("MessageCount() = %d, want 5", count)
		}
	})

	t.Run("session count after clear", func(t *testing.T) {
		sessionID := "test-count-after-clear"
		session, _, err := mgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add messages
		for range 3 {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: "Message",
			}
			if err := session.Append(msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Verify count
		count, err := session.MessageCount()
		if err != nil {
			t.Errorf("MessageCount() error = %v", err)
		}
		if count != 3 {
			t.Errorf("MessageCount() before clear = %d, want 3", count)
		}

		// Clear session
		if err := session.Clear(); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		// Verify count is zero
		count, err = session.MessageCount()
		if err != nil {
			t.Errorf("MessageCount() error = %v", err)
		}
		if count != 0 {
			t.Errorf("MessageCount() after clear = %d, want 0", count)
		}
	})
}
