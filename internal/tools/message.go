package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/bus"
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
	SessionID           string              `json:"session_id"`                      // required
	Message             string              `json:"message,omitempty"`               // optional for edit/delete/media types
	MessageType         string              `json:"message_type,omitempty"`          // text, edit, delete, photo, document
	Format              string              `json:"format,omitempty"`                // plain, markdown, html, markdownv2 (default: plain)
	MessageID           string              `json:"message_id,omitempty"`            // required for edit/delete
	MediaURL            string              `json:"media_url,omitempty"`             // required for photo/document
	MediaCaption        string              `json:"media_caption,omitempty"`         // optional caption for media
	ReplyTo             string              `json:"reply_to,omitempty"`              // message ID to reply to
	InlineKeyboard      *InlineKeyboardArgs `json:"inline_keyboard,omitempty"`       // optional
	WaitForConfirmation *bool               `json:"wait_for_confirmation,omitempty"` // true for sync mode (default), false for async mode
	Timeout             int                 `json:"timeout,omitempty"`               // timeout in seconds for sync mode (default: 5)
}

// InlineKeyboardArgs represents an inline keyboard for the send message tool.
// It mirrors the structure of bus.InlineKeyboard but is defined separately for tool arguments.
type InlineKeyboardArgs struct {
	Rows [][]InlineButtonArgs `json:"rows"` // Array of button rows
}

// InlineButtonArgs represents a single button in an inline keyboard for tool arguments.
type InlineButtonArgs struct {
	Text string `json:"text"`          // Button label
	Data string `json:"data"`          // Callback data for button clicks
	URL  string `json:"url,omitempty"` // URL to open when button is clicked (optional)
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
func (t *SendMessageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID for the message context (e.g., 'telegram:123456789').",
			},
			"message_type": map[string]any{
				"type":        "string",
				"description": "Message type: 'text' (default), 'edit', 'delete', 'photo', 'document'.",
				"enum":        []string{"text", "edit", "delete", "photo", "document"},
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Message content to send. Required for 'text' and 'edit' types.",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Message format: 'plain' (default), 'markdown', 'html', 'markdownv2'.",
				"enum":        []string{"plain", "markdown", "html", "markdownv2"},
			},
			"message_id": map[string]any{
				"type":        "string",
				"description": "ID of the message to edit or delete. Required for 'edit' and 'delete' types.",
			},
			"media_url": map[string]any{
				"type":        "string",
				"description": "URL of the media file. Required for 'photo' and 'document' types.",
			},
			"media_caption": map[string]any{
				"type":        "string",
				"description": "Caption for the media (photo/document).",
			},
			"reply_to": map[string]any{
				"type":        "string",
				"description": "Message ID to reply to.",
			},
			"inline_keyboard": map[string]any{
				"type":        "object",
				"description": "Optional inline keyboard with interactive buttons.",
				"properties": map[string]any{
					"rows": map[string]any{
						"type":        "array",
						"description": "Array of button rows (each row is an array of buttons).",
						"items": map[string]any{
							"type":        "array",
							"description": "Array of buttons in a row.",
							"items": map[string]any{
								"type":        "object",
								"description": "A single button definition.",
								"properties": map[string]any{
									"text": map[string]any{
										"type":        "string",
										"description": "Button label text (required).",
									},
									"data": map[string]any{
										"type":        "string",
										"description": "Callback data sent when button is pressed (for callback buttons).",
									},
									"url": map[string]any{
										"type":        "string",
										"description": "URL to open when button is clicked (for URL buttons).",
									},
								},
								"required": []string{"text"},
							},
						},
					},
				},
				"required": []string{"rows"},
			},
			"wait_for_confirmation": map[string]any{
				"type":        "boolean",
				"description": "Wait for confirmation from channel before returning (default: true). Set to false for async (fire-and-forget) mode.",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds for sync mode (default: 5). Ignored in async mode.",
			},
		},
		"required": []string{"session_id"},
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

	// Default message_type is "text"
	messageType := params.MessageType
	if messageType == "" {
		messageType = "text"
	}

	// Parse format (default is empty = plain)
	format := bus.FormatType(params.Format)

	// Parse session_id to extract channel and user_id
	parts := strings.SplitN(params.SessionID, ":", 2)
	channelType := parts[0]
	userID := parts[1]

	// Convert InlineKeyboardArgs to bus.InlineKeyboard if provided
	var keyboard *bus.InlineKeyboard
	if params.InlineKeyboard != nil && len(params.InlineKeyboard.Rows) > 0 {
		keyboard = &bus.InlineKeyboard{
			Rows: make([][]bus.InlineButton, len(params.InlineKeyboard.Rows)),
		}
		for i, row := range params.InlineKeyboard.Rows {
			keyboard.Rows[i] = make([]bus.InlineButton, len(row))
			for j, btn := range row {
				keyboard.Rows[i][j] = bus.InlineButton{
					Text: btn.Text,
					Data: btn.Data,
					URL:  btn.URL,
				}
			}
		}
	}

	// Execute based on message type
	var result *agent.MessageResult
	var err error
	var actionDesc string

	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}
	waitForConfirmation := true
	if params.WaitForConfirmation != nil {
		waitForConfirmation = *params.WaitForConfirmation
	}
	if waitForConfirmation && timeout == 30*time.Second {
		timeout = 5 * time.Second
	}

	switch messageType {
	case "text":
		if params.Message == "" {
			return "", fmt.Errorf("message parameter is required for text messages")
		}
		if waitForConfirmation {
			if keyboard != nil {
				result, err = t.sender.SendMessageWithKeyboard(userID, channelType, params.SessionID, params.Message, keyboard, format, timeout)
				actionDesc = "text message with keyboard"
			} else {
				result, err = t.sender.SendMessage(userID, channelType, params.SessionID, params.Message, format, timeout)
				actionDesc = "text message"
			}
		} else {
			if keyboard != nil {
				err = t.sender.SendMessageAsyncWithKeyboard(userID, channelType, params.SessionID, params.Message, keyboard, format)
			} else {
				err = t.sender.SendMessageAsync(userID, channelType, params.SessionID, params.Message)
			}
			actionDesc = "text message (async)"
			if err != nil {
				return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
			}
			t.logger.Info("send_message tool executed (async mode)",
				logger.Field{Key: "session_id", Value: params.SessionID},
				logger.Field{Key: "message_type", Value: messageType},
				logger.Field{Key: "action", Value: actionDesc},
				logger.Field{Key: "has_keyboard", Value: keyboard != nil})
			return fmt.Sprintf("✅ %s queued successfully\n   Session: %s\n   Message: %s",
				actionDesc, params.SessionID, params.Message), nil
		}

	case "edit":
		if params.MessageID == "" {
			return "", fmt.Errorf("message_id parameter is required for edit messages")
		}
		if params.Message == "" {
			return "", fmt.Errorf("message parameter is required for edit messages")
		}
		if waitForConfirmation {
			result, err = t.sender.SendEditMessage(userID, channelType, params.SessionID, params.MessageID, params.Message, keyboard, format, timeout)
			actionDesc = "edit message"
		} else {
			err = t.sender.SendEditMessageAsync(userID, channelType, params.SessionID, params.MessageID, params.Message, keyboard, format)
			actionDesc = "edit message (async)"
			if err != nil {
				return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
			}
			t.logger.Info("send_message tool executed (async mode)",
				logger.Field{Key: "session_id", Value: params.SessionID},
				logger.Field{Key: "message_type", Value: messageType},
				logger.Field{Key: "action", Value: actionDesc},
				logger.Field{Key: "message_id", Value: params.MessageID})
			return fmt.Sprintf("✅ %s queued successfully\n   Session: %s\n   Message ID: %s",
				actionDesc, params.SessionID, params.MessageID), nil
		}

	case "delete":
		if params.MessageID == "" {
			return "", fmt.Errorf("message_id parameter is required for delete messages")
		}
		if waitForConfirmation {
			result, err = t.sender.SendDeleteMessage(userID, channelType, params.SessionID, params.MessageID, timeout)
			actionDesc = "delete message"
		} else {
			err = t.sender.SendDeleteMessageAsync(userID, channelType, params.SessionID, params.MessageID)
			actionDesc = "delete message (async)"
			if err != nil {
				return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
			}
			t.logger.Info("send_message tool executed (async mode)",
				logger.Field{Key: "session_id", Value: params.SessionID},
				logger.Field{Key: "message_type", Value: messageType},
				logger.Field{Key: "action", Value: actionDesc},
				logger.Field{Key: "message_id", Value: params.MessageID})
			return fmt.Sprintf("✅ %s queued successfully\n   Session: %s\n   Message ID: %s",
				actionDesc, params.SessionID, params.MessageID), nil
		}

	case "photo":
		if params.MediaURL == "" {
			return "", fmt.Errorf("media_url parameter is required for photo messages")
		}
		media := &bus.MediaData{
			Type:    "photo",
			URL:     params.MediaURL,
			Caption: params.MediaCaption,
		}
		if waitForConfirmation {
			result, err = t.sender.SendPhotoMessage(userID, channelType, params.SessionID, media, keyboard, format, timeout)
			actionDesc = "photo message"
		} else {
			err = t.sender.SendPhotoMessageAsync(userID, channelType, params.SessionID, media, keyboard, format)
			actionDesc = "photo message (async)"
			if err != nil {
				return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
			}
			t.logger.Info("send_message tool executed (async mode)",
				logger.Field{Key: "session_id", Value: params.SessionID},
				logger.Field{Key: "message_type", Value: messageType},
				logger.Field{Key: "action", Value: actionDesc},
				logger.Field{Key: "media_url", Value: params.MediaURL})
			return fmt.Sprintf("✅ %s queued successfully\n   Session: %s\n   Media URL: %s",
				actionDesc, params.SessionID, params.MediaURL), nil
		}

	case "document":
		if params.MediaURL == "" {
			return "", fmt.Errorf("media_url parameter is required for document messages")
		}
		media := &bus.MediaData{
			Type:    "document",
			URL:     params.MediaURL,
			Caption: params.MediaCaption,
		}
		if waitForConfirmation {
			result, err = t.sender.SendDocumentMessage(userID, channelType, params.SessionID, media, keyboard, format, timeout)
			actionDesc = "document message"
		} else {
			err = t.sender.SendDocumentMessageAsync(userID, channelType, params.SessionID, media, keyboard, format)
			actionDesc = "document message (async)"
			if err != nil {
				return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
			}
			t.logger.Info("send_message tool executed (async mode)",
				logger.Field{Key: "session_id", Value: params.SessionID},
				logger.Field{Key: "message_type", Value: messageType},
				logger.Field{Key: "action", Value: actionDesc},
				logger.Field{Key: "media_url", Value: params.MediaURL})
			return fmt.Sprintf("✅ %s queued successfully\n   Session: %s\n   Media URL: %s",
				actionDesc, params.SessionID, params.MediaURL), nil
		}

	default:
		return "", fmt.Errorf("unknown message_type: %s (valid types: text, edit, delete, photo, document)", messageType)
	}

	if err != nil {
		return "", fmt.Errorf("failed to send %s: %w", actionDesc, err)
	}

	t.logger.Info("send_message tool executed",
		logger.Field{Key: "session_id", Value: params.SessionID},
		logger.Field{Key: "message_type", Value: messageType},
		logger.Field{Key: "action", Value: actionDesc},
		logger.Field{Key: "has_keyboard", Value: keyboard != nil})

	if !result.Success {
		var errorMsg string
		if result.Error != nil {
			errorMsg = fmt.Sprintf(`❌ Failed to send %s

%s

The message was not delivered. You may need to:
- Fix the message formatting (if it's a parse error)
- Retry after the specified delay (if rate limited)
- Check permissions and bot rights`,
				actionDesc,
				result.Error.ToLLMContext())
		} else {
			errorMsg = fmt.Sprintf("❌ Failed to send %s (no error details available)", actionDesc)
		}
		return "", errors.New(errorMsg)
	}

	var details string
	switch messageType {
	case "text", "edit":
		details = fmt.Sprintf("   Message: %s", params.Message)
	case "photo", "document":
		details = fmt.Sprintf("   Media URL: %s\n   Caption: %s", params.MediaURL, params.MediaCaption)
	case "delete":
		details = fmt.Sprintf("   Deleted message ID: %s", params.MessageID)
	}

	keyboardInfo := ""
	if keyboard != nil {
		keyboardInfo = fmt.Sprintf("\n   Keyboard: %d row(s)", len(keyboard.Rows))
	}
	return fmt.Sprintf("✅ %s sent successfully\n   Session: %s\n%s%s",
		actionDesc, params.SessionID, details, keyboardInfo), nil
}

// ToSchema returns the OpenAI-compatible schema for this tool.
func (t *SendMessageTool) ToSchema() map[string]any {
	return t.Parameters()
}
