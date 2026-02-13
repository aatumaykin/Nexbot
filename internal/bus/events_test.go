package bus

import (
	"strings"
	"testing"
	"time"
)

// TestEvent_NewProcessingStartEvent tests creating a processing start event
func TestEvent_NewProcessingStartEvent(t *testing.T) {
	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)

	if event.Type != EventTypeProcessingStart {
		t.Errorf("Expected event type %s, got %s", EventTypeProcessingStart, event.Type)
	}

	if event.ChannelType != ChannelTypeTelegram {
		t.Errorf("Expected channel type %s, got %s", ChannelTypeTelegram, event.ChannelType)
	}

	if event.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", event.UserID)
	}

	if event.SessionID != "session456" {
		t.Errorf("Expected session ID 'session456', got '%s'", event.SessionID)
	}

	if time.Since(event.Timestamp) > time.Second {
		t.Error("Event timestamp should be recent")
	}
}

// TestEvent_NewProcessingEndEvent tests creating a processing end event
func TestEvent_NewProcessingEndEvent(t *testing.T) {
	event := NewProcessingEndEvent(ChannelTypeTelegram, "user123", "session456", nil)

	if event.Type != EventTypeProcessingEnd {
		t.Errorf("Expected event type %s, got %s", EventTypeProcessingEnd, event.Type)
	}

	if event.ChannelType != ChannelTypeTelegram {
		t.Errorf("Expected channel type %s, got %s", ChannelTypeTelegram, event.ChannelType)
	}

	if event.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", event.UserID)
	}

	if event.SessionID != "session456" {
		t.Errorf("Expected session ID 'session456', got '%s'", event.SessionID)
	}

	if time.Since(event.Timestamp) > time.Second {
		t.Error("Event timestamp should be recent")
	}
}

// TestEvent_Metadata tests event metadata
func TestEvent_Metadata(t *testing.T) {
	metadata := map[string]any{
		"chat_id": 123456,
		"custom":  "value",
	}

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", metadata)

	if event.Metadata == nil {
		t.Fatal("Event metadata should not be nil")
	}

	if event.Metadata["chat_id"] != 123456 {
		t.Errorf("Expected metadata chat_id 123456, got %v", event.Metadata["chat_id"])
	}

	if event.Metadata["custom"] != "value" {
		t.Errorf("Expected metadata custom 'value', got %v", event.Metadata["custom"])
	}
}

// TestEvent_ToJSON tests event JSON serialization
func TestEvent_ToJSON(t *testing.T) {
	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("ToJSON() returned empty data")
	}

	// Basic check that JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"type":"processing_start"`) {
		t.Errorf("JSON should contain type field, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"user_id":"user123"`) {
		t.Errorf("JSON should contain user_id field, got: %s", jsonStr)
	}
}

// TestEvent_FromJSON tests event JSON deserialization
func TestEvent_FromJSON(t *testing.T) {
	original := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	var restored Event
	err = restored.FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if restored.Type != original.Type {
		t.Errorf("Expected type %s, got %s", original.Type, restored.Type)
	}

	if restored.ChannelType != original.ChannelType {
		t.Errorf("Expected channel type %s, got %s", original.ChannelType, restored.ChannelType)
	}

	if restored.UserID != original.UserID {
		t.Errorf("Expected user ID %s, got %s", original.UserID, restored.UserID)
	}

	if restored.SessionID != original.SessionID {
		t.Errorf("Expected session ID %s, got %s", original.SessionID, restored.SessionID)
	}
}

// TestEvent_JSONRoundTrip tests that JSON serialization and deserialization work together
func TestEvent_JSONRoundTrip(t *testing.T) {
	metadata := map[string]any{
		"chat_id": 123456,
		"custom":  "value",
	}

	original := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", metadata)

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	var restored Event
	err = restored.FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if restored.Type != original.Type {
		t.Errorf("Round trip failed: type mismatch")
	}

	if restored.ChannelType != original.ChannelType {
		t.Errorf("Round trip failed: channel type mismatch")
	}

	if restored.UserID != original.UserID {
		t.Errorf("Round trip failed: user ID mismatch")
	}

	if restored.SessionID != original.SessionID {
		t.Errorf("Round trip failed: session ID mismatch")
	}

	if restored.Metadata == nil {
		t.Fatal("Round trip failed: metadata lost")
	}

	// Note: JSON deserialization changes numeric types from int to float64
	// This is standard JSON behavior in Go
	if chatID, ok := restored.Metadata["chat_id"].(float64); ok {
		if int(chatID) != original.Metadata["chat_id"].(int) {
			t.Errorf("Round trip failed: metadata chat_id mismatch, got %v, want %v",
				int(chatID), original.Metadata["chat_id"].(int))
		}
	} else {
		t.Errorf("Round trip failed: metadata chat_id type mismatch, got %T", restored.Metadata["chat_id"])
	}

	if restored.Metadata["custom"] != original.Metadata["custom"] {
		t.Errorf("Round trip failed: metadata custom mismatch")
	}
}

// TestMessageType_Constants tests that MessageType constants are defined correctly
func TestMessageType_Constants(t *testing.T) {
	tests := []struct {
		name        string
		messageType MessageType
		expected    string
	}{
		{"Text message", MessageTypeText, "text"},
		{"Edit message", MessageTypeEdit, "edit"},
		{"Delete message", MessageTypeDelete, "delete"},
		{"Photo message", MessageTypePhoto, "photo"},
		{"Document message", MessageTypeDocument, "document"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.messageType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.messageType)
			}
		})
	}
}

// TestOutboundMessage_NewOutboundMessage tests creating a text message
func TestOutboundMessage_NewOutboundMessage(t *testing.T) {
	msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello world", "corr789", FormatTypePlain, nil)

	if msg.Type != MessageTypeText {
		t.Errorf("Expected type %s, got %s", MessageTypeText, msg.Type)
	}

	if msg.Content != "Hello world" {
		t.Errorf("Expected content 'Hello world', got '%s'", msg.Content)
	}

	if msg.ChannelType != ChannelTypeTelegram {
		t.Errorf("Expected channel type %s, got %s", ChannelTypeTelegram, msg.ChannelType)
	}

	if time.Since(msg.Timestamp) > time.Second {
		t.Error("Message timestamp should be recent")
	}

	if msg.CorrelationID != "corr789" {
		t.Errorf("Expected correlation ID 'corr789', got '%s'", msg.CorrelationID)
	}
}

// TestOutboundMessage_NewEditMessage tests creating an edit message
func TestOutboundMessage_NewEditMessage(t *testing.T) {
	msg := NewEditMessage(ChannelTypeTelegram, "user123", "session456", "msg789", "Updated content", "corr123", FormatTypePlain, nil)

	if msg.Type != MessageTypeEdit {
		t.Errorf("Expected type %s, got %s", MessageTypeEdit, msg.Type)
	}

	if msg.MessageID != "msg789" {
		t.Errorf("Expected message ID 'msg789', got '%s'", msg.MessageID)
	}

	if msg.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", msg.Content)
	}

	if time.Since(msg.Timestamp) > time.Second {
		t.Error("Message timestamp should be recent")
	}
}

// TestOutboundMessage_NewDeleteMessage tests creating a delete message
func TestOutboundMessage_NewDeleteMessage(t *testing.T) {
	msg := NewDeleteMessage(ChannelTypeTelegram, "user123", "session456", "msg456", "corr789", nil)

	if msg.Type != MessageTypeDelete {
		t.Errorf("Expected type %s, got %s", MessageTypeDelete, msg.Type)
	}

	if msg.MessageID != "msg456" {
		t.Errorf("Expected message ID 'msg456', got '%s'", msg.MessageID)
	}

	if msg.Content != "" {
		t.Errorf("Expected empty content for delete message, got '%s'", msg.Content)
	}

	if time.Since(msg.Timestamp) > time.Second {
		t.Error("Message timestamp should be recent")
	}
}

// TestOutboundMessage_NewPhotoMessage tests creating a photo message
func TestOutboundMessage_NewPhotoMessage(t *testing.T) {
	media := &MediaData{
		Type:     "photo",
		URL:      "https://example.com/photo.jpg",
		Caption:  "A beautiful photo",
		FileName: "photo.jpg",
	}

	msg := NewPhotoMessage(ChannelTypeTelegram, "user123", "session456", media, "corr123", FormatTypePlain, nil)

	if msg.Type != MessageTypePhoto {
		t.Errorf("Expected type %s, got %s", MessageTypePhoto, msg.Type)
	}

	if msg.Media == nil {
		t.Fatal("Media should not be nil")
	}

	if msg.Media.URL != "https://example.com/photo.jpg" {
		t.Errorf("Expected media URL 'https://example.com/photo.jpg', got '%s'", msg.Media.URL)
	}

	if msg.Media.Caption != "A beautiful photo" {
		t.Errorf("Expected caption 'A beautiful photo', got '%s'", msg.Media.Caption)
	}

	if time.Since(msg.Timestamp) > time.Second {
		t.Error("Message timestamp should be recent")
	}
}

// TestOutboundMessage_NewDocumentMessage tests creating a document message
func TestOutboundMessage_NewDocumentMessage(t *testing.T) {
	media := &MediaData{
		Type:     "document",
		URL:      "https://example.com/document.pdf",
		FileName: "document.pdf",
		FileID:   "file123",
	}

	msg := NewDocumentMessage(ChannelTypeTelegram, "user123", "session456", media, "corr456", FormatTypePlain, nil)

	if msg.Type != MessageTypeDocument {
		t.Errorf("Expected type %s, got %s", MessageTypeDocument, msg.Type)
	}

	if msg.Media == nil {
		t.Fatal("Media should not be nil")
	}

	if msg.Media.URL != "https://example.com/document.pdf" {
		t.Errorf("Expected media URL 'https://example.com/document.pdf', got '%s'", msg.Media.URL)
	}

	if msg.Media.FileName != "document.pdf" {
		t.Errorf("Expected file name 'document.pdf', got '%s'", msg.Media.FileName)
	}

	if msg.Media.FileID != "file123" {
		t.Errorf("Expected file ID 'file123', got '%s'", msg.Media.FileID)
	}

	if time.Since(msg.Timestamp) > time.Second {
		t.Error("Message timestamp should be recent")
	}
}

// TestMediaData_Fields tests MediaData structure
func TestMediaData_Fields(t *testing.T) {
	media := &MediaData{
		Type:      "photo",
		URL:       "https://example.com/test.jpg",
		FileID:    "file123",
		LocalPath: "/tmp/test.jpg",
		Caption:   "Test caption",
		FileName:  "test.jpg",
	}

	if media.Type != "photo" {
		t.Errorf("Expected type 'photo', got '%s'", media.Type)
	}

	if media.URL != "https://example.com/test.jpg" {
		t.Errorf("Expected URL 'https://example.com/test.jpg', got '%s'", media.URL)
	}

	if media.FileID != "file123" {
		t.Errorf("Expected file ID 'file123', got '%s'", media.FileID)
	}

	if media.LocalPath != "/tmp/test.jpg" {
		t.Errorf("Expected local path '/tmp/test.jpg', got '%s'", media.LocalPath)
	}

	if media.Caption != "Test caption" {
		t.Errorf("Expected caption 'Test caption', got '%s'", media.Caption)
	}

	if media.FileName != "test.jpg" {
		t.Errorf("Expected file name 'test.jpg', got '%s'", media.FileName)
	}
}

// TestOutboundMessage_WithMedia tests OutboundMessage with media field
func TestOutboundMessage_WithMedia(t *testing.T) {
	media := &MediaData{
		Type:     "photo",
		URL:      "https://example.com/photo.jpg",
		Caption:  "Test",
		FileName: "photo.jpg",
	}

	msg := &OutboundMessage{
		ChannelType: ChannelTypeTelegram,
		UserID:      "user123",
		SessionID:   "session456",
		Type:        MessageTypePhoto,
		Media:       media,
		Timestamp:   time.Now(),
	}

	if msg.Media == nil {
		t.Fatal("Media should not be nil")
	}

	if msg.Media.Type != "photo" {
		t.Errorf("Expected media type 'photo', got '%s'", msg.Media.Type)
	}
}

// TestOutboundMessage_WithMessageID tests OutboundMessage with message ID field
func TestOutboundMessage_WithMessageID(t *testing.T) {
	msg := &OutboundMessage{
		ChannelType: ChannelTypeTelegram,
		UserID:      "user123",
		SessionID:   "session456",
		Type:        MessageTypeEdit,
		MessageID:   "msg123",
		Content:     "Updated",
		Timestamp:   time.Now(),
	}

	if msg.MessageID != "msg123" {
		t.Errorf("Expected message ID 'msg123', got '%s'", msg.MessageID)
	}
}
