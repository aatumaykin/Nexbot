package agent

import (
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels"
)

// MessageResult - результат отправки сообщения
type MessageResult struct {
	Success      bool                  // Успешная отправка
	Error        channels.ErrorDetails // Детали ошибки (если есть)
	ResponseText string                // Текст ответа от канала (если есть)
}

// MessageSender interface for sending messages from tools.
// This abstraction allows tools to send messages without depending
// directly on the message bus implementation.
type MessageSender interface {
	SendMessage(userID, channelType, sessionID, message string, timeout time.Duration) (*MessageResult, error)
	SendMessageWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*MessageResult, error)
	SendEditMessage(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*MessageResult, error)
	SendDeleteMessage(userID, channelType, sessionID, messageID string, timeout time.Duration) (*MessageResult, error)
	SendPhotoMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*MessageResult, error)
	SendDocumentMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*MessageResult, error)
	SendMessageAsync(userID, channelType, sessionID, message string) error
	SendMessageAsyncWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard) error
	SendEditMessageAsync(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard) error
	SendDeleteMessageAsync(userID, channelType, sessionID, messageID string) error
	SendPhotoMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error
	SendDocumentMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error
}
