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

	t.Run("non-existent workspace", func(t *testing.T) {
		_, err := NewBuilder(Config{
			Workspace: "/non/existent/path",
		})
		if err == nil {
			t.Error("NewBuilder() should return error for non-existent workspace")
		}
	})
}

func TestBuild(t *testing.T) {
	t.Run("build with all components", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create context files
		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("# Identity\nTest identity"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agents\nTest agents"), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "SOUL.md"), []byte("# Soul\nTest soul"), 0644); err != nil {
			t.Fatalf("Failed to create SOUL.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "USER.md"), []byte("# User\nTest user"), 0644); err != nil {
			t.Fatalf("Failed to create USER.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "TOOLS.md"), []byte("# Tools\nTest tools"), 0644); err != nil {
			t.Fatalf("Failed to create TOOLS.md: %v", err)
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

		// Verify all components are present
		if !strings.Contains(prompt, "Test identity") {
			t.Error("Build() should contain identity")
		}
		if !strings.Contains(prompt, "Test agents") {
			t.Error("Build() should contain agents")
		}
		if !strings.Contains(prompt, "Test soul") {
			t.Error("Build() should contain soul")
		}
		if !strings.Contains(prompt, "Test user") {
			t.Error("Build() should contain user")
		}
		if !strings.Contains(prompt, "Test tools") {
			t.Error("Build() should contain tools")
		}

		// Verify order: IDENTITY should come before AGENTS
		identityPos := strings.Index(prompt, "Test identity")
		agentsPos := strings.Index(prompt, "Test agents")
		if identityPos == -1 || agentsPos == -1 {
			t.Fatal("Components not found")
		}
		if identityPos > agentsPos {
			t.Error("IDENTITY should come before AGENTS")
		}
	})

	t.Run("build with partial components", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Only create IDENTITY and USER
		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("# Identity\nTest identity"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "USER.md"), []byte("# User\nTest user"), 0644); err != nil {
			t.Fatalf("Failed to create USER.md: %v", err)
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

		// Verify present components
		if !strings.Contains(prompt, "Test identity") {
			t.Error("Build() should contain identity")
		}
		if !strings.Contains(prompt, "Test user") {
			t.Error("Build() should contain user")
		}

		// Verify absent components don't cause errors
		// Build should still succeed
	})

	t.Run("build with no components", func(t *testing.T) {
		tmpDir := t.TempDir()

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

		// Build should succeed even with no components, returning empty string
		// This is expected behavior
		if prompt != "" {
			t.Logf("Build() returned prompt: %s", prompt)
		}
	})
}

func TestBuildWithMemory(t *testing.T) {
	t.Run("build with memory messages", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create context files
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

		// Verify memory section is present
		if !strings.Contains(prompt, "Recent Conversation Memory") {
			t.Error("BuildWithMemory() should contain memory section")
		}
		if !strings.Contains(prompt, "User message") {
			t.Error("BuildWithMemory() should contain user message")
		}
		if !strings.Contains(prompt, "Assistant response") {
			t.Error("BuildWithMemory() should contain assistant message")
		}
	})

	t.Run("build with empty memory", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		prompt, err := builder.BuildWithMemory([]llm.Message{})
		if err != nil {
			t.Fatalf("BuildWithMemory() error = %v", err)
		}

		// Empty memory should not add memory section
		if strings.Contains(prompt, "Recent Conversation Memory") {
			t.Error("BuildWithMemory() should not add memory section for empty messages")
		}
	})

	t.Run("build with tool messages in memory", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "User message"},
			{Role: llm.RoleTool, ToolCallID: "call_123", Content: "Tool result"},
			{Role: llm.RoleAssistant, Content: "Assistant response"},
		}

		prompt, err := builder.BuildWithMemory(messages)
		if err != nil {
			t.Fatalf("BuildWithMemory() error = %v", err)
		}

		// Verify tool message is included
		if !strings.Contains(prompt, "Tool result") {
			t.Error("BuildWithMemory() should contain tool message")
		}
		if !strings.Contains(prompt, "call_123") {
			t.Error("BuildWithMemory() should contain tool call ID")
		}
	})
}

func TestBuildForSession(t *testing.T) {
	t.Run("build with session ID", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		sessionID := "test-session-123"
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Test message"},
		}

		prompt, err := builder.BuildForSession(sessionID, messages)
		if err != nil {
			t.Fatalf("BuildForSession() error = %v", err)
		}

		// Verify session header is present
		if !strings.Contains(prompt, "Session: test-session-123") {
			t.Error("BuildForSession() should contain session ID")
		}
	})

	t.Run("build for session without memory", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		sessionID := "test-session-456"
		prompt, err := builder.BuildForSession(sessionID, []llm.Message{})
		if err != nil {
			t.Fatalf("BuildForSession() error = %v", err)
		}

		// Verify session header is present
		if !strings.Contains(prompt, "Session: test-session-456") {
			t.Error("BuildForSession() should contain session ID")
		}
	})
}

func TestReadMemory(t *testing.T) {
	t.Run("read memory files from workspace", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create memory directory and files
		memoryDir := filepath.Join(tmpDir, "memory")
		if err := os.MkdirAll(memoryDir, 0755); err != nil {
			t.Fatalf("Failed to create memory directory: %v", err)
		}

		if err := os.WriteFile(filepath.Join(memoryDir, "test1.md"), []byte("Memory content 1"), 0644); err != nil {
			t.Fatalf("Failed to create test1.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(memoryDir, "test2.md"), []byte("Memory content 2"), 0644); err != nil {
			t.Fatalf("Failed to create test2.md: %v", err)
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

		if len(messages) != 2 {
			t.Fatalf("ReadMemory() returned %d messages, want 2", len(messages))
		}

		// Verify messages are system messages
		for _, msg := range messages {
			if msg.Role != llm.RoleSystem {
				t.Errorf("Memory messages should have RoleSystem, got %v", msg.Role)
			}
		}
	})

	t.Run("read memory with non-markdown files", func(t *testing.T) {
		tmpDir := t.TempDir()

		memoryDir := filepath.Join(tmpDir, "memory")
		if err := os.MkdirAll(memoryDir, 0755); err != nil {
			t.Fatalf("Failed to create memory directory: %v", err)
		}

		// Create markdown and non-markdown files
		if err := os.WriteFile(filepath.Join(memoryDir, "test.md"), []byte("Memory content"), 0644); err != nil {
			t.Fatalf("Failed to create test.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(memoryDir, "test.txt"), []byte("Non-markdown content"), 0644); err != nil {
			t.Fatalf("Failed to create test.txt: %v", err)
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

		// Should only read markdown files
		if len(messages) != 1 {
			t.Fatalf("ReadMemory() returned %d messages, want 1 (only markdown)", len(messages))
		}
	})

	t.Run("read memory when directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

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

		if len(messages) != 0 {
			t.Fatalf("ReadMemory() returned %d messages, want 0", len(messages))
		}
	})
}

func TestProcessTemplates(t *testing.T) {
	t.Run("process workspace variables", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("Workspace: {{WORKSPACE_PATH}}"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
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

		if !strings.Contains(prompt, tmpDir) {
			t.Errorf("Template variable WORKSPACE_PATH should be replaced with %s", tmpDir)
		}
	})

	t.Run("process time variables", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("Time: {{CURRENT_TIME}}\nDate: {{CURRENT_DATE}}"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
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

		// Variables should be replaced (even if empty)
		if strings.Contains(prompt, "{{CURRENT_TIME}}") {
			t.Error("Template variable should be replaced")
		}
		if strings.Contains(prompt, "{{CURRENT_DATE}}") {
			t.Error("Template variable should be replaced")
		}
	})
}

func TestGetComponent(t *testing.T) {
	t.Run("get identity component", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "IDENTITY.md"), []byte("# Identity\nTest content"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		component, err := builder.GetComponent("IDENTITY")
		if err != nil {
			t.Fatalf("GetComponent() error = %v", err)
		}

		if !strings.Contains(component, "Test content") {
			t.Error("GetComponent() should return identity content")
		}
	})

	t.Run("get non-existent component", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		_, err = builder.GetComponent("IDENTITY")
		if err == nil {
			t.Error("GetComponent() should return error for non-existent component")
		}
	})

	t.Run("get unknown component", func(t *testing.T) {
		tmpDir := t.TempDir()

		builder, err := NewBuilder(Config{
			Workspace: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		_, err = builder.GetComponent("UNKNOWN")
		if err == nil {
			t.Error("GetComponent() should return error for unknown component")
		}
	})
}

func TestGetWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	builder, err := NewBuilder(Config{
		Workspace: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	if builder.GetWorkspace() != tmpDir {
		t.Errorf("GetWorkspace() = %v, want %v", builder.GetWorkspace(), tmpDir)
	}
}

func TestPriorityOrder(t *testing.T) {
	t.Run("verify component priority order", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create all components with unique markers
		components := map[string]string{
			"IDENTITY": "MARKER_IDENTITY_1",
			"AGENTS":   "MARKER_AGENTS_2",
			"SOUL":     "MARKER_SOUL_3",
			"USER":     "MARKER_USER_4",
			"TOOLS":    "MARKER_TOOLS_5",
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

		// Verify order: IDENTITY → AGENTS → SOUL → USER → TOOLS
		positions := make(map[string]int)
		for name, marker := range components {
			pos := strings.Index(prompt, marker)
			if pos == -1 {
				t.Fatalf("Marker for %s not found in prompt", name)
			}
			positions[name] = pos
		}

		// Check priority order
		if positions["IDENTITY"] > positions["AGENTS"] {
			t.Error("IDENTITY should come before AGENTS")
		}
		if positions["AGENTS"] > positions["SOUL"] {
			t.Error("AGENTS should come before SOUL")
		}
		if positions["SOUL"] > positions["USER"] {
			t.Error("SOUL should come before USER")
		}
		if positions["USER"] > positions["TOOLS"] {
			t.Error("USER should come before TOOLS")
		}
	})
}
