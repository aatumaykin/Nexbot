// Package bus provides event structures for the message bus system.
// It defines the core message types used for communication between
// components in the Nexbot system, including inbound messages from
// external channels and outbound messages to be sent to external channels.
//
// Supported channel types include:
//   - Telegram
//   - Discord
//   - Slack
//   - Web
//   - API
//
// All message types support JSON serialization for easy transport and storage.
package bus

import (
	"encoding/json"
	"time"

	"github.com/aatumaykin/nexbot/internal/channels"
)

// EventType represents the type of lifecycle event
type EventType string

const (
	EventTypeProcessingStart EventType = "processing_start" // Event when LLM processing starts
	EventTypeProcessingEnd   EventType = "processing_end"   // Event when LLM processing ends
)

// MessageType represents the type of outbound message
type MessageType string

const (
	MessageTypeText     MessageType = "text"     // Plain text message
	MessageTypeEdit     MessageType = "edit"     // Edit existing message
	MessageTypeDelete   MessageType = "delete"   // Delete existing message
	MessageTypePhoto    MessageType = "photo"    // Photo message
	MessageTypeDocument MessageType = "document" // Document message
)

// FormatType represents the format type for message content
type FormatType string

const (
	FormatTypePlain      FormatType = ""           // Plain text (default)
	FormatTypeMarkdown   FormatType = "markdown"   // Markdown formatting
	FormatTypeHTML       FormatType = "html"       // HTML formatting
	FormatTypeMarkdownV2 FormatType = "markdownv2" // Telegram MarkdownV2 formatting
)

// Event represents a lifecycle event for message processing
type Event struct {
	Type        EventType      `json:"type"`
	ChannelType ChannelType    `json:"channel_type"`
	UserID      string         `json:"user_id"`
	SessionID   string         `json:"session_id"`
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ChannelType represents the type of communication channel
type ChannelType string

const (
	ChannelTypeTelegram ChannelType = "telegram"
	ChannelTypeDiscord  ChannelType = "discord"
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeWeb      ChannelType = "web"
	ChannelTypeAPI      ChannelType = "api"
)

// InboundMessage represents a message received from an external channel
type InboundMessage struct {
	ChannelType ChannelType    `json:"channel_type"`
	UserID      string         `json:"user_id"`
	SessionID   string         `json:"session_id"`
	Content     string         `json:"content"`
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MediaData represents media attachments in outbound messages
type MediaData struct {
	Type      string `json:"type"`       // Media type (e.g., "photo", "document")
	URL       string `json:"url"`        // Direct URL to media (for web)
	FileID    string `json:"file_id"`    // Platform-specific file ID (for telegram, etc.)
	LocalPath string `json:"local_path"` // Path to local file
	Caption   string `json:"caption"`    // Media caption/description
	FileName  string `json:"file_name"`  // Original file name
}

// InlineButton represents a single button in an inline keyboard
type InlineButton struct {
	Text string `json:"text"`          // Button label
	Data string `json:"data"`          // Callback data for button clicks
	URL  string `json:"url,omitempty"` // URL to open when button is clicked (optional)
}

// InlineKeyboard represents an inline keyboard layout
type InlineKeyboard struct {
	Rows [][]InlineButton `json:"rows"` // Array of button rows
}

// OutboundMessage represents a message to be sent to an external channel
type OutboundMessage struct {
	ChannelType    ChannelType     `json:"channel_type"`
	UserID         string          `json:"user_id"`
	SessionID      string          `json:"session_id"`
	Type           MessageType     `json:"type"`                      // Message type (text, edit, delete, photo, document)
	Content        string          `json:"content"`                   // Text content (for text/edit messages)
	Format         FormatType      `json:"format,omitempty"`          // Format type (plain, markdown, html, markdownv2)
	CorrelationID  string          `json:"correlation_id,omitempty"`  // для отслеживания результата отправки
	MessageID      string          `json:"message_id,omitempty"`      // ID of message to edit/delete
	Media          *MediaData      `json:"media,omitempty"`           // Media data (for photo/document messages)
	InlineKeyboard *InlineKeyboard `json:"inline_keyboard,omitempty"` // Inline keyboard for interactive buttons
	Timestamp      time.Time       `json:"timestamp"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

// MessageSendResult - результат отправки сообщения в канал
type MessageSendResult struct {
	CorrelationID string                // ID для сопоставления с запросом
	ChannelType   ChannelType           // Канал отправки (telegram и т.д.)
	Success       bool                  // Успешная отправка
	Error         channels.ErrorDetails // Детали ошибки (если есть)
	Timestamp     time.Time             // Время получения результата
}

// ToJSON serializes the InboundMessage to JSON bytes
func (m *InboundMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes the InboundMessage from JSON bytes
func (m *InboundMessage) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// ToJSON serializes the OutboundMessage to JSON bytes
func (m *OutboundMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes the OutboundMessage from JSON bytes
func (m *OutboundMessage) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// NewInboundMessage creates a new InboundMessage with the current timestamp
func NewInboundMessage(channelType ChannelType, userID, sessionID, content string, metadata map[string]any) *InboundMessage {
	return &InboundMessage{
		ChannelType: channelType,
		UserID:      userID,
		SessionID:   sessionID,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}

// NewOutboundMessage creates a new OutboundMessage with the current timestamp
func NewOutboundMessage(channelType ChannelType, userID, sessionID, content string, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:   channelType,
		UserID:        userID,
		SessionID:     sessionID,
		Type:          MessageTypeText,
		Content:       content,
		Format:        format,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewOutboundMessageWithKeyboard creates a new OutboundMessage with inline keyboard
func NewOutboundMessageWithKeyboard(channelType ChannelType, userID, sessionID, content string, correlationID string, keyboard *InlineKeyboard, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:    channelType,
		UserID:         userID,
		SessionID:      sessionID,
		Type:           MessageTypeText,
		Content:        content,
		Format:         format,
		CorrelationID:  correlationID,
		InlineKeyboard: keyboard,
		Timestamp:      time.Now(),
		Metadata:       metadata,
	}
}

// NewEditMessage creates a new edit message with the current timestamp
func NewEditMessage(channelType ChannelType, userID, sessionID, messageID, content string, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:   channelType,
		UserID:        userID,
		SessionID:     sessionID,
		Type:          MessageTypeEdit,
		Content:       content,
		Format:        format,
		CorrelationID: correlationID,
		MessageID:     messageID,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewEditMessageWithKeyboard creates a new edit message with inline keyboard
func NewEditMessageWithKeyboard(channelType ChannelType, userID, sessionID, messageID, content string, keyboard *InlineKeyboard, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:    channelType,
		UserID:         userID,
		SessionID:      sessionID,
		Type:           MessageTypeEdit,
		Content:        content,
		Format:         format,
		CorrelationID:  correlationID,
		MessageID:      messageID,
		InlineKeyboard: keyboard,
		Timestamp:      time.Now(),
		Metadata:       metadata,
	}
}

// NewDeleteMessage creates a new delete message with the current timestamp
func NewDeleteMessage(channelType ChannelType, userID, sessionID, messageID string, correlationID string, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:   channelType,
		UserID:        userID,
		SessionID:     sessionID,
		Type:          MessageTypeDelete,
		CorrelationID: correlationID,
		MessageID:     messageID,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewPhotoMessage creates a new photo message with the current timestamp
func NewPhotoMessage(channelType ChannelType, userID, sessionID string, media *MediaData, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:   channelType,
		UserID:        userID,
		SessionID:     sessionID,
		Type:          MessageTypePhoto,
		CorrelationID: correlationID,
		Media:         media,
		Format:        format,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewPhotoMessageWithKeyboard creates a new photo message with inline keyboard
func NewPhotoMessageWithKeyboard(channelType ChannelType, userID, sessionID string, media *MediaData, keyboard *InlineKeyboard, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:    channelType,
		UserID:         userID,
		SessionID:      sessionID,
		Type:           MessageTypePhoto,
		CorrelationID:  correlationID,
		Media:          media,
		Format:         format,
		InlineKeyboard: keyboard,
		Timestamp:      time.Now(),
		Metadata:       metadata,
	}
}

// NewDocumentMessage creates a new document message with the current timestamp
func NewDocumentMessage(channelType ChannelType, userID, sessionID string, media *MediaData, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:   channelType,
		UserID:        userID,
		SessionID:     sessionID,
		Type:          MessageTypeDocument,
		CorrelationID: correlationID,
		Media:         media,
		Format:        format,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewDocumentMessageWithKeyboard creates a new document message with inline keyboard
func NewDocumentMessageWithKeyboard(channelType ChannelType, userID, sessionID string, media *MediaData, keyboard *InlineKeyboard, correlationID string, format FormatType, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType:    channelType,
		UserID:         userID,
		SessionID:      sessionID,
		Type:           MessageTypeDocument,
		CorrelationID:  correlationID,
		Media:          media,
		Format:         format,
		InlineKeyboard: keyboard,
		Timestamp:      time.Now(),
		Metadata:       metadata,
	}
}

// ToJSON serializes the Event to JSON bytes
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes the Event from JSON bytes
func (e *Event) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}

// NewProcessingStartEvent creates a new processing start event
func NewProcessingStartEvent(channelType ChannelType, userID, sessionID string, metadata map[string]any) *Event {
	return &Event{
		Type:        EventTypeProcessingStart,
		ChannelType: channelType,
		UserID:      userID,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}

// NewProcessingEndEvent creates a new processing end event
func NewProcessingEndEvent(channelType ChannelType, userID, sessionID string, metadata map[string]any) *Event {
	return &Event{
		Type:        EventTypeProcessingEnd,
		ChannelType: channelType,
		UserID:      userID,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}

// Metrics holds message bus metrics
type Metrics struct {
	InboundMessagesDropped   int64
	OutboundMessagesDropped  int64
	EventsDropped            int64
	ResultsDropped           int64
	InboundSubscribersCount  int
	OutboundSubscribersCount int
	EventSubscribersCount    int
	ResultSubscribersCount   int
}

// GetDroppedMetrics returns a map of dropped message counts
func (m *Metrics) GetDroppedMetrics() map[string]int64 {
	return map[string]int64{
		"inbound_messages_dropped":  m.InboundMessagesDropped,
		"outbound_messages_dropped": m.OutboundMessagesDropped,
		"events_dropped":            m.EventsDropped,
		"results_dropped":           m.ResultsDropped,
	}
}

// Reset resets all metrics counters
func (m *Metrics) Reset() {
	m.InboundMessagesDropped = 0
	m.OutboundMessagesDropped = 0
	m.EventsDropped = 0
	m.ResultsDropped = 0
}
