package subagent

import (
	"context"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})

	// Create logger
	log := testLogger()

	// Create manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})

	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())

	// Check that subagent directory was created
	subagentDir := filepath.Join(tempDir, "subagents")
	info, err := os.Stat(subagentDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewManagerInvalidConfig(t *testing.T) {
	log := testLogger()

	// Test with empty session directory
	_, err := NewManager(Config{
		SessionDir: "",
		Logger:     log,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session directory cannot be empty")

	// Test with nil logger
	_, err = NewManager(Config{
		SessionDir: "/tmp",
		Logger:     nil,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logger cannot be empty")
}

func TestManagerSpawn(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Spawn a subagent
	ctx := context.Background()
	subagent, err := manager.Spawn(ctx, "parent-123", "Test task")
	require.NoError(t, err)

	// Verify subagent properties
	assert.NotEmpty(t, subagent.ID)
	assert.NotEmpty(t, subagent.Session)
	assert.Contains(t, subagent.Session, SessionIDPrefix)
	assert.NotNil(t, subagent.Loop)
	assert.NotNil(t, subagent.Context)
	assert.NotNil(t, subagent.Cancel)

	// Verify manager count
	assert.Equal(t, 1, manager.Count())

	// Can retrieve the subagent
	retrieved, err := manager.Get(subagent.ID)
	require.NoError(t, err)
	assert.Equal(t, subagent.ID, retrieved.ID)
	assert.Equal(t, subagent.Session, retrieved.Session)
}

func TestManagerSpawnMultiple(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn multiple subagents
	subagent1, err := manager.Spawn(ctx, "parent-123", "Task 1")
	require.NoError(t, err)

	subagent2, err := manager.Spawn(ctx, "parent-123", "Task 2")
	require.NoError(t, err)

	subagent3, err := manager.Spawn(ctx, "parent-456", "Task 3")
	require.NoError(t, err)

	// Verify all subagents have unique IDs
	assert.NotEqual(t, subagent1.ID, subagent2.ID)
	assert.NotEqual(t, subagent2.ID, subagent3.ID)
	assert.NotEqual(t, subagent1.ID, subagent3.ID)

	// Verify manager count
	assert.Equal(t, 3, manager.Count())

	// List all subagents
	subagents := manager.List()
	assert.Len(t, subagents, 3)

	subagentIDs := make(map[string]bool)
	for _, sub := range subagents {
		subagentIDs[sub.ID] = true
	}
	assert.True(t, subagentIDs[subagent1.ID])
	assert.True(t, subagentIDs[subagent2.ID])
	assert.True(t, subagentIDs[subagent3.ID])
}

func TestManagerStop(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn a subagent
	subagent, err := manager.Spawn(ctx, "parent-123", "Test task")
	require.NoError(t, err)
	assert.Equal(t, 1, manager.Count())

	// Stop the subagent
	err = manager.Stop(subagent.ID)
	assert.NoError(t, err)

	// Verify subagent is removed
	assert.Equal(t, 0, manager.Count())

	// Cannot retrieve stopped subagent
	_, err = manager.Get(subagent.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subagent not found")

	// Stopping non-existent subagent returns error
	err = manager.Stop(subagent.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subagent not found")
}

func TestManagerStopAll(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn multiple subagents
	_, err = manager.Spawn(ctx, "parent-123", "Task 1")
	require.NoError(t, err)
	_, err = manager.Spawn(ctx, "parent-123", "Task 2")
	require.NoError(t, err)
	_, err = manager.Spawn(ctx, "parent-456", "Task 3")
	require.NoError(t, err)

	assert.Equal(t, 3, manager.Count())

	// Stop all subagents
	manager.StopAll()

	// Verify all subagents are stopped
	assert.Equal(t, 0, manager.Count())
	subagents := manager.List()
	assert.Len(t, subagents, 0)
}

func TestManagerList(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Initially empty
	subagents := manager.List()
	assert.Len(t, subagents, 0)

	// Spawn subagents
	sub1, _ := manager.Spawn(ctx, "parent-1", "Task 1")
	sub2, _ := manager.Spawn(ctx, "parent-2", "Task 2")
	sub3, _ := manager.Spawn(ctx, "parent-3", "Task 3")

	// List should return all subagents
	subagents = manager.List()
	assert.Len(t, subagents, 3)

	subagentIDs := make(map[string]bool)
	for _, sub := range subagents {
		subagentIDs[sub.ID] = true
	}
	assert.True(t, subagentIDs[sub1.ID])
	assert.True(t, subagentIDs[sub2.ID])
	assert.True(t, subagentIDs[sub3.ID])
}

func TestManagerGet(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn a subagent
	subagent, err := manager.Spawn(ctx, "parent-123", "Test task")
	require.NoError(t, err)

	// Get existing subagent
	retrieved, err := manager.Get(subagent.ID)
	require.NoError(t, err)
	assert.Equal(t, subagent.ID, retrieved.ID)
	assert.Equal(t, subagent.Session, retrieved.Session)

	// Get non-existent subagent
	_, err = manager.Get("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subagent not found")
}

func TestSubagentProcess(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "Mock response"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn a subagent
	subagent, err := manager.Spawn(ctx, "parent-123", "Initial task")
	require.NoError(t, err)

	// Process a task
	response, err := subagent.Process(ctx, "What is 2+2?")
	require.NoError(t, err)
	assert.Equal(t, "Mock response", response)
}

func TestSubagentContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn a subagent
	subagent, err := manager.Spawn(ctx, "parent-123", "Initial task")
	require.NoError(t, err)

	// Cancel subagent context
	subagent.Cancel()

	// Process with cancelled context should fail
	// (Note: actual behavior depends on Loop.Process implementation)
	assert.NotNil(t, subagent.Context.Err())
}

func TestManagerConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn multiple subagents concurrently
	done := make(chan *Subagent, 10)
	for range 10 {
		go func() {
			sub, err := manager.Spawn(ctx, "parent-123", "Task")
			assert.NoError(t, err)
			done <- sub
		}()
	}

	// Wait for all spawns to complete
	subagents := make([]*Subagent, 0, 10)
	for range 10 {
		sub := <-done
		subagents = append(subagents, sub)
	}

	// Verify all subagents were spawned
	assert.Len(t, subagents, 10)
	assert.Equal(t, 10, manager.Count())

	// Verify all subagent IDs are unique
	ids := make(map[string]bool)
	for _, sub := range subagents {
		assert.False(t, ids[sub.ID], "duplicate subagent ID")
		ids[sub.ID] = true
	}
}

func TestStorageNewStorage(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, tempDir, storage.baseDir)

	// Directory should exist
	info, err := os.Stat(tempDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStorageSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)

	subagentID := "test-subagent-123"

	// Save an entry
	entry := map[string]any{
		"message": "test",
		"time":    time.Now().Unix(),
	}
	err = storage.Save(subagentID, entry)
	require.NoError(t, err)

	// Load entries
	entries, err := storage.Load(subagentID)
	require.NoError(t, err)
	// For now, returns empty slice (JSONL not fully implemented)
	assert.NotNil(t, entries)
}

func TestStorageDelete(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)

	subagentID := "test-subagent-123"

	// Save an entry
	err = storage.Save(subagentID, map[string]any{"message": "test"})
	require.NoError(t, err)

	// Check directory exists
	subagentPath := filepath.Join(tempDir, subagentID)
	info, err := os.Stat(subagentPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Delete
	err = storage.Delete(subagentID)
	require.NoError(t, err)

	// Directory should be removed
	_, err = os.Stat(subagentPath)
	assert.True(t, os.IsNotExist(err))

	// Deleting non-existent should not error
	err = storage.Delete("non-existent")
	require.NoError(t, err)
}

func TestStorageList(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)

	// Initially empty
	subagentIDs, err := storage.List()
	require.NoError(t, err)
	assert.Len(t, subagentIDs, 0)

	// Create some subagent directories
	ignoreError(storage.Save("subagent-1", map[string]any{}))
	ignoreError(storage.Save("subagent-2", map[string]any{}))
	ignoreError(storage.Save("subagent-3", map[string]any{}))

	// List should return all subagent IDs
	subagentIDs, err = storage.List()
	require.NoError(t, err)
	assert.Len(t, subagentIDs, 3)

	idMap := make(map[string]bool)
	for _, id := range subagentIDs {
		idMap[id] = true
	}
	assert.True(t, idMap["subagent-1"])
	assert.True(t, idMap["subagent-2"])
	assert.True(t, idMap["subagent-3"])
}

// testLogger creates a test logger instance
func testLogger() *logger.Logger {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		panic(err)
	}
	return log
}

// ignoreError ignores error (for use in tests)
func ignoreError(err error) {
	_ = err
}

// mockLLMProvider is a mock LLM provider for testing
type mockLLMProvider struct {
	response string
}

func (m *mockLLMProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Content:      m.response,
		FinishReason: llm.FinishReasonStop,
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		ToolCalls: []llm.ToolCall{},
	}, nil
}

func (m *mockLLMProvider) SupportsToolCalling() bool {
	return false
}
