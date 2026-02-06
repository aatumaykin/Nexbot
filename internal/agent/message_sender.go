package agent

// MessageSender interface for sending messages from tools.
// This abstraction allows tools to send messages without depending
// directly on the message bus implementation.
type MessageSender interface {
	SendMessage(userID, channelType, sessionID, message string) error
}
