package tools

import (
	"context"
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessageSender is a simple mock implementation of agent.MessageSender.
type mockMessageSender struct {
	sendFunc func(userID, channelType, sessionID, message string) (*agent.MessageResult, error)
}

func (m *mockMessageSender) SendMessage(userID, channelType, sessionID, message string) (*agent.MessageResult, error) {
	if m.sendFunc != nil {
		return m.sendFunc(userID, channelType, sessionID, message)
	}
	return &agent.MessageResult{Success: true}, nil
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
		sendFunc: func(userID, channelType, sessionID, message string) (*agent.MessageResult, error) {
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
	assert.Contains(t, result, "Message sent successfully", "Result should contain success message")
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
	assert.Contains(t, result, "Message sent successfully", "Result should contain success message")
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
		sendFunc: func(userID, channelType, sessionID, message string) (*agent.MessageResult, error) {
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
	assert.Contains(t, err.Error(), "failed to send message", "Error should mention send failure")
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
	assert.Contains(t, result, "Message sent successfully", "Result should contain success message")
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

	// Check required fields - try both types
	required := params["required"]
	switch v := required.(type) {
	case []interface{}:
		assert.Contains(t, v, "session_id", "Required should contain 'session_id'")
		assert.Contains(t, v, "message", "Required should contain 'message'")
		assert.Len(t, v, 2, "Both 'session_id' and 'message' should be required")
	case []string:
		assert.Contains(t, v, "session_id", "Required should contain 'session_id'")
		assert.Contains(t, v, "message", "Required should contain 'message'")
		assert.Len(t, v, 2, "Both 'session_id' and 'message' should be required")
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
