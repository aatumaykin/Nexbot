package memory

import (
	"fmt"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

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
		errChan := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				msg := llm.Message{
					Role:    llm.RoleUser,
					Content: fmt.Sprintf("Message %d", idx),
				}
				errChan <- store.Write(sessionID, msg)
			}(i)
		}

		// Wait for all goroutines and check errors
		for i := 0; i < 10; i++ {
			if err := <-errChan; err != nil {
				t.Fatalf("Write() error = %v", err)
			}
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
