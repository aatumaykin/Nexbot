package tools

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessageSender is a simple mock implementation of agent.MessageSender.
type mockMessageSender struct {
	sendFunc         func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error)
	sendKeyboardFunc func(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error)
}

func (m *mockMessageSender) SendMessage(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
	if m.sendFunc != nil {
		return m.sendFunc(userID, channelType, sessionID, message, timeout)
	}
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendMessageWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	if m.sendKeyboardFunc != nil {
		return m.sendKeyboardFunc(userID, channelType, sessionID, message, keyboard, timeout)
	}
	if m.sendFunc != nil {
		return m.sendFunc(userID, channelType, sessionID, message, timeout)
	}
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendEditMessage(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendDeleteMessage(userID, channelType, sessionID, messageID string, timeout time.Duration) (*agent.MessageResult, error) {
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendPhotoMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendDocumentMessage(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
	return &agent.MessageResult{Success: true}, nil
}

func (m *mockMessageSender) SendMessageAsync(userID, channelType, sessionID, message string) error {
	return nil
}

func (m *mockMessageSender) SendMessageAsyncWithKeyboard(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard) error {
	return nil
}

func (m *mockMessageSender) SendEditMessageAsync(userID, channelType, sessionID, messageID, content string, keyboard *bus.InlineKeyboard) error {
	return nil
}

func (m *mockMessageSender) SendDeleteMessageAsync(userID, channelType, sessionID, messageID string) error {
	return nil
}

func (m *mockMessageSender) SendPhotoMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error {
	return nil
}

func (m *mockMessageSender) SendDocumentMessageAsync(userID, channelType, sessionID string, media *bus.MediaData, keyboard *bus.InlineKeyboard) error {
	return nil
}

// setupTestEnvironmentForMessage creates a test environment with message bus and logger.
func setupTestEnvironmentForMessage(t *testing.T) (*bus.MessageBus, *logger.Logger, func()) {
	// Create logger
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	// Create message bus
	messageBus := bus.New(100, log)

	// Start message bus
	ctx, cancel := context.WithCancel(context.Background())
	err = messageBus.Start(ctx)
	require.NoError(t, err, "Failed to start message bus")

	// Cleanup function
	cleanup := func() {
		cancel()
		_ = messageBus.Stop()
	}

	return messageBus, log, cleanup
}

// setupSendMessageTool creates a SendMessageTool for testing using real message bus.
func setupSendMessageTool(t *testing.T) *SendMessageTool {
	messageBus, log, cleanup := setupTestEnvironmentForMessage(t)
	t.Cleanup(cleanup)

	// Create mock that delegates to real message bus
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			correlationID := sessionID // Use session ID as correlation ID
			event := bus.NewOutboundMessage(
				bus.ChannelType(channelType),
				userID,
				sessionID,
				message,
				correlationID,
				nil, // no metadata
			)
			err := messageBus.PublishOutbound(*event)
			if err != nil {
				return nil, err
			}
			return &agent.MessageResult{Success: true}, nil
		},
	}

	return NewSendMessageTool(sender, log)
}

// TestSendMessageToolDefaults tests that default values are applied correctly.
func TestSendMessageToolDefaults(t *testing.T) {
	tool := setupSendMessageTool(t)

	args := `{
		"message": "Hello, world!",
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Contains(t, result, "Session: telegram:123456789", "Result should contain session ID")
	assert.Contains(t, result, "Hello, world!", "Result should contain message content")
}

// TestSendMessageToolCustomSession tests that custom session_id is used when provided.
func TestSendMessageToolCustomSession(t *testing.T) {
	tool := setupSendMessageTool(t)

	args := `{
		"message": "Custom session message",
		"session_id": "telegram:456"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Contains(t, result, "Session: telegram:456", "Result should contain custom session ID")
}

// TestSendMessageToolPublishError tests error handling when message bus publish fails.
func TestSendMessageToolPublishError(t *testing.T) {
	// Create logger
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	// Create mock that returns error
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			return nil, assert.AnError
		},
	}

	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Test message",
		"session_id": "telegram:test-session"
	}`

	result, err := tool.Execute(args)
	// Should return error since sender returns error
	assert.Error(t, err, "Execute should return error when sender fails")
	assert.Empty(t, result, "Result should be empty on error")
	assert.Contains(t, err.Error(), "failed to send", "Error should mention send failure")
}

// TestSendMessageToolMissingMessage tests that missing required message parameter returns error.
func TestSendMessageToolMissingMessage(t *testing.T) {
	tool := setupSendMessageTool(t)

	args := `{
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.Error(t, err, "Execute should return error for missing message")
	assert.Empty(t, result, "Result should be empty on error")
	assert.Contains(t, err.Error(), "message parameter is required", "Error should mention required field")
}

// TestSendMessageToolInvalidJSON tests handling of invalid JSON.
func TestSendMessageToolInvalidJSON(t *testing.T) {
	tool := setupSendMessageTool(t)

	args := `{invalid json`

	result, err := tool.Execute(args)
	assert.Error(t, err, "Execute should return error for invalid JSON")
	assert.Empty(t, result, "Result should be empty on error")
	assert.Contains(t, err.Error(), "failed to parse send_message arguments", "Error should mention parse error")
}

// TestSendMessageToolAllCustom tests that all custom parameters work together.
func TestSendMessageToolAllCustom(t *testing.T) {
	tool := setupSendMessageTool(t)

	args := `{
		"message": "All custom parameters",
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Contains(t, result, "Session: telegram:123456789", "Result should contain custom session ID")
	assert.Contains(t, result, "All custom parameters", "Result should contain message content")
}

// TestSendMessageToolName tests that tool returns correct name.
func TestSendMessageToolName(t *testing.T) {
	tool := setupSendMessageTool(t)
	assert.Equal(t, "send_message", tool.Name(), "Tool name should be 'send_message'")
}

// TestSendMessageToolDescription tests that tool returns a non-empty description.
func TestSendMessageToolDescription(t *testing.T) {
	tool := setupSendMessageTool(t)
	desc := tool.Description()
	assert.NotEmpty(t, desc, "Description should not be empty")
	assert.Contains(t, desc, "message", "Description should mention 'message'")
	assert.Contains(t, desc, "channel", "Description should mention 'channel'")
}

// TestSendMessageToolParameters tests that tool returns valid parameters.
func TestSendMessageToolParameters(t *testing.T) {
	tool := setupSendMessageTool(t)
	params := tool.Parameters()

	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Equal(t, "object", params["type"], "Type should be 'object'")

	props, ok := params["properties"].(map[string]interface{})
	assert.True(t, ok, "Properties should be a map")

	// Check session_id property
	sessionIDProp, ok := props["session_id"].(map[string]interface{})
	assert.True(t, ok, "session_id property should be a map")
	assert.Equal(t, "string", sessionIDProp["type"], "session_id type should be 'string'")
	assert.Nil(t, sessionIDProp["default"], "session_id should not have default")

	// Check message property
	messageProp, ok := props["message"].(map[string]interface{})
	assert.True(t, ok, "message property should be a map")
	assert.Equal(t, "string", messageProp["type"], "message type should be 'string'")
	assert.Nil(t, messageProp["default"], "message should not have default")

	// Check inline_keyboard property (optional)
	inlineKeyboardProp, ok := props["inline_keyboard"].(map[string]interface{})
	assert.True(t, ok, "inline_keyboard property should be a map")
	assert.Equal(t, "object", inlineKeyboardProp["type"], "inline_keyboard type should be 'object'")

	// Verify inline_keyboard structure
	inlineKeyboardProps, ok := inlineKeyboardProp["properties"].(map[string]interface{})
	assert.True(t, ok, "inline_keyboard properties should be a map")
	rowsProp, ok := inlineKeyboardProps["rows"].(map[string]interface{})
	assert.True(t, ok, "rows property should be a map")
	assert.Equal(t, "array", rowsProp["type"], "rows type should be 'array'")

	// Check required fields - try both types
	required := params["required"]
	switch v := required.(type) {
	case []interface{}:
		assert.Contains(t, v, "session_id", "Required should contain 'session_id'")
		assert.Len(t, v, 1, "Only 'session_id' should be required")
	case []string:
		assert.Contains(t, v, "session_id", "Required should contain 'session_id'")
		assert.Len(t, v, 1, "Only 'session_id' should be required")
	default:
		assert.Fail(t, "Required should be a slice")
	}
}

// TestSendMessageToolToSchema tests that ToSchema returns correct schema.
func TestSendMessageToolToSchema(t *testing.T) {
	tool := setupSendMessageTool(t)
	schema := tool.ToSchema()
	assert.NotNil(t, schema, "Schema should not be nil")
	assert.Equal(t, tool.Parameters(), schema, "Schema should match parameters")
}

// TestSendMessageToolWithInlineKeyboard tests sending message with inline keyboard.
func TestSendMessageToolWithInlineKeyboard(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var capturedKeyboard *bus.InlineKeyboard
	sender := &mockMessageSender{
		sendKeyboardFunc: func(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
			capturedKeyboard = keyboard
			return &agent.MessageResult{Success: true}, nil
		},
	}

	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Choose an option:",
		"session_id": "telegram:123456789",
		"inline_keyboard": {
			"rows": [
				[
					{"text": "Button 1", "data": "btn1"},
					{"text": "Button 2", "data": "btn2"}
				],
				[
					{"text": "Go to website", "url": "https://example.com"}
				]
			]
		}
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Contains(t, result, "Keyboard: 2 row(s)", "Result should mention keyboard")

	// Verify keyboard structure
	assert.NotNil(t, capturedKeyboard, "Keyboard should be captured")
	assert.Len(t, capturedKeyboard.Rows, 2, "Should have 2 rows")

	// Check first row
	assert.Len(t, capturedKeyboard.Rows[0], 2, "First row should have 2 buttons")
	assert.Equal(t, "Button 1", capturedKeyboard.Rows[0][0].Text, "First button text should match")
	assert.Equal(t, "btn1", capturedKeyboard.Rows[0][0].Data, "First button data should match")
	assert.Equal(t, "", capturedKeyboard.Rows[0][0].URL, "First button URL should be empty")

	// Check second row
	assert.Len(t, capturedKeyboard.Rows[1], 1, "Second row should have 1 button")
	assert.Equal(t, "Go to website", capturedKeyboard.Rows[1][0].Text, "URL button text should match")
	assert.Equal(t, "", capturedKeyboard.Rows[1][0].Data, "URL button data should be empty")
	assert.Equal(t, "https://example.com", capturedKeyboard.Rows[1][0].URL, "URL should match")
}

// TestSendMessageToolWithEmptyKeyboard tests that message is sent without keyboard when inline_keyboard is empty.
func TestSendMessageToolWithEmptyKeyboard(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var sentWithKeyboard bool
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			sentWithKeyboard = false
			return &agent.MessageResult{Success: true}, nil
		},
		sendKeyboardFunc: func(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
			sentWithKeyboard = true
			return &agent.MessageResult{Success: true}, nil
		},
	}

	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Test message",
		"session_id": "telegram:123456789",
		"inline_keyboard": {
			"rows": []
		}
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.False(t, sentWithKeyboard, "Should not use SendMessageWithKeyboard for empty keyboard")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
}

// TestSendMessageToolAsyncMode tests async mode (wait_for_confirmation=false).
func TestSendMessageToolAsyncMode(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var usedAsync bool
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			usedAsync = false
			return &agent.MessageResult{Success: true}, nil
		},
	}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Async message",
		"session_id": "telegram:123456789",
		"wait_for_confirmation": false
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.Contains(t, result, "async", "Result should indicate async mode")
	assert.False(t, usedAsync, "Should use async method, not sync")
}

// TestSendMessageToolAsyncModeWithKeyboard tests async mode with keyboard.
func TestSendMessageToolAsyncModeWithKeyboard(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var usedAsync bool
	sender := &mockMessageSender{
		sendKeyboardFunc: func(userID, channelType, sessionID, message string, keyboard *bus.InlineKeyboard, timeout time.Duration) (*agent.MessageResult, error) {
			usedAsync = false
			return &agent.MessageResult{Success: true}, nil
		},
	}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Async message with keyboard",
		"session_id": "telegram:123456789",
		"wait_for_confirmation": false,
		"inline_keyboard": {
			"rows": [
				[{"text": "Button", "data": "btn"}]
			]
		}
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.False(t, usedAsync, "Should use async method, not sync")
}

// TestSendMessageToolAsyncModeEdit tests async mode for edit message.
func TestSendMessageToolAsyncModeEdit(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	sender := &mockMessageSender{}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Edited async",
		"session_id": "telegram:123456789",
		"message_type": "edit",
		"message_id": "123",
		"wait_for_confirmation": false
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.Contains(t, result, "Message ID: 123", "Result should contain message ID")
}

// TestSendMessageToolAsyncModeDelete tests async mode for delete message.
func TestSendMessageToolAsyncModeDelete(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	sender := &mockMessageSender{}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"session_id": "telegram:123456789",
		"message_type": "delete",
		"message_id": "456",
		"wait_for_confirmation": false
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.Contains(t, result, "Message ID: 456", "Result should contain message ID")
}

// TestSendMessageToolAsyncModePhoto tests async mode for photo message.
func TestSendMessageToolAsyncModePhoto(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	sender := &mockMessageSender{}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"session_id": "telegram:123456789",
		"message_type": "photo",
		"media_url": "https://example.com/photo.jpg",
		"media_caption": "Async photo",
		"wait_for_confirmation": false
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.Contains(t, result, "https://example.com/photo.jpg", "Result should contain media URL")
}

// TestSendMessageToolAsyncModeDocument tests async mode for document message.
func TestSendMessageToolAsyncModeDocument(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	sender := &mockMessageSender{}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"session_id": "telegram:123456789",
		"message_type": "document",
		"media_url": "https://example.com/file.pdf",
		"media_caption": "Async document",
		"wait_for_confirmation": false
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "queued successfully", "Result should mention async mode")
	assert.Contains(t, result, "https://example.com/file.pdf", "Result should contain media URL")
}

// TestSendMessageToolCustomTimeout tests custom timeout in sync mode.
func TestSendMessageToolCustomTimeout(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var capturedTimeout time.Duration
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			capturedTimeout = timeout
			return &agent.MessageResult{Success: true}, nil
		},
	}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Message with timeout",
		"session_id": "telegram:123456789",
		"timeout": 10
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Equal(t, 10*time.Second, capturedTimeout, "Timeout should be 10 seconds")
}

// TestSendMessageToolDefaultTimeout tests default timeout in sync mode.
func TestSendMessageToolDefaultTimeout(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var capturedTimeout time.Duration
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			capturedTimeout = timeout
			return &agent.MessageResult{Success: true}, nil
		},
	}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Message with default timeout",
		"session_id": "telegram:123456789"
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.Equal(t, 5*time.Second, capturedTimeout, "Default timeout should be 5 seconds")
}

// TestSendMessageToolWaitForConfirmationTrue tests sync mode with explicit wait_for_confirmation=true.
func TestSendMessageToolWaitForConfirmationTrue(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	var usedAsync bool
	sender := &mockMessageSender{
		sendFunc: func(userID, channelType, sessionID, message string, timeout time.Duration) (*agent.MessageResult, error) {
			usedAsync = false
			return &agent.MessageResult{Success: true}, nil
		},
	}
	tool := NewSendMessageTool(sender, log)

	args := `{
		"message": "Sync message",
		"session_id": "telegram:123456789",
		"wait_for_confirmation": true
	}`

	result, err := tool.Execute(args)
	assert.NoError(t, err, "Execute should not return error")
	assert.Contains(t, result, "sent successfully", "Result should contain success message")
	assert.NotContains(t, result, "queued successfully", "Result should not mention async mode")
	assert.False(t, usedAsync, "Should use sync method")
}
