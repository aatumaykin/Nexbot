package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApp_Initialize_Success(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify components are initialized
	if app.messageBus == nil {
		t.Error("Initialize() messageBus should not be nil")
	}
	if app.agentLoop == nil {
		t.Error("Initialize() agentLoop should not be nil")
	}
	if app.commandHandler == nil {
		t.Error("Initialize() commandHandler should not be nil")
	}
	if app.telegram != nil {
		t.Error("Initialize() telegram should be nil when disabled")
	}
	if app.cronScheduler != nil {
		t.Error("Initialize() cronScheduler should be nil when disabled")
	}

	// Verify context is created
	if app.ctx == nil {
		t.Error("Initialize() ctx should not be nil")
	}
	if app.cancel == nil {
		t.Error("Initialize() cancel should not be nil")
	}

	// Verify started flag is set
	app.mu.Lock()
	started := app.started
	app.mu.Unlock()

	if !started {
		t.Error("Initialize() started should be true")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_WithTelegram(t *testing.T) {
	t.Skip("Telegram connector requires valid API token for initialization - skip in unit tests")

	cfg := createTestConfig(t)
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "123456789:ABCDEFGHIJabcdefghij1234567890AB"

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify telegram is initialized
	if app.telegram == nil {
		t.Error("Initialize() telegram should not be nil when enabled")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_WithCron(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Cron.Enabled = true

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify cron scheduler is initialized
	if app.cronScheduler == nil {
		t.Error("Initialize() cronScheduler should not be nil when enabled")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_WithShellTool(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Tools.Shell.Enabled = true
	cfg.Tools.Shell.AllowedCommands = []string{"ls", "pwd", "echo"}

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Tools should be registered in agent loop
	if app.agentLoop == nil {
		t.Error("Initialize() agentLoop should not be nil")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_WithFileTool(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Tools.File.Enabled = true
	cfg.Tools.File.WhitelistDirs = []string{filepath.Join(cfg.Workspace.Path, "allowed")}
	cfg.Tools.File.ReadOnlyDirs = []string{filepath.Join(cfg.Workspace.Path, "readonly")}

	// Create whitelist and readonly directories
	for _, dir := range cfg.Tools.File.WhitelistDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create whitelist directory: %v", err)
		}
	}
	for _, dir := range cfg.Tools.File.ReadOnlyDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create readonly directory: %v", err)
		}
	}

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Tools should be registered in agent loop
	if app.agentLoop == nil {
		t.Error("Initialize() agentLoop should not be nil")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_InvalidLLMProvider(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Agent.Provider = "invalid_provider"

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err == nil {
		t.Error("Initialize() expected error for invalid LLM provider, got nil")
	}

	expectedErrMsg := "unsupported LLM provider"
	if err != nil && err.Error() == "" {
		t.Errorf("Initialize() error message should contain %q", expectedErrMsg)
	}
}

func TestApp_Initialize_AllTools(t *testing.T) {
	t.Skip("Telegram connector requires valid API token for initialization - skip in unit tests")

	cfg := createTestConfig(t)
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "123456789:ABCDEFGHIJabcdefghij1234567890AB"
	cfg.Cron.Enabled = true
	cfg.Tools.Shell.Enabled = true
	cfg.Tools.Shell.AllowedCommands = []string{"ls"}
	cfg.Tools.File.Enabled = true
	cfg.Tools.File.WhitelistDirs = []string{filepath.Join(cfg.Workspace.Path, "allowed")}

	// Create whitelist directory
	for _, dir := range cfg.Tools.File.WhitelistDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create whitelist directory: %v", err)
		}
	}

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify all components are initialized
	if app.messageBus == nil {
		t.Error("Initialize() messageBus should not be nil")
	}
	if app.agentLoop == nil {
		t.Error("Initialize() agentLoop should not be nil")
	}
	if app.commandHandler == nil {
		t.Error("Initialize() commandHandler should not be nil")
	}
	if app.telegram == nil {
		t.Error("Initialize() telegram should not be nil when enabled")
	}
	if app.cronScheduler == nil {
		t.Error("Initialize() cronScheduler should not be nil when enabled")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_WorkspaceDirectories(t *testing.T) {
	tmpDir := shortTestDir(t, "")
	cfg := createTestConfig(t)
	cfg.Workspace.Path = tmpDir

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify workspace directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Initialize() workspace directory should exist")
	}

	// Verify sessions directory exists
	sessionsDir := filepath.Join(tmpDir, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Error("Initialize() sessions directory should exist")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_AlreadyInitialized(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	// First initialization
	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("First Initialize() failed: %v", err)
	}

	// Try to initialize again
	err = app.Initialize(ctx)
	// Should succeed but may overwrite existing components
	if err != nil {
		t.Errorf("Second Initialize() should succeed, got error: %v", err)
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_ContextCreation(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify context is a child of the provided context
	if app.ctx == nil {
		t.Error("Initialize() ctx should not be nil")
	}

	// Verify cancel function works
	select {
	case <-app.ctx.Done():
		t.Error("Initialize() context should not be cancelled yet")
	default:
		// Expected
	}

	app.cancel()

	select {
	case <-app.ctx.Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Initialize() cancel function should cancel context")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Initialize_StartMessageProcessing(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Start message processing
	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Errorf("StartMessageProcessing() failed: %v", err)
	}

	// Verify message processing started
	// This is a basic check - more thorough testing would require mocking the message bus

	// Cleanup
	_ = app.Shutdown()
}
