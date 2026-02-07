package agent

import "github.com/aatumaykin/nexbot/internal/channels"

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
	SendMessage(userID, channelType, sessionID, message string) (*MessageResult, error)
}
