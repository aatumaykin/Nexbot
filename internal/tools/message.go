package tools

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// SendMessageTool implements the Tool interface for sending messages through the message bus.
// It allows the LLM to send messages to external channels (e.g., Telegram).
type SendMessageTool struct {
	sender agent.MessageSender
	logger *logger.Logger
}

// SendMessageArgs represents the arguments for the send message tool.
type SendMessageArgs struct {
	UserID      string `json:"user_id"`      // User ID (default: "user")
	ChannelType string `json:"channel_type"` // Channel type (default: "telegram")
	SessionID   string `json:"session_id"`   // Session ID (default: "heartbeat-check")
	Message     string `json:"message"`      // Message to send (required)
}

// NewSendMessageTool creates a new SendMessageTool instance.
func NewSendMessageTool(sender agent.MessageSender, logger *logger.Logger) *SendMessageTool {
	return &SendMessageTool{
		sender: sender,
		logger: logger,
	}
}

// Name returns the tool name.
func (t *SendMessageTool) Name() string {
	return "send_message"
}

// Description returns a description of what the tool does.
func (t *SendMessageTool) Description() string {
	return "Sends a message to an external channel through the message bus. Useful for proactively sending notifications, status updates, or responses to users."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *SendMessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user_id": map[string]interface{}{
				"type":        "string",
				"description": "User ID to send the message to. Defaults to 'user' if not specified.",
				"default":     "user",
			},
			"channel_type": map[string]interface{}{
				"type":        "string",
				"description": "Channel type for the message (e.g., 'telegram'). Defaults to 'telegram' if not specified.",
				"default":     "telegram",
			},
			"session_id": map[string]interface{}{
				"type":        "string",
				"description": "Session ID for the message. Defaults to 'heartbeat-check' if not specified.",
				"default":     "heartbeat-check",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Message content to send. This is a required field.",
			},
		},
		"required": []string{"message"},
	}
}

// Execute executes the send message tool.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *SendMessageTool) Execute(args string) (string, error) {
	// Parse arguments
	var params SendMessageArgs
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("failed to parse send_message arguments: %w", err)
	}

	// Apply defaults
	if params.UserID == "" {
		params.UserID = "user"
	}
	if params.ChannelType == "" {
		params.ChannelType = "telegram"
	}
	if params.SessionID == "" {
		params.SessionID = "heartbeat-check"
	}

	// Validate required field
	if params.Message == "" {
		return "", fmt.Errorf("message parameter is required for send_message action")
	}

	// Send message through the sender interface
	result, err := t.sender.SendMessage(params.UserID, params.ChannelType, params.SessionID, params.Message)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	t.logger.Info("send_message tool executed",
		logger.Field{Key: "user_id", Value: params.UserID},
		logger.Field{Key: "channel_type", Value: params.ChannelType},
		logger.Field{Key: "session_id", Value: params.SessionID},
		logger.Field{Key: "message_length", Value: len(params.Message)})

	if !result.Success {
		var errorMsg string
		if result.Error != nil {
			errorMsg = fmt.Sprintf(`❌ Failed to send message to %s

%s

The message was not delivered. You may need to:
- Fix the message formatting (if it's a parse error)
- Retry after the specified delay (if rate limited)
- Check permissions and bot rights

Original message: %q`,
				params.ChannelType,
				result.Error.ToLLMContext(),
				params.Message)
		} else {
			errorMsg = fmt.Sprintf("❌ Failed to send message to %s (no error details available)", params.ChannelType)
		}
		return "", errors.New(errorMsg)
	}

	return fmt.Sprintf("✅ Message sent successfully\n   Channel: %s\n   User: %s\n   Session: %s\n   Message: %s",
		params.ChannelType, params.UserID, params.SessionID, params.Message), nil
}

// ToSchema returns the OpenAI-compatible schema for this tool.
func (t *SendMessageTool) ToSchema() map[string]interface{} {
	return t.Parameters()
}
