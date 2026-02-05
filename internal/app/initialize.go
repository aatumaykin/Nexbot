package app

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels/telegram"
	"github.com/aatumaykin/nexbot/internal/commands"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// Initialize initializes all application components.
// It sets up the message bus, LLM provider, workspace, agent loop,
// command handler, tools, telegram connector, and cron scheduler.
func (a *App) Initialize(ctx context.Context) error {
	// 1. Create application context
	a.ctx, a.cancel = context.WithCancel(ctx)

	// 2. Initialize message bus
	a.messageBus = bus.New(a.config.MessageBus.Capacity, a.logger)
	if err := a.messageBus.Start(a.ctx); err != nil {
		return fmt.Errorf("failed to start message bus: %w", err)
	}

	// 3. Initialize LLM provider
	var provider llm.Provider
	switch a.config.Agent.Provider {
	case "zai":
		zaiConfig := llm.ZAIConfig{
			APIKey:         a.config.LLM.ZAI.APIKey,
			TimeoutSeconds: a.config.LLM.ZAI.TimeoutSeconds,
		}
		provider = llm.NewZAIProvider(zaiConfig, a.logger)
	default:
		return fmt.Errorf("unsupported LLM provider: %s", a.config.Agent.Provider)
	}

	// 4. Initialize workspace
	ws := workspace.New(a.config.Workspace)
	if err := ws.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}
	if err := ws.EnsureSubpath("sessions"); err != nil {
		return fmt.Errorf("failed to create sessions subdirectory: %w", err)
	}

	// 5. Initialize agent loop
	agentLoop, err := loop.NewLoop(loop.Config{
		Workspace:         ws.Path(),
		SessionDir:        ws.Subpath("sessions"),
		LLMProvider:       provider,
		Logger:            a.logger,
		Model:             a.config.Agent.Model,
		MaxTokens:         a.config.Agent.MaxTokens,
		Temperature:       a.config.Agent.Temperature,
		MaxToolIterations: a.config.Agent.MaxIterations,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent loop: %w", err)
	}
	a.agentLoop = agentLoop

	// 6. Create command handler
	a.commandHandler = commands.NewHandler(
		a.agentLoop,
		a.messageBus,
		a.logger,
		a.Restart,
	)

	// 7. Register tools
	// Register shell tool if enabled
	if a.config.Tools.Shell.Enabled {
		shellTool := tools.NewShellExecTool(a.config, a.logger)
		a.agentLoop.RegisterTool(shellTool)
	}

	// Register file tools if enabled
	if a.config.Tools.File.Enabled {
		readFileTool := tools.NewReadFileTool(ws)
		a.agentLoop.RegisterTool(readFileTool)

		writeFileTool := tools.NewWriteFileTool(ws)
		a.agentLoop.RegisterTool(writeFileTool)

		listDirTool := tools.NewListDirTool(ws)
		a.agentLoop.RegisterTool(listDirTool)
	}

	// 8. Initialize telegram connector if enabled
	if a.config.Channels.Telegram.Enabled {
		a.telegram = telegram.New(
			a.config.Channels.Telegram,
			a.logger,
			a.messageBus,
			provider,
		)
		if err := a.telegram.Start(a.ctx); err != nil {
			return fmt.Errorf("failed to start telegram connector: %w", err)
		}
	}

	// 9. Initialize cron scheduler if enabled
	if a.config.Cron.Enabled {
		// For now, pass nil as worker pool - will be implemented later
		cronStorage := cron.NewStorage(ws.Path(), a.logger)
		a.cronScheduler = cron.NewScheduler(a.logger, a.messageBus, nil, cronStorage)
		if err := a.cronScheduler.Start(a.ctx); err != nil {
			return fmt.Errorf("failed to start cron scheduler: %w", err)
		}
	}

	// 10. Mark as started
	a.mu.Lock()
	a.started = true
	a.mu.Unlock()

	return nil
}
