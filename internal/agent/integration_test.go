package agent_test

import (
	"context"
	"github.com/aatumaykin/nexbot/internal/config"
	"os"
	"path/filepath"
	"testing"

	agentcontext "github.com/aatumaykin/nexbot/internal/agent/context"
	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/agent/memory"
	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentIntegration_FullWorkflow(t *testing.T) {
	// Настройка временной директории
	tmpDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tmpDir})
	sessionDir := filepath.Join(tmpDir, "sessions")
	memoryDir := filepath.Join(tmpDir, "memory")

	require.NoError(t, os.MkdirAll(sessionDir, 0755))
	require.NoError(t, os.MkdirAll(memoryDir, 0755))

	// Создать bootstrap файлы для workspace
	createBootstrapFiles(t, tmpDir)

	// Создать logger
	log, err := logger.New(logger.Config{
		Format: "text",
		Level:  "debug",
		Output: "stdout",
	})
	require.NoError(t, err)

	t.Run("context builder + session manager integration", func(t *testing.T) {
		// Создать context builder
		contextBuilder, err := agentcontext.NewBuilder(agentcontext.Config{
			Workspace: ws,
		})
		require.NoError(t, err)
		require.NotNil(t, contextBuilder)

		// Создать session manager
		sessionManager, err := session.NewManager(sessionDir)
		require.NoError(t, err)

		// Создать тестовую сессию
		sess, created, err := sessionManager.GetOrCreate("test-session")
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.True(t, created)

		// Добавить сообщения в сессию
		err = sess.Append(llm.Message{
			Role:    llm.RoleUser,
			Content: "Hello",
		})
		require.NoError(t, err)

		// Прочитать сообщения из сессии
		messages, err := sess.Read()
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, llm.RoleUser, messages[0].Role)
		assert.Equal(t, "Hello", messages[0].Content)

		// Построить контекст с памятью из сессии
		contextStr, err := contextBuilder.BuildForSession("test-session", messages)
		require.NoError(t, err)
		require.NotEmpty(t, contextStr)

		// Проверить что контекст содержит сообщения сессии
		assert.Contains(t, contextStr, "Hello")
	})

	t.Run("context builder + memory integration", func(t *testing.T) {
		// Создать memory store
		memoryStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		require.NoError(t, err)
		require.NotNil(t, memoryStore)

		// Сохранить память
		testMemory := llm.Message{
			Role:    llm.RoleSystem,
			Content: "This is a test memory about the project",
		}
		err = memoryStore.Write("test-session", testMemory)
		require.NoError(t, err)

		// Прочитать память
		memories, err := memoryStore.Read("test-session")
		require.NoError(t, err)
		require.Len(t, memories, 1)
		assert.Equal(t, llm.RoleSystem, memories[0].Role)
		assert.Contains(t, memories[0].Content, "test memory")

		// Создать context builder
		contextBuilder, err := agentcontext.NewBuilder(agentcontext.Config{
			Workspace: ws,
		})
		require.NoError(t, err)

		// Создать тестовый файл памяти в workspace
		workspaceMemoryDir := filepath.Join(tmpDir, "memory")
		require.NoError(t, os.MkdirAll(workspaceMemoryDir, 0755))
		testMemoryFile := filepath.Join(workspaceMemoryDir, "test.md")
		require.NoError(t, os.WriteFile(testMemoryFile, []byte("Test workspace memory"), 0644))

		// Прочитать память через context builder
		memoriesFromBuilder, err := contextBuilder.ReadMemory()
		require.NoError(t, err)
		require.Len(t, memoriesFromBuilder, 1)

		// Построить контекст с памятью
		contextStr, err := contextBuilder.BuildWithMemory(memoriesFromBuilder)
		require.NoError(t, err)
		require.NotEmpty(t, contextStr)

		// Проверить что контекст содержит секцию памяти
		assert.Contains(t, contextStr, "Recent Conversation Memory")
		assert.Contains(t, contextStr, "Test workspace memory")
	})

	t.Run("session manager + loop integration", func(t *testing.T) {
		ctx := context.Background()

		// Создать session manager
		sessionManager, err := session.NewManager(sessionDir)
		require.NoError(t, err)

		// Создать mock LLM provider
		testLLM := &mockLLMProvider{
			responses: []string{"Mock response"},
		}

		// Создать loop config
		loopCfg := loop.Config{
			Workspace:         ws,
			SessionDir:        sessionDir,
			LLMProvider:       testLLM,
			Logger:            log,
			Model:             "test-model",
			MaxTokens:         4096,
			Temperature:       0.7,
			MaxToolIterations: 10,
		}

		// Создать loop
		loopInstance, err := loop.NewLoop(loopCfg)
		require.NoError(t, err)
		require.NotNil(t, loopInstance)

		// Проверить что loop использует session manager
		assert.Equal(t, sessionManager, loopInstance.GetSessionManager())

		// Проверить что context builder создан
		assert.NotNil(t, loopInstance.GetContextBuilder())

		// Проверить что LLM provider установлен
		assert.Equal(t, testLLM, loopInstance.GetLLMProvider())

		// Создать сессию через loop
		_, created, err := sessionManager.GetOrCreate("test-session-2")
		require.NoError(t, err)
		assert.True(t, created)

		// Добавить сообщение в сессию через loop
		err = loopInstance.AddMessageToSession(ctx, "test-session-2", llm.Message{
			Role:    llm.RoleUser,
			Content: "Hello, test!",
		})
		require.NoError(t, err)

		// Прочитать историю сессии через loop
		history, err := loopInstance.GetSessionHistory(ctx, "test-session-2")
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, llm.RoleUser, history[0].Role)
		assert.Equal(t, "Hello, test!", history[0].Content)
	})

	t.Run("full integration: context + session + memory + loop", func(t *testing.T) {
		ctx := context.Background()

		// Создать memory store
		memoryStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		require.NoError(t, err)

		// Сохранить долгосрочную память
		err = memoryStore.Write("full-integration", llm.Message{
			Role:    llm.RoleSystem,
			Content: "User preferences: likes concise answers",
		})
		require.NoError(t, err)

		// Создать loop
		testLLM := &mockLLMProvider{
			responses: []string{"Concise answer"},
		}
		loopInstance, err := loop.NewLoop(loop.Config{
			Workspace:         ws,
			SessionDir:        sessionDir,
			LLMProvider:       testLLM,
			Logger:            log,
			Model:             "test-model",
			MaxTokens:         4096,
			Temperature:       0.7,
			MaxToolIterations: 10,
		})
		require.NoError(t, err)

		// Обработать сообщение через loop
		response, err := loopInstance.Process(ctx, "full-integration", "Hello")
		require.NoError(t, err)
		assert.NotEmpty(t, response)

		// Проверить что сессия создана и содержит сообщения
		history, err := loopInstance.GetSessionHistory(ctx, "full-integration")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 2) // user message + assistant response

		// Проверить что память сохранена отдельно
		memories, err := memoryStore.Read("full-integration")
		require.NoError(t, err)
		assert.Len(t, memories, 1) // Только долгосрочная память
	})
}

func TestAgentIntegration_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("context builder with non-existent workspace", func(t *testing.T) {
		nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
		nonExistentWs := workspace.New(config.WorkspaceConfig{Path: nonExistentDir})

		_, err := agentcontext.NewBuilder(agentcontext.Config{
			Workspace: nonExistentWs,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("session manager with invalid directory", func(t *testing.T) {
		invalidPath := "\x00invalid" // Invalid path

		_, err := session.NewManager(invalidPath)
		assert.Error(t, err)
	})

	t.Run("memory store with invalid configuration", func(t *testing.T) {
		_, err := memory.NewStore(memory.Config{
			BaseDir: "", // Empty base dir
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("loop with missing configuration", func(t *testing.T) {
		// Missing LLM provider
		log, err := logger.New(logger.Config{Format: "text", Level: "info", Output: "stdout"})
		require.NoError(t, err)

		_, err = loop.NewLoop(loop.Config{
			Workspace:  workspace.New(config.WorkspaceConfig{Path: tmpDir}),
			SessionDir: tmpDir,
			Logger:     log,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM provider cannot be nil")

		// Missing logger
		testLLM := &mockLLMProvider{}
		_, err = loop.NewLoop(loop.Config{
			Workspace:   workspace.New(config.WorkspaceConfig{Path: tmpDir}),
			SessionDir:  tmpDir,
			LLMProvider: testLLM,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})
}

func TestAgentIntegration_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	t.Run("concurrent session operations", func(t *testing.T) {
		sessionManager, err := session.NewManager(sessionDir)
		require.NoError(t, err)

		const numGoroutines = 10
		const messagesPerGoroutine = 100

		done := make(chan bool, numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer func() { done <- true }()

				sessionID := "concurrent-session"
				for j := range messagesPerGoroutine {
					sess, _, err := sessionManager.GetOrCreate(sessionID)
					require.NoError(t, err)

					err = sess.Append(llm.Message{
						Role:    llm.RoleUser,
						Content: string(rune('A' + j)),
					})
					require.NoError(t, err)
				}
			}(i)
		}

		// Wait for all goroutines
		for range numGoroutines {
			<-done
		}

		// Verify message count
		sess, _, err := sessionManager.GetOrCreate("concurrent-session")
		require.NoError(t, err)

		messages, err := sess.Read()
		require.NoError(t, err)
		assert.Equal(t, numGoroutines*messagesPerGoroutine, len(messages))
	})

	t.Run("concurrent memory operations", func(t *testing.T) {
		memoryDir := filepath.Join(tmpDir, "memory")
		memoryStore, err := memory.NewStore(memory.Config{
			BaseDir: memoryDir,
			Format:  memory.FormatJSONL,
		})
		require.NoError(t, err)

		const numGoroutines = 10
		const messagesPerGoroutine = 50

		done := make(chan bool, numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer func() { done <- true }()

				sessionID := "concurrent-memory"
				for j := range messagesPerGoroutine {
					err := memoryStore.Write(sessionID, llm.Message{
						Role:    llm.RoleSystem,
						Content: string(rune('A' + j)),
					})
					require.NoError(t, err)
				}
			}(i)
		}

		// Wait for all goroutines
		for range numGoroutines {
			<-done
		}

		// Verify message count
		messages, err := memoryStore.Read("concurrent-memory")
		require.NoError(t, err)
		assert.Equal(t, numGoroutines*messagesPerGoroutine, len(messages))
	})
}

// Helper functions

func createBootstrapFiles(t *testing.T, dir string) {
	mainDir := filepath.Join(dir, "main")
	require.NoError(t, os.MkdirAll(mainDir, 0755))

	identityPath := filepath.Join(mainDir, workspace.BootstrapIdentity)
	require.NoError(t, os.WriteFile(identityPath, []byte("Test Identity"), 0644))

	agentsPath := filepath.Join(mainDir, workspace.BootstrapAgents)
	require.NoError(t, os.WriteFile(agentsPath, []byte("Test Agents"), 0644))

	userPath := filepath.Join(mainDir, workspace.BootstrapUser)
	require.NoError(t, os.WriteFile(userPath, []byte("Test User"), 0644))

	toolsPath := filepath.Join(mainDir, workspace.BootstrapTools)
	require.NoError(t, os.WriteFile(toolsPath, []byte("Test Tools"), 0644))
}

// Mock LLM provider для тестов
type mockLLMProvider struct {
	responses []string
	callCount int
}

func (m *mockLLMProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	response := "Mock response"
	if m.callCount < len(m.responses) {
		response = m.responses[m.callCount]
	}
	m.callCount++

	return &llm.ChatResponse{
		Content:      response,
		FinishReason: llm.FinishReasonStop,
		ToolCalls:    []llm.ToolCall{},
	}, nil
}

func (m *mockLLMProvider) SupportsToolCalling() bool {
	return true
}
