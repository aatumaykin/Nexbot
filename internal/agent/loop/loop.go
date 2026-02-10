package loop

import (
	stdcontext "context"
	"fmt"

	agentcontext "github.com/aatumaykin/nexbot/internal/agent/context"
	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/secrets"
	"github.com/aatumaykin/nexbot/internal/tools"
)

// contextKey is the type for context keys to avoid collisions
type contextKey struct{}

var (
	sessionIDKey contextKey = struct{}{}
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
	secrets      *secrets.Store
	config       Config
}

// Config holds configuration for the loop.
type Config struct {
	Workspace         string
	SessionDir        string
	Timezone          string
	LLMProvider       llm.Provider
	Logger            *logger.Logger
	Model             string
	MaxTokens         int
	Temperature       float64
	MaxToolIterations int
	SecretsDir        string
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

	// Create secrets store
	secretsStore := secrets.NewStore(cfg.SecretsDir)

	// Create context builder
	contextBldr, err := agentcontext.NewBuilder(agentcontext.Config{
		Workspace: cfg.Workspace,
		Timezone:  cfg.Timezone,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create context builder: %w", err)
	}

	// Create tool registry
	toolRegistry := tools.NewRegistry()

	// Create tool executor with secrets support
	toolExecutor := NewToolExecutor(cfg.Logger, toolRegistry, secretsStore)

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
		secrets:      secretsStore,
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
	if iteration >= l.config.MaxToolIterations {
		l.logger.ErrorCtx(ctx, "Maximum tool call iterations reached", nil,
			logger.Field{Key: "iterations", Value: iteration})
		return "", fmt.Errorf("reached maximum tool call iterations (%d)", l.config.MaxToolIterations)
	}

	// Prepare LLM request
	req, err := l.prepareLLMRequest(ctx, sessionID, iteration)
	if err != nil {
		return "", err
	}

	// Call LLM
	resp, err := l.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	l.logger.DebugCtx(ctx, "LLM response received",
		logger.Field{Key: "finish_reason", Value: resp.FinishReason},
		logger.Field{Key: "content_length", Value: len(resp.Content)},
		logger.Field{Key: "tool_calls_count", Value: len(resp.ToolCalls)},
		logger.Field{Key: "iteration", Value: iteration})

	// Handle tool calls or normal response
	if resp.FinishReason == llm.FinishReasonToolCalls && len(resp.ToolCalls) > 0 {
		return l.handleToolCalls(ctx, sessionID, iteration, *resp)
	}

	return l.handleNormalResponse(ctx, sessionID, *resp)
}

// prepareLLMRequest prepares the LLM chat request with context and tools.
func (l *Loop) prepareLLMRequest(ctx stdcontext.Context, sessionID string, iteration int) (llm.ChatRequest, error) {
	sessionHistory, err := l.sessionOps.GetSessionHistory(ctx, sessionID)
	if err != nil {
		return llm.ChatRequest{}, fmt.Errorf("failed to get session history: %w", err)
	}

	// Build system prompt (only on first iteration)
	messages := sessionHistory
	if iteration == 0 {
		systemPrompt, err := l.buildSystemPrompt(sessionID)
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

	return req, nil
}

// handleToolCalls processes tool calls from LLM response.
func (l *Loop) handleToolCalls(ctx stdcontext.Context, sessionID string, iteration int, resp llm.ChatResponse) (string, error) {
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

	// Add sessionID to context for secret resolution
	ctxWithSession := stdcontext.WithValue(ctx, sessionIDKey, sessionID)

	// Prepare and execute tool calls
	toolCalls := l.toolExecutor.PrepareToolCalls(resp.ToolCalls)
	results, err := l.toolExecutor.ProcessToolCalls(ctxWithSession, toolCalls)
	if err != nil {
		return "", fmt.Errorf("failed to execute tools: %w", err)
	}

	// Add tool results to session
	if err := l.addToolResultsToSession(ctx, sessionID, results); err != nil {
		return "", err
	}

	// Recursively process again with tool results
	l.logger.DebugCtx(ctx, "Recursively processing with tool results",
		logger.Field{Key: "next_iteration", Value: iteration + 1})
	return l.processWithToolCalling(ctx, sessionID, iteration+1)
}

// handleNormalResponse processes a normal LLM response without tool calls.
func (l *Loop) handleNormalResponse(ctx stdcontext.Context, sessionID string, resp llm.ChatResponse) (string, error) {
	l.logger.DebugCtx(ctx, "Returning final response",
		logger.Field{Key: "response_length", Value: len(resp.Content)},
		logger.Field{Key: "iteration", Value: resp.Content})
	if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
		Role:    llm.RoleAssistant,
		Content: resp.Content,
	}); err != nil {
		return "", fmt.Errorf("failed to add assistant message: %w", err)
	}

	return resp.Content, nil
}

// addToolResultsToSession adds tool execution results to the session history.
func (l *Loop) addToolResultsToSession(ctx stdcontext.Context, sessionID string, results []tools.ToolResult) error {
	for _, result := range results {
		var content string
		if result.Error != nil {
			if result.TimedOut {
				content = fmt.Sprintf("❌ Tool execution timed out\n\n%s", result.Error.ToLLMContext())
			} else {
				content = fmt.Sprintf("❌ Tool execution failed\n\n%s", result.Error.ToLLMContext())
			}
		} else {
			content = result.Content
		}

		if err := l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
			Role:       llm.RoleTool,
			Content:    content,
			ToolCallID: result.ToolCallID,
		}); err != nil {
			return fmt.Errorf("failed to add tool result: %w", err)
		}
	}
	return nil
}

// buildSystemPrompt builds the system prompt from workspace context.
func (l *Loop) buildSystemPrompt(sessionID string) (string, error) {
	systemPrompt, err := l.contextBldr.BuildForSession(sessionID, nil)
	if err != nil {
		return "", err
	}

	// Log system prompt for debugging
	var preview string
	if len(systemPrompt) > 500 {
		preview = systemPrompt[:500] + "..."
	} else {
		preview = systemPrompt
	}

	l.logger.Debug("System prompt built",
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "system_prompt_length", Value: len(systemPrompt)},
		logger.Field{Key: "preview", Value: preview})

	return systemPrompt, nil
}

// AddMessageToSession adds a message to the session history.
func (l *Loop) AddMessageToSession(ctx stdcontext.Context, sessionID string, message llm.Message) error {
	return l.sessionOps.AddMessageToSession(ctx, sessionID, message)
}

// GetSessionHistory returns the message history for a session.
func (l *Loop) GetSessionHistory(ctx stdcontext.Context, sessionID string) ([]llm.Message, error) {
	return l.sessionOps.GetSessionHistory(ctx, sessionID)
}

// ClearSession clears all messages from a session.
func (l *Loop) ClearSession(ctx stdcontext.Context, sessionID string) error {
	return l.sessionOps.ClearSession(ctx, sessionID)
}

// DeleteSession deletes a session entirely.
func (l *Loop) DeleteSession(ctx stdcontext.Context, sessionID string) error {
	return l.sessionOps.DeleteSession(ctx, sessionID)
}

// GetSessionStatus returns status information about a session.
func (l *Loop) GetSessionStatus(ctx stdcontext.Context, sessionID string) (map[string]any, error) {
	return l.sessionOps.GetSessionStatus(ctx, sessionID, l)
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

// GetSecretsStore returns the secrets store.
func (l *Loop) GetSecretsStore() *secrets.Store {
	return l.secrets
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

// AddErrorToSession adds an error message to the session history.
func (l *Loop) AddErrorToSession(ctx stdcontext.Context, sessionID string, err error) error {
	l.logger.ErrorCtx(ctx, "Adding error to session", err,
		logger.Field{Key: "session_id", Value: sessionID})
	errorMsg := fmt.Sprintf("**Error from previous attempt:**\n%s\n\nPlease analyze this error and suggest a solution.", err.Error())
	return l.sessionOps.AddMessageToSession(ctx, sessionID, llm.Message{
		Role:    llm.RoleUser,
		Content: errorMsg,
	})
}

// ProcessRecovery processes a recovery request after an error.
func (l *Loop) ProcessRecovery(ctx stdcontext.Context, sessionID string, originalErr error) (string, error) {
	l.logger.ErrorCtx(ctx, "Starting recovery processing", originalErr,
		logger.Field{Key: "session_id", Value: sessionID})

	// Build recovery prompt with length limit (500 chars)
	basePrompt := "The previous attempt failed. Please analyze this error and suggest a solution:"
	errText := originalErr.Error()

	// Limit error text to fit within 500 char total limit
	maxErrLen := 500 - len(basePrompt) - len("\n\n")
	if len(errText) > maxErrLen {
		errText = errText[:maxErrLen] + "..."
	}

	recoveryPrompt := fmt.Sprintf("%s\n\n%s", basePrompt, errText)

	// Process with normal timeout (not reduced)
	return l.Process(ctx, sessionID, recoveryPrompt)
}
