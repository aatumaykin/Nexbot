package memory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

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
		for i := range 10 {
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
		for i := range 3 {
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
