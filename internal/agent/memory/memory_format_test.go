package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

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
		filePath := filepath.Join(tmpDir, sessionID+store.format.FileExtension())
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
