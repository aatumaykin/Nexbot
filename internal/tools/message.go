package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	SessionID string `json:"session_id"` // required
	Message   string `json:"message"`    // required
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
			"session_id": map[string]interface{}{
				"type":        "string",
				"description": "Session ID for the message context.",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Message content to send. This is a required field.",
			},
		},
		"required": []string{"session_id", "message"},
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

	// Validate required fields
	if params.SessionID == "" {
		return "", fmt.Errorf("session_id parameter is required for send_message action")
	}
	// Валидация session_id формата
	if !strings.Contains(params.SessionID, ":") {
		return "", errors.New("session_id must be in format 'channel:chat_id' (e.g., 'telegram:123456789')")
	}
	if params.Message == "" {
		return "", fmt.Errorf("message parameter is required for send_message action")
	}

	// Send message through the sender interface
	result, err := t.sender.SendMessage("", "", params.SessionID, params.Message)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	t.logger.Info("send_message tool executed",
		logger.Field{Key: "session_id", Value: params.SessionID},
		logger.Field{Key: "message_length", Value: len(params.Message)})

	if !result.Success {
		var errorMsg string
		if result.Error != nil {
			errorMsg = fmt.Sprintf(`❌ Failed to send message

%s

The message was not delivered. You may need to:
- Fix the message formatting (if it's a parse error)
- Retry after the specified delay (if rate limited)
- Check permissions and bot rights

Original message: %q`,
				result.Error.ToLLMContext(),
				params.Message)
		} else {
			errorMsg = "❌ Failed to send message (no error details available)"
		}
		return "", errors.New(errorMsg)
	}

	return fmt.Sprintf("✅ Message sent successfully\n   Session: %s\n   Message: %s",
		params.SessionID, params.Message), nil
}

// ToSchema returns the OpenAI-compatible schema for this tool.
func (t *SendMessageTool) ToSchema() map[string]interface{} {
	return t.Parameters()
}
