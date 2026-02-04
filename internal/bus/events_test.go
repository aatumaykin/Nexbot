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
