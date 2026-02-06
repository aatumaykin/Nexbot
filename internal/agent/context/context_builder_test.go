package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/llm"
)

func TestNewBuilder(t *testing.T) {
	t.Run("valid workspace", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("NewBuilder() error = %v", err)
		}

		if builder == nil {
			t.Error("NewBuilder() returned nil builder")
			return
		}

		if builder.workspace != tmpDir {
			t.Errorf("Builder.workspace = %v, want %v", builder.workspace, tmpDir)
		}
	})

	t.Run("empty workspace", func(t *testing.T) {
		_, err := NewBuilder(Config{
			Workspace: "",
		})
		if err == nil {
			t.Error("NewBuilder() should return error for empty workspace")
		}
	})
}

func TestBuild(t *testing.T) {
	t.Run("build with all components", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("# Identity\nTest identity"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agents\nTest agents"), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		prompt, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if !strings.Contains(prompt, "Test identity") {
			t.Error("Build() should contain identity")
		}
		if !strings.Contains(prompt, "Test agents") {
			t.Error("Build() should contain agents")
		}

		agentsPos := strings.Index(prompt, "Test agents")
		identityPos := strings.Index(prompt, "Test identity")
		if agentsPos == -1 || identityPos == -1 {
			t.Fatal("Components not found")
		}
		if agentsPos > identityPos {
			t.Error("AGENTS should come before IDENTITY")
		}
	})
}

func TestBuildWithMemory(t *testing.T) {
	t.Run("build with memory messages", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("# Identity\nTest"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "User message"},
			{Role: llm.RoleAssistant, Content: "Assistant response"},
		}

		prompt, err := builder.BuildWithMemory(messages)
		if err != nil {
			t.Fatalf("BuildWithMemory() error = %v", err)
		}

		if !strings.Contains(prompt, "Recent Conversation Memory") {
			t.Error("BuildWithMemory() should contain memory section")
		}
		if !strings.Contains(prompt, "User message") {
			t.Error("BuildWithMemory() should contain user message")
		}
	})
}

func TestReadMemory(t *testing.T) {
	t.Run("read memory files from workspace", func(t *testing.T) {
		tmpDir := t.TempDir()

		memoryDir := filepath.Join(tmpDir, "memory")
		if err := os.MkdirAll(memoryDir, 0755); err != nil {
			t.Fatalf("Failed to create memory directory: %v", err)
		}

		if err := os.WriteFile(filepath.Join(memoryDir, "test1.md"), []byte("Memory content 1"), 0644); err != nil {
			t.Fatalf("Failed to create test1.md: %v", err)
		}

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		messages, err := builder.ReadMemory()
		if err != nil {
			t.Fatalf("ReadMemory() error = %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("ReadMemory() returned %d messages, want 1", len(messages))
		}

		for _, msg := range messages {
			if msg.Role != llm.RoleSystem {
				t.Errorf("Memory messages should have RoleSystem, got %v", msg.Role)
			}
		}
	})
}

func TestPriorityOrder(t *testing.T) {
	t.Run("verify component priority order", func(t *testing.T) {
		tmpDir := t.TempDir()

		components := map[string]string{
			"IDENTITY": "MARKER_IDENTITY_2",
			"AGENTS":   "MARKER_AGENTS_1",
			"USER":     "MARKER_USER_3",
			"TOOLS":    "MARKER_TOOLS_4",
		}

		for name, content := range components {
			if err := os.WriteFile(filepath.Join(tmpDir, name+".md"), []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create %s.md: %v", name, err)
			}
		}

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		prompt, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		positions := make(map[string]int)
		for name, marker := range components {
			pos := strings.Index(prompt, marker)
			if pos == -1 {
				t.Fatalf("Marker for %s not found in prompt", name)
			}
			positions[name] = pos
		}

		if positions["AGENTS"] > positions["IDENTITY"] {
			t.Error("AGENTS should come before IDENTITY")
		}
		if positions["IDENTITY"] > positions["USER"] {
			t.Error("IDENTITY should come before USER")
		}
	})
}
