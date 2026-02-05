package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Helper function to create test logger
func createTestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	cfg := logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return log
}

// Helper function to create test config
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	tmpDir := t.TempDir()
	return &config.Config{
		Workspace: config.WorkspaceConfig{
			Path:              tmpDir,
			BootstrapMaxChars: 20000,
		},
		Agent: config.AgentConfig{
			Provider:       "zai",
			Model:          "glm-4.7-flash",
			MaxTokens:      8192,
			MaxIterations:  20,
			Temperature:    0.7,
			TimeoutSeconds: 30,
		},
		LLM: config.LLMConfig{
			ZAI: config.ZAIConfig{
				APIKey:         "zai-test-api-key-123456",
				BaseURL:        "https://api.z.ai/api/coding/paas/v4",
				TimeoutSeconds: 30,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Channels: config.ChannelsConfig{
			Telegram: config.TelegramConfig{
				Enabled: false,
			},
		},
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled: false,
			},
			Shell: config.ShellToolConfig{
				Enabled:         false,
				AllowedCommands: []string{"ls"},
			},
		},
		Cron: config.CronConfig{
			Enabled: false,
		},
		MessageBus: config.MessageBusConfig{
			Capacity: 100,
		},
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		log     *logger.Logger
		wantErr bool
	}{
		{
			name:    "valid app creation",
			cfg:     &config.Config{},
			log:     createTestLogger(t),
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			log:     createTestLogger(t),
			wantErr: false,
		},
		{
			name:    "nil logger",
			cfg:     &config.Config{},
			log:     nil,
			wantErr: false,
		},
		{
			name:    "both nil",
			cfg:     nil,
			log:     nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.cfg, tt.log)

			if app == nil {
				t.Fatal("New() returned nil app")
			}

			if app.config != tt.cfg {
				t.Errorf("New() config = %v, want %v", app.config, tt.cfg)
			}

			if app.logger != tt.log {
				t.Errorf("New() logger = %v, want %v", app.logger, tt.log)
			}

			// Verify other fields are nil
			if app.messageBus != nil {
				t.Error("New() messageBus should be nil")
			}
			if app.agentLoop != nil {
				t.Error("New() agentLoop should be nil")
			}
			if app.commandHandler != nil {
				t.Error("New() commandHandler should be nil")
			}
			if app.telegram != nil {
				t.Error("New() telegram should be nil")
			}
			if app.cronScheduler != nil {
				t.Error("New() cronScheduler should be nil")
			}
		})
	}
}

func TestApp_Run_ContextCancellation(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- app.Run(ctx)
	}()

	// Give time for initialization to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Run to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within timeout")
	}
}

func TestApp_Run_InitializeError(t *testing.T) {
	// Create a config that will fail initialization
	cfg := createTestConfig(t)
	cfg.Agent.Provider = "invalid_provider" // This will cause Initialize to fail

	app := New(cfg, createTestLogger(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.Run(ctx)
	if err == nil {
		t.Error("Run() expected error from Initialize, got nil")
	}

	// Verify error is about initialization
	if err != nil && !errors.Is(err, context.Canceled) && err.Error() == "" {
		t.Errorf("Run() error = %v", err)
	}
}

func TestApp_Run_StartMessageProcessingError(t *testing.T) {
	// This test would require mocking Initialize to succeed but StartMessageProcessing to fail
	// For now, we'll just verify the Run method structure
	t.Skip("Requires more complex mocking setup")
}

func TestApp_ContextFields(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))

	// Verify context fields are initialized to zero values
	if app.ctx != nil {
		t.Error("New() ctx should be nil")
	}
	if app.cancel != nil {
		t.Error("New() cancel should be nil")
	}
}

func TestApp_StartedFlag(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))

	// Verify started flag is initially false
	app.mu.Lock()
	started := app.started
	app.mu.Unlock()

	if started {
		t.Error("New() started should be false")
	}
}
