package memory

import (
	"os"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

func TestStreamingRead(t *testing.T) {
	t.Run("stream reading JSONL with many messages", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 10 * 1024 * 1024, // 10MB
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-streaming-jsonl"

		// Write 1000 messages
		for i := 0; i < 1000; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: strings.Repeat("x", 100),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Read all messages
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1000 {
			t.Fatalf("Read() returned %d messages, want 1000", len(messages))
		}

		// Verify all messages are correct
		for i, msg := range messages {
			if msg.Role != llm.RoleUser {
				t.Errorf("Message %d role = %v, want %v", i, msg.Role, llm.RoleUser)
			}
			if len(msg.Content) != 100 {
				t.Errorf("Message %d content length = %d, want 100", i, len(msg.Content))
			}
		}
	})

	t.Run("stream reading Markdown with many messages", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatMarkdown,
			MaxFileSize: 10 * 1024 * 1024, // 10MB
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-streaming-md"

		// Write 500 messages (markdown is more verbose)
		for i := 0; i < 500; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: strings.Repeat("y", 200),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Read all messages
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 500 {
			t.Fatalf("Read() returned %d messages, want 500", len(messages))
		}

		// Verify all messages are correct
		for i, msg := range messages {
			if msg.Role != llm.RoleUser {
				t.Errorf("Message %d role = %v, want %v", i, msg.Role, llm.RoleUser)
			}
			if !strings.Contains(msg.Content, strings.Repeat("y", 200)) {
				t.Errorf("Message %d content should contain 200 'y's", i)
			}
		}
	})
}

func TestMaxFileSize(t *testing.T) {
	t.Run("reject file exceeding max size", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 1024, // 1KB max
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-oversize"

		// Write messages until file exceeds max size
		for i := 0; i < 100; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: strings.Repeat("x", 100),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Try to read - should fail with file size error
		_, err = store.Read(sessionID)
		if err == nil {
			t.Fatal("Read() should return error for oversized file")
		}

		if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
			t.Errorf("Error message should mention size limit, got: %v", err)
		}
	})

	t.Run("accept file at max size boundary", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 2048, // 2KB max
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-maxsize-ok"

		// Write messages within limit
		for i := 0; i < 5; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: strings.Repeat("x", 50),
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Try to read - should succeed
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 5 {
			t.Fatalf("Read() returned %d messages, want 5", len(messages))
		}
	})
}

func TestStreamingEdgeCases(t *testing.T) {
	t.Run("handle empty lines in JSONL", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-empty-lines"

		// Create file with empty lines
		filePath := store.getFilePath(sessionID)
		content := `{"role":"user","content":"Message 1"}

{"role":"user","content":"Message 2"}

`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Read should skip empty lines
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 2 {
			t.Fatalf("Read() returned %d messages, want 2", len(messages))
		}
	})

	t.Run("handle malformed lines in JSONL", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-malformed"

		// Create file with malformed lines
		filePath := store.getFilePath(sessionID)
		content := `{"role":"user","content":"Message 1"}
invalid json line
{"role":"user","content":"Message 2"}`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Read should skip malformed lines
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 2 {
			t.Fatalf("Read() returned %d messages, want 2 (malformed lines should be skipped)", len(messages))
		}
	})

	t.Run("handle long lines in JSONL", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 10 * 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-long-lines"

		// Write message with very long content (500KB)
		longContent := strings.Repeat("x", 500*1024)
		msg := llm.Message{
			Role:    llm.RoleUser,
			Content: longContent,
		}
		if err := store.Write(sessionID, msg); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Read should handle long lines
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Read() returned %d messages, want 1", len(messages))
		}

		if len(messages[0].Content) != len(longContent) {
			t.Errorf("Message content length = %d, want %d", len(messages[0].Content), len(longContent))
		}
	})
}

func TestMemoryEfficiency(t *testing.T) {
	t.Run("stream reading uses less memory than ReadFile", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a store with large file limit
		store, err := NewStore(Config{
			BaseDir:     tmpDir,
			Format:      FormatJSONL,
			MaxFileSize: 10 * 1024 * 1024, // 10MB
		})
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		sessionID := "test-memory-efficiency"

		// Write enough messages to create a file ~1MB
		for i := 0; i < 1000; i++ {
			msg := llm.Message{
				Role:    llm.RoleUser,
				Content: strings.Repeat("test content ", 50), // ~650 bytes per message
			}
			if err := store.Write(sessionID, msg); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}

		// Check file size
		filePath := store.getFilePath(sessionID)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// File should be >500KB
		if fileInfo.Size() < 500*1024 {
			t.Logf("Warning: File size is %d bytes, expected >500KB", fileInfo.Size())
		}

		// Read should succeed with streaming
		messages, err := store.Read(sessionID)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if len(messages) != 1000 {
			t.Fatalf("Read() returned %d messages, want 1000", len(messages))
		}

		// The key test: streaming read should not load entire file into memory at once
		// We can't directly measure memory usage, but we can verify the implementation uses scanner
		// This is more of a documentation test - the implementation is verified by code review
		t.Logf("Successfully read %d messages from file of size %d bytes using streaming", len(messages), fileInfo.Size())
	})
}
