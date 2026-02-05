package app

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
)

func TestApp_Shutdown_NotStarted(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))

	// Shutdown without starting - should succeed
	err := app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() should succeed when not started, got error: %v", err)
	}

	// Verify started flag remains false
	app.mu.Lock()
	started := app.started
	app.mu.Unlock()

	if started {
		t.Error("Shutdown() started should be false after shutdown of not started app")
	}
}

func TestApp_Shutdown_Started(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	// Initialize and start
	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// Verify started flag is false
	app.mu.Lock()
	started := app.started
	app.mu.Unlock()

	if started {
		t.Error("Shutdown() started should be false after shutdown")
	}

	// Verify context is cancelled
	select {
	case <-app.ctx.Done():
		// Expected
	default:
		t.Error("Shutdown() context should be cancelled")
	}
}

func TestApp_Shutdown_WithTelegram(t *testing.T) {
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

	// Shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// Verify telegram is still present in app struct
	if app.telegram == nil {
		t.Error("Shutdown() telegram should not be nil after shutdown")
	}
}

func TestApp_Shutdown_WithCron(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Cron.Enabled = true

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// Verify cron scheduler is still present in app struct
	if app.cronScheduler == nil {
		t.Error("Shutdown() cronScheduler should not be nil after shutdown")
	}
}

func TestApp_Shutdown_MultipleTimes(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// First shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("First Shutdown() failed: %v", err)
	}

	// Second shutdown - should succeed
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Second Shutdown() should succeed, got error: %v", err)
	}
}

func TestApp_Shutdown_MessageBusStop(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify message bus is started
	if !app.messageBus.IsStarted() {
		t.Error("Initialize() messageBus should be started")
	}

	// Shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// Verify message bus is stopped
	if app.messageBus != nil && app.messageBus.IsStarted() {
		t.Error("Shutdown() messageBus should be stopped")
	}
}

func TestApp_Shutdown_WithActiveMessageProcessing(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Publish a message
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"test message",
		nil,
	)

	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait a bit then shutdown
	time.Sleep(100 * time.Millisecond)

	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}

func TestApp_Restart(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	// Initial initialization
	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("First Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("First StartMessageProcessing() failed: %v", err)
	}

	// Get initial references
	initialMessageBus := app.messageBus
	initialAgentLoop := app.agentLoop

	// Restart
	err = app.Restart()
	if err != nil {
		t.Errorf("Restart() failed: %v", err)
	}

	// Verify components are reinitialized
	if app.messageBus == nil {
		t.Error("Restart() messageBus should not be nil")
	}
	if app.agentLoop == nil {
		t.Error("Restart() agentLoop should not be nil")
	}

	// Verify new instances were created
	if app.messageBus == initialMessageBus {
		t.Error("Restart() messageBus should be a new instance")
	}
	if app.agentLoop == initialAgentLoop {
		t.Error("Restart() agentLoop should be a new instance")
	}

	// Verify started flag is true
	app.mu.Lock()
	started := app.started
	app.mu.Unlock()

	if !started {
		t.Error("Restart() started should be true after restart")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Restart_NotStarted(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))

	// Restart without starting - should succeed
	err := app.Restart()
	if err != nil {
		t.Errorf("Restart() should succeed when not started, got error: %v", err)
	}

	// Verify components are initialized
	if app.messageBus == nil {
		t.Error("Restart() messageBus should not be nil")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Restart_WithAllComponents(t *testing.T) {
	t.Skip("Telegram connector requires valid API token for initialization - skip in unit tests")

	cfg := createTestConfig(t)
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "123456789:ABCDEFGHIJabcdefghij1234567890AB"
	cfg.Cron.Enabled = true
	cfg.Tools.Shell.Enabled = true
	cfg.Tools.Shell.AllowedCommands = []string{"ls"}

	app := New(cfg, createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Restart
	err = app.Restart()
	if err != nil {
		t.Errorf("Restart() failed: %v", err)
	}

	// Verify all components are reinitialized
	if app.messageBus == nil {
		t.Error("Restart() messageBus should not be nil")
	}
	if app.agentLoop == nil {
		t.Error("Restart() agentLoop should not be nil")
	}
	if app.commandHandler == nil {
		t.Error("Restart() commandHandler should not be nil")
	}
	if app.telegram == nil {
		t.Error("Restart() telegram should not be nil when enabled")
	}
	if app.cronScheduler == nil {
		t.Error("Restart() cronScheduler should not be nil when enabled")
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Restart_MultipleTimes(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("First Initialize() failed: %v", err)
	}

	// First restart
	err = app.Restart()
	if err != nil {
		t.Errorf("First Restart() failed: %v", err)
	}

	// Second restart
	err = app.Restart()
	if err != nil {
		t.Errorf("Second Restart() failed: %v", err)
	}

	// Third restart
	err = app.Restart()
	if err != nil {
		t.Errorf("Third Restart() failed: %v", err)
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Restart_ContextRecreation(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Get initial context
	initialCtx := app.ctx

	// Restart
	err = app.Restart()
	if err != nil {
		t.Errorf("Restart() failed: %v", err)
	}

	// Verify new context was created
	if app.ctx == nil {
		t.Error("Restart() ctx should not be nil")
	}
	if app.cancel == nil {
		t.Error("Restart() cancel should not be nil")
	}
	if app.ctx == initialCtx {
		t.Error("Restart() ctx should be a new instance")
	}

	// Verify initial context is cancelled
	select {
	case <-initialCtx.Done():
		// Expected
	default:
		t.Error("Restart() initial context should be cancelled")
	}

	// Verify new context is not cancelled
	select {
	case <-app.ctx.Done():
		t.Error("Restart() new context should not be cancelled")
	default:
		// Expected
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_Shutdown_ThreadSafety(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Call shutdown from multiple goroutines
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			done <- app.Shutdown()
		}()
	}

	// Wait for all shutdowns to complete
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent Shutdown() #%d failed: %v", i, err)
		}
	}
}

func TestApp_Restart_ThreadSafety(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Call restart from multiple goroutines
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			done <- app.Restart()
		}()
	}

	// Wait for all restarts to complete
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent Restart() #%d failed: %v", i, err)
		}
	}

	// Cleanup
	_ = app.Shutdown()
}
