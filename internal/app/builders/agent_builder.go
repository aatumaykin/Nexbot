package builders

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/agent/subagent"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

type AgentBuilder struct {
	config    *config.Config
	logger    *logger.Logger
	provider  llm.Provider
	workspace *workspace.Workspace
}

func NewAgentBuilder(cfg *config.Config, log *logger.Logger, provider llm.Provider, ws *workspace.Workspace) *AgentBuilder {
	return &AgentBuilder{
		config:    cfg,
		logger:    log,
		provider:  provider,
		workspace: ws,
	}
}

func (b *AgentBuilder) BuildLoop() (*loop.Loop, error) {
	agentLoop, err := loop.NewLoop(loop.Config{
		Workspace:         b.workspace,
		WorkspaceCfg:      config.WorkspaceConfig{Path: b.workspace.Path()},
		SessionDir:        b.workspace.Subpath("sessions"),
		Timezone:          b.config.Cron.Timezone,
		LLMProvider:       b.provider,
		Logger:            b.logger,
		Model:             b.config.Agent.Model,
		MaxTokens:         b.config.Agent.MaxTokens,
		Temperature:       b.config.Agent.Temperature,
		MaxToolIterations: b.config.Agent.MaxIterations,
		SecretsDir:        b.config.SecretsDir(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent loop: %w", err)
	}
	return agentLoop, nil
}

func (b *AgentBuilder) BuildSubagentManager(agentLoop *loop.Loop) (*subagent.Manager, tools.SpawnFunc, error) {
	if !b.config.Subagent.Enabled {
		return nil, nil, nil
	}

	b.logger.Info("🧬 Initializing subagent manager")

	manager, err := subagent.NewManager(subagent.Config{
		SessionDir: b.workspace.Subpath("sessions"),
		Logger:     b.logger,
		LoopConfig: loop.Config{
			Workspace:         b.workspace,
			WorkspaceCfg:      config.WorkspaceConfig{Path: b.workspace.Path()},
			SessionDir:        b.workspace.Subpath("sessions"),
			LLMProvider:       b.provider,
			Logger:            b.logger,
			Model:             b.config.Agent.Model,
			MaxTokens:         b.config.Agent.MaxTokens,
			Temperature:       b.config.Agent.Temperature,
			MaxToolIterations: b.config.Agent.MaxIterations,
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize subagent manager: %w", err)
	}

	spawnFunc := func(ctx context.Context, parentSession string, task string) (string, error) {
		timeout := 300
		if deadline, ok := ctx.Deadline(); ok {
			timeout = int(time.Until(deadline).Seconds())
		}
		return manager.ExecuteTask(ctx, parentSession, task, timeout)
	}

	b.logger.Info("✅ Subagent manager initialized")

	return manager, spawnFunc, nil
}
