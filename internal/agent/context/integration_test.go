package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent/memory"
	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/llm"
)

// TestFullContextWorkflow tests the complete workflow: Session → Memory → Context
func TestFullContextWorkflow(t *testing.T) {
	t.Run("complete workflow with all components", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create workspace structure
		workspaceDir := filepath.Join(tmpDir, "workspace")
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		// Create context files
		if err := os.WriteFile(filepath.Join(workspaceDir, "IDENTITY.md"), []byte("# Identity\nNexbot Assistant"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(workspaceDir, "AGENTS.md"), []byte("# Agents\nBe helpful"), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(workspaceDir, "TOOLS.md"), []byte("# Tools\nFile ops, Shell commands"), 0644); err != nil {
			t.Fatalf("Failed to create TOOLS.md: %v", err)
		}

		// Create session manager
		sessionDir := filepath.Join(tmpDir, "sessions")
		sessionMgr, err := session.NewManager(sessionDir)
		if err != nil {
			t.Fatalf("Failed to create session manager: %v", err)
		}

		// Create memory store
		memoryDir := filepath.Join(tmpDir, "memory")
		memStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create memory store: %v", err)
		}

		// Create context builder
		ctxBuilder, err := NewBuilder(Config{
			Workspace: workspaceDir,
		})
		if err != nil {
			t.Fatalf("Failed to create context builder: %v", err)
		}

		// Workflow: Create session
		sessionID := "integration-test-1"
		sess, created, err := sessionMgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to get/create session: %v", err)
		}
		if !created {
			t.Error("Session should be newly created")
		}

		// Workflow: Add messages to session
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Hello, how are you?"},
			{Role: llm.RoleAssistant, Content: "I'm doing well, thank you!"},
			{Role: llm.RoleUser, Content: "What can you do?"},
		}

		for _, msg := range messages {
			if err := sess.Append(msg); err != nil {
				t.Fatalf("Failed to append message to session: %v", err)
			}
		}

		// Workflow: Store messages in memory
		for _, msg := range messages {
			if err := memStore.Write(sessionID, msg); err != nil {
				t.Fatalf("Failed to write to memory: %v", err)
			}
		}

		// Workflow: Read from memory
		readMessages, err := memStore.Read(sessionID)
		if err != nil {
			t.Fatalf("Failed to read from memory: %v", err)
		}

		if len(readMessages) != len(messages) {
			t.Fatalf("Expected %d messages, got %d", len(messages), len(readMessages))
		}

		// Workflow: Build context with memory
		contextWithMemory, err := ctxBuilder.BuildWithMemory(readMessages)
		if err != nil {
			t.Fatalf("Failed to build context with memory: %v", err)
		}

		// Verify context contains all components
		if !strings.Contains(contextWithMemory, "Nexbot Assistant") {
			t.Error("Context should contain identity")
		}
		if !strings.Contains(contextWithMemory, "Be helpful") {
			t.Error("Context should contain agents")
		}
		if !strings.Contains(contextWithMemory, "Hello, how are you?") {
			t.Error("Context should contain user message from memory")
		}
	})
}

// TestSessionMemoryIntegration tests integration between Session and Memory stores
func TestSessionMemoryIntegration(t *testing.T) {
	t.Run("sync session to memory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create session manager and memory store
		sessionDir := filepath.Join(tmpDir, "sessions")
		sessionMgr, err := session.NewManager(sessionDir)
		if err != nil {
			t.Fatalf("Failed to create session manager: %v", err)
		}

		memoryDir := filepath.Join(tmpDir, "memory")
		memStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create memory store: %v", err)
		}

		// Create session and add messages
		sessionID := "sync-test-1"
		sess, _, err := sessionMgr.GetOrCreate(sessionID)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add messages to session
		sessionMessages := []llm.Message{
			{Role: llm.RoleUser, Content: "First message"},
			{Role: llm.RoleAssistant, Content: "First response"},
			{Role: llm.RoleUser, Content: "Second message"},
		}

		for _, msg := range sessionMessages {
			if err := sess.Append(msg); err != nil {
				t.Fatalf("Failed to append to session: %v", err)
			}
		}

		// Sync to memory
		for _, msg := range sessionMessages {
			if err := memStore.Write(sessionID, msg); err != nil {
				t.Fatalf("Failed to write to memory: %v", err)
			}
		}

		// Verify consistency
		sessionMsgs, err := sess.Read()
		if err != nil {
			t.Fatalf("Failed to read session: %v", err)
		}

		memoryMsgs, err := memStore.Read(sessionID)
		if err != nil {
			t.Fatalf("Failed to read memory: %v", err)
		}

		if len(sessionMsgs) != len(memoryMsgs) {
			t.Errorf("Session has %d messages, memory has %d", len(sessionMsgs), len(memoryMsgs))
		}

		// Verify messages match
		for i := range sessionMsgs {
			if sessionMsgs[i].Role != memoryMsgs[i].Role {
				t.Errorf("Message %d role mismatch: session=%v, memory=%v", i, sessionMsgs[i].Role, memoryMsgs[i].Role)
			}
			if sessionMsgs[i].Content != memoryMsgs[i].Content {
				t.Errorf("Message %d content mismatch: session=%v, memory=%v", i, sessionMsgs[i].Content, memoryMsgs[i].Content)
			}
		}
	})
}

// TestContextBuilderWithMemory tests Context builder with messages from Memory
func TestContextBuilderWithMemory(t *testing.T) {
	t.Run("build context with recent memory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create workspace
		workspaceDir := filepath.Join(tmpDir, "workspace")
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		if err := os.WriteFile(filepath.Join(workspaceDir, "IDENTITY.md"), []byte("# Identity\nTest Assistant"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}

		// Create memory store with messages
		memoryDir := filepath.Join(tmpDir, "memory")
		memStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create memory store: %v", err)
		}

		// Add conversation history
		sessionID := "context-test-1"
		history := []llm.Message{
			{Role: llm.RoleUser, Content: "User: I need help with Go"},
			{Role: llm.RoleAssistant, Content: "Assistant: I'd be happy to help with Go!"},
			{Role: llm.RoleUser, Content: "User: How do I write a struct?"},
			{Role: llm.RoleAssistant, Content: "Assistant: Here's how..."},
		}

		for _, msg := range history {
			if err := memStore.Write(sessionID, msg); err != nil {
				t.Fatalf("Failed to write to memory: %v", err)
			}
		}

		// Get recent messages (last 2)
		recent, err := memStore.GetLastN(sessionID, 2)
		if err != nil {
			t.Fatalf("Failed to get recent messages: %v", err)
		}

		// Build context with recent memory
		ctxBuilder, err := NewBuilder(Config{
			Workspace: workspaceDir,
		})
		if err != nil {
			t.Fatalf("Failed to create context builder: %v", err)
		}

		context, err := ctxBuilder.BuildWithMemory(recent)
		if err != nil {
			t.Fatalf("Failed to build context: %v", err)
		}

		// Verify context contains identity
		if !strings.Contains(context, "Test Assistant") {
			t.Error("Context should contain identity")
		}

		// Verify context contains recent messages
		if !strings.Contains(context, "How do I write a struct?") {
			t.Error("Context should contain recent user message")
		}

		// Verify context doesn't contain old messages (first message should not be present)
		// GetLastN(2) should return only last 2 messages
		if strings.Contains(context, "I need help with Go") {
			t.Error("Context should not contain old messages beyond GetLastN limit")
		}
	})
}

// TestMultipleSessionsWithSameContext tests handling multiple sessions with same context
func TestMultipleSessionsWithSameContext(t *testing.T) {
	t.Run("multiple sessions independent contexts", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create workspace
		workspaceDir := filepath.Join(tmpDir, "workspace")
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		if err := os.WriteFile(filepath.Join(workspaceDir, "IDENTITY.md"), []byte("# Identity\nShared Assistant"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}

		// Create session manager
		sessionDir := filepath.Join(tmpDir, "sessions")
		sessionMgr, err := session.NewManager(sessionDir)
		if err != nil {
			t.Fatalf("Failed to create session manager: %v", err)
		}

		// Create memory store
		memoryDir := filepath.Join(tmpDir, "memory")
		memStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create memory store: %v", err)
		}

		// Create context builder
		ctxBuilder, err := NewBuilder(Config{
			Workspace: workspaceDir,
		})
		if err != nil {
			t.Fatalf("Failed to create context builder: %v", err)
		}

		// Session 1: Python discussion
		sessionID1 := "python-session"
		sess1, _, err := sessionMgr.GetOrCreate(sessionID1)
		if err != nil {
			t.Fatalf("Failed to create session 1: %v", err)
		}

		msgs1 := []llm.Message{
			{Role: llm.RoleUser, Content: "Help with Python lists"},
			{Role: llm.RoleAssistant, Content: "Here's how to use lists..."},
		}

		for _, msg := range msgs1 {
			sess1.Append(msg)
			memStore.Write(sessionID1, msg)
		}

		// Session 2: JavaScript discussion
		sessionID2 := "js-session"
		sess2, _, err := sessionMgr.GetOrCreate(sessionID2)
		if err != nil {
			t.Fatalf("Failed to create session 2: %v", err)
		}

		msgs2 := []llm.Message{
			{Role: llm.RoleUser, Content: "Help with JavaScript arrays"},
			{Role: llm.RoleAssistant, Content: "Here's how to use arrays..."},
		}

		for _, msg := range msgs2 {
			sess2.Append(msg)
			memStore.Write(sessionID2, msg)
		}

		// Build contexts for each session
		memory1, _ := memStore.Read(sessionID1)
		memory2, _ := memStore.Read(sessionID2)

		context1, err := ctxBuilder.BuildForSession(sessionID1, memory1)
		if err != nil {
			t.Fatalf("Failed to build context 1: %v", err)
		}

		context2, err := ctxBuilder.BuildForSession(sessionID2, memory2)
		if err != nil {
			t.Fatalf("Failed to build context 2: %v", err)
		}

		// Verify contexts are different
		if !strings.Contains(context1, "Session: python-session") {
			t.Error("Context 1 should have session ID")
		}
		if !strings.Contains(context2, "Session: js-session") {
			t.Error("Context 2 should have session ID")
		}

		// Verify contexts contain only their respective messages
		if !strings.Contains(context1, "Python lists") {
			t.Error("Context 1 should contain Python messages")
		}
		if strings.Contains(context1, "JavaScript arrays") {
			t.Error("Context 1 should not contain JavaScript messages")
		}

		if !strings.Contains(context2, "JavaScript arrays") {
			t.Error("Context 2 should contain JavaScript messages")
		}
		if strings.Contains(context2, "Python lists") {
			t.Error("Context 2 should not contain Python messages")
		}

		// Verify both share the same identity
		if !strings.Contains(context1, "Shared Assistant") {
			t.Error("Context 1 should contain shared identity")
		}
		if !strings.Contains(context2, "Shared Assistant") {
			t.Error("Context 2 should contain shared identity")
		}
	})
}

// TestMemoryFormatsIntegration tests different memory formats with context builder
func TestMemoryFormatsIntegration(t *testing.T) {
	t.Run("JSONL and Markdown formats", func(t *testing.T) {
		tmpDir := t.TempDir()

		workspaceDir := filepath.Join(tmpDir, "workspace")
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		if err := os.WriteFile(filepath.Join(workspaceDir, "IDENTITY.md"), []byte("# Identity\nTest"), 0644); err != nil {
			t.Fatalf("Failed to create IDENTITY.md: %v", err)
		}

		ctxBuilder, err := NewBuilder(Config{
			Workspace: workspaceDir,
		})
		if err != nil {
			t.Fatalf("Failed to create context builder: %v", err)
		}

		// Test JSONL format
		jsonlDir := filepath.Join(tmpDir, "memory-jsonl")
		jsonlStore, err := memory.NewStore(memory.Config{
			BaseDir: jsonlDir,
			Format:  memory.FormatJSONL,
		})
		if err != nil {
			t.Fatalf("Failed to create JSONL store: %v", err)
		}

		sessionID := "format-test"
		msg := llm.Message{Role: llm.RoleUser, Content: "Test message"}
		jsonlStore.Write(sessionID, msg)

		jsonlMemory, _ := jsonlStore.Read(sessionID)
		contextJSONL, _ := ctxBuilder.BuildWithMemory(jsonlMemory)

		if !strings.Contains(contextJSONL, "Test message") {
			t.Error("JSONL context should contain message")
		}

		// Test Markdown format
		markdownDir := filepath.Join(tmpDir, "memory-markdown")
		markdownStore, err := memory.NewStore(memory.Config{
			BaseDir: markdownDir,
			Format:  memory.FormatMarkdown,
		})
		if err != nil {
			t.Fatalf("Failed to create Markdown store: %v", err)
		}

		markdownStore.Write(sessionID, msg)

		markdownMemory, _ := markdownStore.Read(sessionID)
		contextMD, _ := ctxBuilder.BuildWithMemory(markdownMemory)

		if !strings.Contains(contextMD, "Test message") {
			t.Error("Markdown context should contain message")
		}

		// Both should work with context builder
		if contextJSONL == "" || contextMD == "" {
			t.Error("Both formats should produce valid contexts")
		}
	})
}

// TestPersistenceAcrossRestart tests data persistence across component restarts
func TestPersistenceAcrossRestart(t *testing.T) {
	t.Run("data persists after component recreation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Phase 1: Create and populate
		sessionDir := filepath.Join(tmpDir, "sessions")
		sessionMgr1, _ := session.NewManager(sessionDir)

		memoryDir := filepath.Join(tmpDir, "memory")
		memStore1, _ := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})

		sessionID := "persist-test"
		sess1, _, _ := sessionMgr1.GetOrCreate(sessionID)

		msg := llm.Message{Role: llm.RoleUser, Content: "Persistent message"}
		sess1.Append(msg)
		memStore1.Write(sessionID, msg)

		// Phase 2: Recreate components (simulate restart)
		sessionMgr2, _ := session.NewManager(sessionDir)
		memStore2, _ := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})

		// Phase 3: Verify data persistence
		sess2, created, _ := sessionMgr2.GetOrCreate(sessionID)
		if created {
			t.Error("Session should already exist, not be newly created")
		}

		sess2Messages, _ := sess2.Read()
		mem2Messages, _ := memStore2.Read(sessionID)

		if len(sess2Messages) != 1 {
			t.Errorf("Session should have 1 message, got %d", len(sess2Messages))
		}

		if len(mem2Messages) != 1 {
			t.Errorf("Memory should have 1 message, got %d", len(mem2Messages))
		}

		if sess2Messages[0].Content != "Persistent message" {
			t.Error("Message content should persist")
		}
	})
}
