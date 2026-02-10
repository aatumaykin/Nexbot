package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/google/uuid"
)

// AgentMessageSender implements agent.MessageSender through the message bus.
// This bridges the Agent Layer's MessageSender interface with the Bus Layer.
type AgentMessageSender struct {
	messageBus *bus.MessageBus
	logger     *logger.Logger
}

// NewAgentMessageSender creates a new AgentMessageSender instance.
func NewAgentMessageSender(messageBus *bus.MessageBus, logger *logger.Logger) *AgentMessageSender {
	return &AgentMessageSender{
		messageBus: messageBus,
		logger:     logger,
	}
}

// SendMessage sends a message through the message bus and waits for result.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendMessage(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
	return a.SendMessageWithKeyboard(userID, channelType, sessionID, message, nil, timeout)
}

// SendMessageWithKeyboard sends a message with inline keyboard through the message bus and waits for result.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendMessageWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	// Use default timeout of 5 seconds if not provided
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Генерируем correlation ID
	correlationID := uuid.New().String()

	// Регистрируем ожидание результата
	tracker := a.messageBus.GetResultTracker()
	resultCh := tracker.Register(correlationID)

	// Публикуем сообщение в bus
	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewOutboundMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			message,
			correlationID,
			keyboard,
			nil, // metadata
		)
	} else {
		event = bus.NewOutboundMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			message,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		// Удаляем регистрацию при ошибке публикации
		a.logger.ErrorCtx(context.Background(), "failed to publish outbound message", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	// Ждем результат отправки с указанным timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Ждем результат напрямую через канал (более эффективно, чем через Wait)
	select {
	case result := <-resultCh:
		a.logger.DebugCtx(context.Background(), "message send result received",
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "success", Value: result.Success})

		// Возвращаем результат
		return &agent.MessageResult{
			Success:      result.Success,
			Error:        result.Error,
			ResponseText: "",
		}, nil
	case <-ctx.Done():
		a.logger.ErrorCtx(context.Background(), "timeout waiting for send result", ctx.Err(),
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "timeout", Value: timeout})
		return nil, fmt.Errorf("timeout waiting for send result: %w", ctx.Err())
	}
}

// SendEditMessage edits an existing message.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendEditMessage(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	// Use default timeout of 5 seconds if not provided
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Генерируем correlation ID
	correlationID := uuid.New().String()

	// Регистрируем ожидание результата
	tracker := a.messageBus.GetResultTracker()
	resultCh := tracker.Register(correlationID)

	// Публикуем сообщение в bus
	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewEditMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			messageID,
			content,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewEditMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			messageID,
			content,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish edit message", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType},
			logger.Field{Key: "message_id", Value: messageID})
		return nil, fmt.Errorf("failed to publish edit message: %w", err)
	}

	// Ждем результат с указанным timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case result := <-resultCh:
		return &agent.MessageResult{
			Success:      result.Success,
			Error:        result.Error,
			ResponseText: "",
		}, nil
	case <-ctx.Done():
		a.logger.ErrorCtx(context.Background(), "timeout waiting for edit message result", ctx.Err(),
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "timeout", Value: timeout})
		return nil, fmt.Errorf("timeout waiting for edit message result: %w", ctx.Err())
	}
}

// SendDeleteMessage deletes an existing message.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendDeleteMessage(userID, channelType, sessionID, messageID string, timeout time.Duration) (*agent.MessageResult, error) {
	// Use default timeout of 5 seconds if not provided
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Генерируем correlation ID
	correlationID := uuid.New().String()

	// Регистрируем ожидание результата
	tracker := a.messageBus.GetResultTracker()
	resultCh := tracker.Register(correlationID)

	// Публикуем сообщение в bus
	event := bus.NewDeleteMessage(
		bus.ChannelType(channelType),
		userID,
		sessionID,
		messageID,
		correlationID,
		nil, // metadata
	)

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish delete message", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType},
			logger.Field{Key: "message_id", Value: messageID})
		return nil, fmt.Errorf("failed to publish delete message: %w", err)
	}

	// Ждем результат с указанным timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case result := <-resultCh:
		return &agent.MessageResult{
			Success:      result.Success,
			Error:        result.Error,
			ResponseText: "",
		}, nil
	case <-ctx.Done():
		a.logger.ErrorCtx(context.Background(), "timeout waiting for delete message result", ctx.Err(),
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "timeout", Value: timeout})
		return nil, fmt.Errorf("timeout waiting for delete message result: %w", ctx.Err())
	}
}

// SendPhotoMessage sends a photo message.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendPhotoMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	// Use default timeout of 5 seconds if not provided
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Генерируем correlation ID
	correlationID := uuid.New().String()

	// Регистрируем ожидание результата
	tracker := a.messageBus.GetResultTracker()
	resultCh := tracker.Register(correlationID)

	// Публикуем сообщение в bus
	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewPhotoMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewPhotoMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish photo message", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return nil, fmt.Errorf("failed to publish photo message: %w", err)
	}

	// Ждем результат с указанным timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case result := <-resultCh:
		return &agent.MessageResult{
			Success:      result.Success,
			Error:        result.Error,
			ResponseText: "",
		}, nil
	case <-ctx.Done():
		a.logger.ErrorCtx(context.Background(), "timeout waiting for photo message result", ctx.Err(),
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "timeout", Value: timeout})
		return nil, fmt.Errorf("timeout waiting for photo message result: %w", ctx.Err())
	}
}

// SendDocumentMessage sends a document message.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendDocumentMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	// Use default timeout of 5 seconds if not provided
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Генерируем correlation ID
	correlationID := uuid.New().String()

	// Регистрируем ожидание результата
	tracker := a.messageBus.GetResultTracker()
	resultCh := tracker.Register(correlationID)

	// Публикуем сообщение в bus
	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewDocumentMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewDocumentMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish document message", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return nil, fmt.Errorf("failed to publish document message: %w", err)
	}

	// Ждем результат с указанным timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case result := <-resultCh:
		return &agent.MessageResult{
			Success:      result.Success,
			Error:        result.Error,
			ResponseText: "",
		}, nil
	case <-ctx.Done():
		a.logger.ErrorCtx(context.Background(), "timeout waiting for document message result", ctx.Err(),
			logger.Field{Key: "correlation_id", Value: correlationID},
			logger.Field{Key: "timeout", Value: timeout})
		return nil, fmt.Errorf("timeout waiting for document message result: %w", ctx.Err())
	}
}

// SendMessageAsync sends a message asynchronously (fire-and-forget) without waiting for result.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendMessageAsync(userID, channelType, sessionID, message string) error {
	return a.SendMessageAsyncWithKeyboard(userID, channelType, sessionID, message, nil)
}

// SendMessageAsyncWithKeyboard sends a message with inline keyboard asynchronously.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendMessageAsyncWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard) error {
	correlationID := uuid.New().String()

	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewOutboundMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			message,
			correlationID,
			keyboard,
			nil, // metadata
		)
	} else {
		event = bus.NewOutboundMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			message,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish outbound message (async)", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// SendEditMessageAsync edits an existing message asynchronously.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendEditMessageAsync(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard) error {
	correlationID := uuid.New().String()

	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewEditMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			messageID,
			content,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewEditMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			messageID,
			content,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish edit message (async)", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType},
			logger.Field{Key: "message_id", Value: messageID})
		return fmt.Errorf("failed to publish edit message: %w", err)
	}

	return nil
}

// SendDeleteMessageAsync deletes an existing message asynchronously.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendDeleteMessageAsync(userID, channelType, sessionID, messageID string) error {
	correlationID := uuid.New().String()

	event := bus.NewDeleteMessage(
		bus.ChannelType(channelType),
		userID,
		sessionID,
		messageID,
		correlationID,
		nil, // metadata
	)

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish delete message (async)", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType},
			logger.Field{Key: "message_id", Value: messageID})
		return fmt.Errorf("failed to publish delete message: %w", err)
	}

	return nil
}

// SendPhotoMessageAsync sends a photo message asynchronously.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendPhotoMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error {
	correlationID := uuid.New().String()

	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewPhotoMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewPhotoMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish photo message (async)", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return fmt.Errorf("failed to publish photo message: %w", err)
	}

	return nil
}

// SendDocumentMessageAsync sends a document message asynchronously.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendDocumentMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error {
	correlationID := uuid.New().String()

	var event *bus.OutboundMessage
	if keyboard != nil {
		event = bus.NewDocumentMessageWithKeyboard(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			keyboard,
			correlationID,
			nil, // metadata
		)
	} else {
		event = bus.NewDocumentMessage(
			bus.ChannelType(channelType),
			userID,
			sessionID,
			media,
			correlationID,
			nil, // metadata
		)
	}

	if err := a.messageBus.PublishOutbound(*event); err != nil {
		a.logger.ErrorCtx(context.Background(), "failed to publish document message (async)", err,
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "channel_type", Value: channelType})
		return fmt.Errorf("failed to publish document message: %w", err)
	}

	return nil
}

var _ agent.MessageSender = (*AgentMessageSender)(nil) // Compile-time interface check
