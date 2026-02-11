// Package commands provides command handling for Telegram messages.
package commands

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/messages"
)

// AgentLoopInterface defines the interface for agent loop operations needed by Handler
type AgentLoopInterface interface {
	ClearSession(ctx context.Context, sessionID string) error
	GetSessionStatus(ctx context.Context, sessionID string) (map[string]any, error)
}

// MessageBusInterface defines the interface for message bus operations needed by Handler
type MessageBusInterface interface {
	PublishOutbound(msg bus.OutboundMessage) error
}

// Handler handles Telegram commands for the agent.
type Handler struct {
	agentLoop  AgentLoopInterface
	messageBus MessageBusInterface
	logger     *logger.Logger
	onRestart  func() error
}

// NewHandler creates a new command handler.
func NewHandler(
	agentLoop AgentLoopInterface,
	messageBus MessageBusInterface,
	log *logger.Logger,
	onRestart func() error,
) *Handler {
	return &Handler{
		agentLoop:  agentLoop,
		messageBus: messageBus,
		logger:     log,
		onRestart:  onRestart,
	}
}

// HandleCommand processes a command based on its type.
func (h *Handler) HandleCommand(ctx context.Context, cmd string, msg bus.InboundMessage) error {
	switch cmd {
	case constants.CommandNewSession:
		return h.handleNewSession(ctx, msg)
	case constants.CommandStatus:
		return h.handleStatus(ctx, msg)
	case constants.CommandRestart:
		return h.handleRestart(ctx, msg)
	default:
		h.logger.WarnCtx(ctx, "Unknown command",
			logger.Field{Key: "command", Value: cmd},
			logger.Field{Key: "session_id", Value: msg.SessionID})
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// handleNewSession clears the current session.
func (h *Handler) handleNewSession(ctx context.Context, msg bus.InboundMessage) error {
	h.logger.InfoCtx(ctx, "Clearing session",
		logger.Field{Key: "session_id", Value: msg.SessionID})

	if err := h.agentLoop.ClearSession(ctx, msg.SessionID); err != nil {
		h.logger.ErrorCtx(ctx, "Failed to clear session", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
		return fmt.Errorf("failed to clear session: %w", err)
	}

	// Send confirmation message
	confirmationMsg := bus.NewOutboundMessage(
		msg.ChannelType,
		msg.UserID,
		msg.SessionID,
		constants.MsgSessionCleared,
		"", // correlationID (not used for commands)
		bus.FormatTypePlain,
		nil, // metadata
	)

	if err := h.messageBus.PublishOutbound(*confirmationMsg); err != nil {
		h.logger.ErrorCtx(ctx, "Failed to publish session cleared message", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
		return fmt.Errorf("failed to publish session cleared message: %w", err)
	}

	return nil
}

// handleStatus retrieves and displays the current session status.
func (h *Handler) handleStatus(ctx context.Context, msg bus.InboundMessage) error {
	h.logger.InfoCtx(ctx, "Getting status for session",
		logger.Field{Key: "session_id", Value: msg.SessionID})

	status, err := h.agentLoop.GetSessionStatus(ctx, msg.SessionID)
	if err != nil {
		h.logger.ErrorCtx(ctx, "Failed to get session status", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})

		// Send error message
		errorMsg := bus.NewOutboundMessage(
			msg.ChannelType,
			msg.UserID,
			msg.SessionID,
			constants.MsgStatusError,
			"", // correlationID (not used for commands)
			bus.FormatTypePlain,
			nil, // metadata
		)

		if pubErr := h.messageBus.PublishOutbound(*errorMsg); pubErr != nil {
			return fmt.Errorf("failed to get status and failed to publish error message: %w (publish error: %v)", err, pubErr)
		}
		return fmt.Errorf("failed to get session status: %w", err)
	}

	// Format status message
	sessionID, _ := status["session_id"].(string)
	messageCount, _ := status["message_count"].(int)
	fileSizeHuman, _ := status["file_size_human"].(string)
	model, _ := status["model"].(string)
	temperature, _ := status["temperature"].(float64)
	maxTokens, _ := status["max_tokens"].(int)

	statusMsg := messages.FormatStatusMessage(
		sessionID,
		messageCount,
		fileSizeHuman,
		model,
		temperature,
		maxTokens,
	)

	// Send status message
	outboundMsg := bus.NewOutboundMessage(
		msg.ChannelType,
		msg.UserID,
		msg.SessionID,
		statusMsg,
		"", // correlationID (not used for commands)
		bus.FormatTypePlain,
		nil, // metadata
	)

	if err := h.messageBus.PublishOutbound(*outboundMsg); err != nil {
		h.logger.ErrorCtx(ctx, "Failed to publish status message", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
		return fmt.Errorf("failed to publish status message: %w", err)
	}

	return nil
}

// handleRestart restarts the agent.
func (h *Handler) handleRestart(ctx context.Context, msg bus.InboundMessage) error {
	h.logger.InfoCtx(ctx, "Restart command received",
		logger.Field{Key: "session_id", Value: msg.SessionID})

	// Send notification message
	notificationMsg := bus.NewOutboundMessage(
		msg.ChannelType,
		msg.UserID,
		msg.SessionID,
		constants.MsgRestarting,
		"", // correlationID (not used for commands)
		bus.FormatTypePlain,
		nil, // metadata
	)

	if err := h.messageBus.PublishOutbound(*notificationMsg); err != nil {
		h.logger.ErrorCtx(ctx, "Failed to publish restarting message", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
		return fmt.Errorf("failed to publish restarting message: %w", err)
	}

	// Call restart callback
	if h.onRestart != nil {
		if err := h.onRestart(); err != nil {
			return fmt.Errorf("restart callback failed: %w", err)
		}
	}

	return nil
}
