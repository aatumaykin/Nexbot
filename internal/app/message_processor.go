// Package app provides message processing logic for Nexbot.
// This file implements StartMessageProcessing and processMessage methods.
package app

import (
	"context"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/messages"
)

// StartMessageProcessing starts the message processing loop.
// It subscribes to inbound messages and processes them in a goroutine.
func (a *App) StartMessageProcessing(ctx context.Context) error {
	// Subscribe to inbound messages from the message bus
	inboundCh := a.messageBus.SubscribeInbound(ctx)
	if inboundCh == nil {
		a.logger.ErrorCtx(ctx, "Failed to subscribe to inbound messages: channel is nil", nil)
		return nil
	}

	// Start goroutine to process messages
	go func() {
		a.logger.Info("Message processing started")
		for {
			select {
			case <-ctx.Done():
				a.logger.Info("Message processing stopped")
				return
			case msg, ok := <-inboundCh:
				if !ok {
					a.logger.Info("Inbound channel closed")
					return
				}
				// Process message (don't block on errors)
				a.processMessage(ctx, msg)
			}
		}
	}()

	return nil
}

// processMessage processes a single inbound message.
// It handles commands, publishes events, and processes through the agent loop.
func (a *App) processMessage(ctx context.Context, msg bus.InboundMessage) {
	// Log message processing start
	a.logger.InfoCtx(ctx, "Processing message",
		logger.Field{Key: "user_id", Value: msg.UserID},
		logger.Field{Key: "session_id", Value: msg.SessionID})

	// Check if message contains a command in metadata
	var cmd string
	if msg.Metadata != nil {
		if cmdVal, ok := msg.Metadata["command"].(string); ok {
			cmd = cmdVal
		}
	}

	// Handle command if present
	if cmd != "" {
		a.logger.InfoCtx(ctx, "Command received",
			logger.Field{Key: "command", Value: cmd},
			logger.Field{Key: "session_id", Value: msg.SessionID})

		err := a.commandHandler.HandleCommand(ctx, cmd, msg)
		if err != nil {
			a.logger.ErrorCtx(ctx, "Failed to handle command", err,
				logger.Field{Key: "command", Value: cmd},
				logger.Field{Key: "session_id", Value: msg.SessionID})
		}

		// Return early for commands (don't process further)
		return
	}

	// Publish processing start event
	startEvent := bus.NewProcessingStartEvent(msg.ChannelType, msg.UserID, msg.SessionID, nil)
	if err := a.messageBus.PublishEvent(*startEvent); err != nil {
		a.logger.ErrorCtx(ctx, "Failed to publish processing start event", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
	}

	// Create context with timeout for agent processing
	cfg := a.config
	agentCtx, cancel := context.WithTimeout(ctx,
		time.Duration(cfg.Agent.TimeoutSeconds)*time.Second)

	// Process message through agent loop
	response, err := a.agentLoop.Process(agentCtx, msg.SessionID, msg.Content)
	cancel()

	// Handle error
	if err != nil {
		a.logger.ErrorCtx(ctx, "Failed to process message through agent", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
		response = messages.FormatError(err)
	}

	// Publish processing end event
	endEvent := bus.NewProcessingEndEvent(msg.ChannelType, msg.UserID, msg.SessionID, nil)
	if err := a.messageBus.PublishEvent(*endEvent); err != nil {
		a.logger.ErrorCtx(ctx, "Failed to publish processing end event", err,
			logger.Field{Key: "session_id", Value: msg.SessionID})
	}

	// Send response if non-empty
	if response != "" {
		outboundMsg := bus.NewOutboundMessage(
			msg.ChannelType,
			msg.UserID,
			msg.SessionID,
			response,
			nil,
		)
		if err := a.messageBus.PublishOutbound(*outboundMsg); err != nil {
			a.logger.ErrorCtx(ctx, "Failed to publish outbound message", err,
				logger.Field{Key: "session_id", Value: msg.SessionID})
		}
	}
}
