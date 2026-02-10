package builders

import (
	"fmt"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/tools/fetch"
	"github.com/aatumaykin/nexbot/internal/tools/file"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

type ToolsBuilder struct {
	config     *config.Config
	logger     *logger.Logger
	workspace  *workspace.Workspace
	messageBus *bus.MessageBus
}

func NewToolsBuilder(cfg *config.Config, log *logger.Logger, ws *workspace.Workspace, mb *bus.MessageBus) *ToolsBuilder {
	return &ToolsBuilder{
		config:     cfg,
		logger:     log,
		workspace:  ws,
		messageBus: mb,
	}
}

func (b *ToolsBuilder) RegisterAllTools(agentLoop *loop.Loop) error {
	if err := b.RegisterSendMessageTool(agentLoop); err != nil {
		return err
	}

	if err := b.RegisterSystemTimeTool(agentLoop); err != nil {
		return err
	}

	if b.config.Tools.Shell.Enabled {
		if err := b.RegisterShellTool(agentLoop); err != nil {
			return err
		}
	}

	if b.config.Tools.File.Enabled {
		if err := b.RegisterFileTools(agentLoop); err != nil {
			return err
		}
	}

	if b.config.Tools.Fetch.Enabled {
		if err := b.RegisterFetchTool(agentLoop); err != nil {
			return err
		}
	}

	return nil
}

func (b *ToolsBuilder) RegisterSendMessageTool(agentLoop *loop.Loop) error {
	messageSender := loop.NewAgentMessageSender(b.messageBus, b.logger)
	sendMessageTool := tools.NewSendMessageTool(messageSender, b.logger)
	if err := agentLoop.RegisterTool(sendMessageTool); err != nil {
		return fmt.Errorf("failed to register send message tool: %w", err)
	}
	b.logger.Info("Send message tool registered")
	return nil
}

func (b *ToolsBuilder) RegisterShellTool(agentLoop *loop.Loop) error {
	shellTool := tools.NewShellExecTool(b.config, b.logger)
	if err := agentLoop.RegisterTool(shellTool); err != nil {
		return fmt.Errorf("failed to register shell tool: %w", err)
	}
	return nil
}

func (b *ToolsBuilder) RegisterFileTools(agentLoop *loop.Loop) error {
	readFileTool := file.NewReadFileTool(b.workspace, b.config)
	if err := agentLoop.RegisterTool(readFileTool); err != nil {
		return fmt.Errorf("failed to register read file tool: %w", err)
	}

	writeFileTool := file.NewWriteFileTool(b.workspace, b.config)
	if err := agentLoop.RegisterTool(writeFileTool); err != nil {
		return fmt.Errorf("failed to register write file tool: %w", err)
	}

	listDirTool := file.NewListDirTool(b.workspace, b.config)
	if err := agentLoop.RegisterTool(listDirTool); err != nil {
		return fmt.Errorf("failed to register list dir tool: %w", err)
	}

	deleteFileTool := file.NewDeleteFileTool(b.workspace, b.config)
	if err := agentLoop.RegisterTool(deleteFileTool); err != nil {
		return fmt.Errorf("failed to register delete file tool: %w", err)
	}

	return nil
}

func (b *ToolsBuilder) RegisterFetchTool(agentLoop *loop.Loop) error {
	fetchTool := fetch.NewFetchTool(b.config, b.logger)
	if err := agentLoop.RegisterTool(fetchTool); err != nil {
		return fmt.Errorf("failed to register fetch tool: %w", err)
	}
	b.logger.Info("Fetch tool registered")
	return nil
}

func (b *ToolsBuilder) RegisterSystemTimeTool(agentLoop *loop.Loop) error {
	systemTimeTool := tools.NewSystemTimeTool(b.logger)
	if err := agentLoop.RegisterTool(systemTimeTool); err != nil {
		return fmt.Errorf("failed to register system time tool: %w", err)
	}
	b.logger.Info("System time tool registered")
	return nil
}

func (b *ToolsBuilder) RegisterSpawnTool(agentLoop *loop.Loop, spawnFunc tools.SpawnFunc) error {
	spawnTool := tools.NewSpawnTool(spawnFunc)
	if err := agentLoop.RegisterTool(spawnTool); err != nil {
		return fmt.Errorf("failed to register spawn tool: %w", err)
	}
	b.logger.Info("Spawn tool registered")
	return nil
}
