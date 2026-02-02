package bus

import (
	"encoding/json"
	"time"
)

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

// OutboundMessage represents a message to be sent to an external channel
type OutboundMessage struct {
	ChannelType ChannelType    `json:"channel_type"`
	UserID      string         `json:"user_id"`
	SessionID   string         `json:"session_id"`
	Content     string         `json:"content"`
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata,omitempty"`
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
func NewOutboundMessage(channelType ChannelType, userID, sessionID, content string, metadata map[string]any) *OutboundMessage {
	return &OutboundMessage{
		ChannelType: channelType,
		UserID:      userID,
		SessionID:   sessionID,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}
