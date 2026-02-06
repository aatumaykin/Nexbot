package loop

import (
	stdcontext "context"
	"fmt"
	"os"

	agentcontext "github.com/aatumaykin/nexbot/internal/agent/context"
	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
)

// Loop manages the agent's execution loop, coordinating between
// LLM provider, session management, and tools.
type Loop struct {
	workspace    string
	sessionDir   string
	sessionMgr   *session.Manager
	sessionOps   *SessionOperations
	contextBldr  *agentcontext.Builder
	provider     llm.Provider
	logger       *logger.Logger
	tools        *tools.Registry
	toolExecutor *ToolExecutor
	config       Config
}

// Config holds configuration for the loop.
type Config struct {
	Workspace         string
	SessionDir        string
	LLMProvider       llm.Provider
	Logger            *logger.Logger
	Model             string
	MaxTokens         int
	Temperature       float64
	MaxToolIterations int
}

// NewLoop creates a new execution loop.
func NewLoop(cfg Config) (*Loop, error) {
	// Validate configuration
	if cfg.Workspace == "" {
		return nil, fmt.Errorf("workspace path cannot be empty")
	}
	if cfg.SessionDir == "" {
		return nil, fmt.Errorf("session directory cannot be empty")
	}
	if cfg.LLMProvider == nil {
		return nil, fmt.Errorf("LLM provider cannot be nil")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	if cfg.MaxToolIterations == 0 {
		cfg.MaxToolIterations = 10
	}

	// Create session manager
	sessionMgr, err := session.NewManager(cfg.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create context builder
	contextBldr, err := agentcontext.NewBuilder(agentcontext.Config{
		Workspace: cfg.Workspace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create context builder: %w", err)
	}

	// Create tool registry
	toolRegistry := tools.NewRegistry()

	// Create tool executor
	toolExecutor := NewToolExecutor(cfg.Logger, toolRegistry)

	// Create session operations
	sessionOps := NewSessionOperations(sessionMgr)

	return &Loop{
		workspace:    cfg.Workspace,
		sessionDir:   cfg.SessionDir,
		sessionMgr:   sessionMgr,
		sessionOps:   sessionOps,
		contextBldr:  contextBldr,
		provider:     cfg.LLMProvider,
		logger:       cfg.Logger,
		tools:        toolRegistry,
		toolExecutor: toolExecutor,
		config:       cfg,
	}, nil
}

// Process handles a user message and returns the assistant's response.
// This is the main entry point for the agent loop.
func (l *Loop) Process(ctx stdcontext.Context, sessionID, userMessage string) (string, error) {
	l.logger.DebugCtx(ctx, "Processing user message",
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "message_length", Value: len(userMessage)})

	// Add user message to session
	if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
		Role:    llm.RoleUser,
		Content: userMessage,
	}); err != nil {
		return "", fmt.Errorf("failed to add user message: %w", err)
	}

	// Process message with tool calling support
	response, err := l.processWithToolCalling(ctx, sessionID, 0)
	if err != nil {
		l.logger.ErrorCtx(ctx, "Failed to process message", err,
			logger.Field{Key: "session_id", Value: sessionID})
		// Return a graceful error message instead of failing
		return fmt.Sprintf("I encountered an error processing your message: %v", err), nil
	}

	return response, nil
}

// processWithToolCalling processes a message, handling tool calls recursively.
func (l *Loop) processWithToolCalling(ctx stdcontext.Context, sessionID string, iteration int) (string, error) {
	// Prevent infinite loops
	maxIterations := l.config.MaxToolIterations
	if iteration >= maxIterations {
		l.logger.ErrorCtx(ctx, "Maximum tool call iterations reached", nil,
			logger.Field{Key: "iterations", Value: iteration})
		return "", fmt.Errorf("reached maximum tool call iterations (%d)", maxIterations)
	}

	// Prepare request
	sessionHistory, err := l.sessionOps.GetSessionHistory(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session history: %w", err)
	}

	// Build system prompt (only on first iteration)
	messages := sessionHistory
	if iteration == 0 {
		systemPrompt, err := l.buildSystemPrompt()
		if err != nil {
			l.logger.WarnCtx(ctx, "Failed to build system prompt",
				logger.Field{Key: "error", Value: err.Error()})
		} else if systemPrompt != "" {
			messages = append([]llm.Message{{
				Role:    llm.RoleSystem,
				Content: systemPrompt,
			}}, sessionHistory...)
		}
	}

	// Create LLM request
	req := llm.ChatRequest{
		Messages:    messages,
		Model:       l.config.Model,
		Temperature: l.config.Temperature,
		MaxTokens:   l.config.MaxTokens,
	}

	// Add tool definitions if provider supports them
	if l.provider.SupportsToolCalling() {
		toolSchemas := l.tools.ToSchema()
		if len(toolSchemas) > 0 {
			// Convert tools.ToolDefinition to llm.ToolDefinition
			llmTools := make([]llm.ToolDefinition, len(toolSchemas))
			for i, schema := range toolSchemas {
				llmTools[i] = llm.ToolDefinition{
					Name:        schema.Name,
					Description: schema.Description,
					Parameters:  schema.Parameters,
				}
			}
			req.Tools = llmTools
			l.logger.DebugCtx(ctx, "Added tool definitions to request",
				logger.Field{Key: "tool_count", Value: len(llmTools)},
				logger.Field{Key: "tools", Value: fmt.Sprintf("%+v", llmTools)})
		}
	}

	// Call LLM
	resp, err := l.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Debug: response received
	l.logger.DebugCtx(ctx, "LLM response received",
		logger.Field{Key: "finish_reason", Value: resp.FinishReason},
		logger.Field{Key: "content_length", Value: len(resp.Content)},
		logger.Field{Key: "tool_calls_count", Value: len(resp.ToolCalls)},
		logger.Field{Key: "iteration", Value: iteration})

	// Handle tool calls
	if resp.FinishReason == llm.FinishReasonToolCalls && len(resp.ToolCalls) > 0 {
		l.logger.DebugCtx(ctx, "LLM requested tool calls",
			logger.Field{Key: "tool_call_count", Value: len(resp.ToolCalls)},
			logger.Field{Key: "iteration", Value: iteration})

		// Add assistant message with tool calls to session
		if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
			Role:    llm.RoleAssistant,
			Content: resp.Content,
		}); err != nil {
			return "", fmt.Errorf("failed to add assistant message: %w", err)
		}

		// Prepare tool calls for execution
		toolCalls := l.toolExecutor.PrepareToolCalls(resp.ToolCalls)

		// Execute tools
		results, err := l.toolExecutor.ProcessToolCalls(ctx, toolCalls)
		if err != nil {
			return "", fmt.Errorf("failed to execute tools: %w", err)
		}

		// Add tool results to session
		for _, result := range results {
			content := result.Content
			if result.Error != "" {
				content = fmt.Sprintf("Error: %s", result.Error)
			}
			if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
				Role:       llm.RoleTool,
				Content:    content,
				ToolCallID: result.ToolCallID,
			}); err != nil {
				return "", fmt.Errorf("failed to add tool result: %w", err)
			}
		}

		// Recursively process again with tool results
		l.logger.DebugCtx(ctx, "Recursively processing with tool results",
			logger.Field{Key: "next_iteration", Value: iteration + 1})
		return l.processWithToolCalling(ctx, sessionID, iteration+1)
	}

	// Normal response without tool calls
	l.logger.DebugCtx(ctx, "Returning final response",
		logger.Field{Key: "response_length", Value: len(resp.Content)},
		logger.Field{Key: "iteration", Value: iteration})
	if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
		Role:    llm.RoleAssistant,
		Content: resp.Content,
	}); err != nil {
		return "", fmt.Errorf("failed to add assistant message: %w", err)
	}

	return resp.Content, nil
}

// buildSystemPrompt builds the system prompt from workspace context.
func (l *Loop) buildSystemPrompt() (string, error) {
	return l.contextBldr.Build()
}

// AddMessageToSession adds a message to the session history.
// This is a public wrapper for compatibility.
func (l *Loop) AddMessageToSession(ctx stdcontext.Context, sessionID string, message llm.Message) error {
	return l.sessionOps.AddMessageToSession(ctx, sessionID, message)
}

// GetSessionHistory returns the message history for a session.
// This is a public wrapper for compatibility.
func (l *Loop) GetSessionHistory(ctx stdcontext.Context, sessionID string) ([]llm.Message, error) {
	return l.sessionOps.GetSessionHistory(ctx, sessionID)
}

// ClearSession clears all messages from a session.
// This is a public wrapper for compatibility.
func (l *Loop) ClearSession(ctx stdcontext.Context, sessionID string) error {
	return l.sessionOps.ClearSession(ctx, sessionID)
}

// DeleteSession deletes a session entirely.
// This is a public wrapper for compatibility.
func (l *Loop) DeleteSession(ctx stdcontext.Context, sessionID string) error {
	return l.sessionOps.DeleteSession(ctx, sessionID)
}

// GetContextBuilder returns the context builder.
func (l *Loop) GetContextBuilder() *agentcontext.Builder {
	return l.contextBldr
}

// GetSessionManager returns the session manager.
func (l *Loop) GetSessionManager() *session.Manager {
	return l.sessionMgr
}

// GetLLMProvider returns the LLM provider.
func (l *Loop) GetLLMProvider() llm.Provider {
	return l.provider
}

// GetSessionModel returns the model for the given session (always returns config model).
func (l *Loop) GetSessionModel(ctx stdcontext.Context, sessionID string) string {
	return l.config.Model
}

// GetSessionMaxTokens returns the max tokens for the given session (always returns config max tokens).
func (l *Loop) GetSessionMaxTokens(sessionID string) int {
	return l.config.MaxTokens
}

// RegisterTool registers a tool with the loop's tool registry.
func (l *Loop) RegisterTool(tool tools.Tool) error {
	if err := l.tools.Register(tool); err != nil {
		return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
	}
	l.logger.DebugCtx(stdcontext.Background(), "Tool registered",
		logger.Field{Key: "tool_name", Value: tool.Name()})
	return nil
}

// GetTools returns the tool registry.
func (l *Loop) GetTools() *tools.Registry {
	return l.tools
}

// ProcessHeartbeatCheck processes a heartbeat check request by consulting HEARTBEAT.md.
// This is used by cron to periodically check if there are any tasks that need attention.
func (l *Loop) ProcessHeartbeatCheck(ctx stdcontext.Context) (string, error) {
	l.logger.InfoCtx(ctx, "Processing heartbeat check")

	// Build prompt for heartbeat check
	prompt := "Read HEARTBEAT.md from workspace. Follow it strictly. Do not infer or repeat old tasks from prior chats. If nothing needs attention, reply HEARTBEAT_OK."

	// Create chat request for LLM
	req := llm.ChatRequest{
		Messages:    []llm.Message{{Role: llm.RoleUser, Content: prompt}},
		Model:       l.config.Model,
		Temperature: l.config.Temperature,
		MaxTokens:   l.config.MaxTokens,
	}

	// Call LLM provider
	resp, err := l.provider.Chat(ctx, req)
	if err != nil {
		l.logger.ErrorCtx(ctx, "Failed to get heartbeat check response from LLM", err)
		return "", fmt.Errorf("failed to get heartbeat check response: %w", err)
	}

	// Log the response
	l.logger.InfoCtx(ctx, "Heartbeat check response",
		logger.Field{Key: "response", Value: resp.Content})

	// Return the LLM response
	return resp.Content, nil
}

// GetSessionStatus returns status information about a session.
// This is a public wrapper for compatibility.
func (l *Loop) GetSessionStatus(ctx stdcontext.Context, sessionID string) (map[string]any, error) {
	return l.sessionOps.GetSessionStatus(ctx, sessionID, l)
}

func getFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
