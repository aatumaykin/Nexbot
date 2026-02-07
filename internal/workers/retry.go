package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// executeTask dispatches task execution based on type.
func (p *WorkerPool) executeTask(ctx context.Context, task Task) Result {
	// Handle context cancellation before execution
	select {
	case <-ctx.Done():
		return Result{
			TaskID: task.ID,
			Error:  ctx.Err(),
		}
	default:
	}

	// Execute based on task type
	switch task.Type {
	case "cron":
		return p.executeCronTask(ctx, task)
	case "subagent":
		return p.executeSubagentTask(ctx, task)
	default:
		return Result{
			TaskID: task.ID,
			Error:  fmt.Errorf("unknown task type: %s", task.Type),
		}
	}
}

// executeWithRetry executes a task with panic recovery and context cancellation
func (p *WorkerPool) executeWithRetry(ctx context.Context, task Task, executor TaskExecutor) Result {
	done := make(chan struct{})
	var output string
	var err error

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic during task execution: %v", r)
				p.logger.ErrorCtx(ctx, "task panic recovered", fmt.Errorf("panic: %v", r),
					logger.Field{Key: "task_id", Value: task.ID})
			}
		}()

		output, err = executor(ctx, task)
	}()

	select {
	case <-done:
		return Result{
			TaskID: task.ID,
			Output: output,
			Error:  err,
		}
	case <-ctx.Done():
		return Result{
			TaskID: task.ID,
			Error:  ctx.Err(),
		}
	}
}

// executeCronTask executes a cron-scheduled task.
func (p *WorkerPool) executeCronTask(ctx context.Context, task Task) Result {
	return p.executeWithRetry(ctx, task, func(ctx context.Context, t Task) (string, error) {
		// Unmarshal CronTaskPayload from task.Payload
		payload, ok := task.Payload.(cron.CronTaskPayload)
		if !ok {
			return "", fmt.Errorf("invalid cron task payload: expected CronTaskPayload")
		}

		fields := []logger.Field{
			{Key: "task_id", Value: task.ID},
			{Key: "command", Value: payload.Command},
			{Key: "tool", Value: payload.Tool},
			{Key: "session_id", Value: payload.SessionID},
		}

		p.logger.DebugCtx(ctx, "executing cron task", fields...)

		// Determine session ID
		sessionID := payload.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("cron_%s", task.ID)
		}

		// Dispatch based on tool type
		switch payload.Tool {
		case "send_message":
			return p.executeSendMessage(ctx, task, payload, sessionID)
		case "agent":
			return p.executeAgent(ctx, task, payload, sessionID)
		default:
			return "", fmt.Errorf("unsupported tool type: '%s'. Supported tools: 'send_message', 'agent'. Empty tool is deprecated", payload.Tool)
		}
	})
}

// executeSendMessage handles the send_message tool - publishes outbound message directly
func (p *WorkerPool) executeSendMessage(ctx context.Context, task Task, payload cron.CronTaskPayload, sessionID string) (string, error) {
	p.logger.DebugCtx(ctx, "executing send_message tool",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "session_id", Value: sessionID})

	// Parse channel and chat ID from session_id
	var channel, chatID string
	if strings.Contains(sessionID, ":") {
		parts := strings.Split(sessionID, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid session_id format: expected 'channel:chat_id', got '%s'", sessionID)
		}
		channel = parts[0]
		chatID = parts[1]
	} else {
		return "", fmt.Errorf("invalid session_id format: expected 'channel:chat_id', got '%s'", sessionID)
	}

	// Extract message content from payload or command
	content := payload.Command
	if payload.Payload != nil {
		if msg, ok := payload.Payload["message"].(string); ok {
			content = msg
		}
	}

	if content == "" {
		return "", fmt.Errorf("no message content provided")
	}

	// Create outbound message
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelType(channel),
		UserID:      "",
		SessionID:   chatID,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"cron_job_id": task.ID,
		},
	}

	if err := p.messageBus.PublishOutbound(outboundMsg); err != nil {
		p.logger.ErrorCtx(ctx, "failed to publish outbound message", err,
			logger.Field{Key: "task_id", Value: task.ID})
		return "", fmt.Errorf("failed to publish outbound message: %w", err)
	}

	p.logger.InfoCtx(ctx, "send_message tool executed successfully",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "channel", Value: channel},
		logger.Field{Key: "chat_id", Value: chatID})

	return fmt.Sprintf("message sent to %s:%s", channel, chatID), nil
}

// executeAgent handles the agent tool - publishes inbound message for agent processing
func (p *WorkerPool) executeAgent(ctx context.Context, task Task, payload cron.CronTaskPayload, sessionID string) (string, error) {
	p.logger.DebugCtx(ctx, "executing agent tool",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "session_id", Value: sessionID})

	// Parse channel and chat ID from session_id
	var channel, chatID string
	if strings.Contains(sessionID, ":") {
		parts := strings.Split(sessionID, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid session_id format: expected 'channel:chat_id', got '%s'", sessionID)
		}
		channel = parts[0]
		chatID = parts[1]
	} else {
		return "", fmt.Errorf("invalid session_id format: expected 'channel:chat_id', got '%s'", sessionID)
	}

	// Extract message content from payload
	content := payload.Command
	if payload.Payload != nil {
		if msg, ok := payload.Payload["message"].(string); ok {
			content = msg
		}
	}

	if content == "" {
		return "", fmt.Errorf("no message content provided in payload")
	}

	// Create inbound message for agent processing
	msg := bus.NewInboundMessage(
		bus.ChannelType(channel),
		"", // Empty user_id for cron tasks
		chatID,
		content,
		map[string]interface{}{
			"cron_job_id": task.ID,
			"tool":        "agent",
			"payload":     payload.Payload,
		},
	)

	if err := p.messageBus.PublishInbound(*msg); err != nil {
		p.logger.ErrorCtx(ctx, "failed to publish inbound message for agent", err,
			logger.Field{Key: "task_id", Value: task.ID})
		return "", fmt.Errorf("failed to publish inbound message: %w", err)
	}

	p.logger.InfoCtx(ctx, "agent tool executed successfully",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "channel", Value: channel},
		logger.Field{Key: "chat_id", Value: chatID})

	return fmt.Sprintf("agent message sent to %s:%s", channel, chatID), nil
}

// executeSubagentTask executes a subagent task.
func (p *WorkerPool) executeSubagentTask(ctx context.Context, task Task) Result {
	return p.executeWithRetry(ctx, task, func(ctx context.Context, t Task) (string, error) {
		p.logger.DebugCtx(ctx, "executing subagent task",
			logger.Field{Key: "task_id", Value: task.ID},
			logger.Field{Key: "payload", Value: task.Payload})

		p.logger.InfoCtx(ctx, "subagent task completed",
			logger.Field{Key: "task_id", Value: task.ID})

		return fmt.Sprintf("subagent task completed with payload: %v", task.Payload), nil
	})
}
