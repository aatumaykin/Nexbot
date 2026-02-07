package bus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMessage_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		message *InboundMessage
		wantErr bool
	}{
		{
			name: "valid message with metadata",
			message: &InboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "Hello, world!",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"key": "value", "count": 42},
			},
			wantErr: false,
		},
		{
			name: "valid message without metadata",
			message: &InboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Test message",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
			wantErr: false,
		},
		{
			name: "message with empty content",
			message: &InboundMessage{
				ChannelType: ChannelTypeSlack,
				UserID:      "user111",
				SessionID:   "session222",
				Content:     "",
				Timestamp:   time.Unix(1234567892, 0),
				Metadata:    map[string]any{},
			},
			wantErr: false,
		},
		{
			name: "message with special characters",
			message: &InboundMessage{
				ChannelType: ChannelTypeWeb,
				UserID:      "user333",
				SessionID:   "session444",
				Content:     "Special chars: \n\t\r\"'\\",
				Timestamp:   time.Unix(1234567893, 0),
				Metadata:    map[string]any{"emoji": "😀", "unicode": "привет"},
			},
			wantErr: false,
		},
		{
			name: "message with API channel",
			message: &InboundMessage{
				ChannelType: ChannelTypeAPI,
				UserID:      "user555",
				SessionID:   "session666",
				Content:     "API request",
				Timestamp:   time.Unix(1234567894, 0),
				Metadata:    nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.message.ToJSON()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got)

			// Verify it's valid JSON
			var v map[string]any
			err = json.Unmarshal(got, &v)
			require.NoError(t, err)

			// Check required fields are present
			assert.Contains(t, v, "channel_type")
			assert.Contains(t, v, "user_id")
			assert.Contains(t, v, "session_id")
			assert.Contains(t, v, "content")
			assert.Contains(t, v, "timestamp")
		})
	}
}

func TestInboundMessage_FromJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *InboundMessage
		wantErr bool
	}{
		{
			name: "valid JSON with metadata",
			data: []byte(`{
				"channel_type":"telegram",
				"user_id":"user123",
				"session_id":"session456",
				"content":"Hello, world!",
				"timestamp":"2009-02-13T23:31:30Z",
				"metadata":{"key":"value","count":42}
			}`),
			want: &InboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "Hello, world!",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"key": "value", "count": 42.0},
			},
			wantErr: false,
		},
		{
			name: "valid JSON without metadata",
			data: []byte(`{
				"channel_type":"discord",
				"user_id":"user789",
				"session_id":"session012",
				"content":"Test message",
				"timestamp":"2009-02-13T23:31:31Z"
			}`),
			want: &InboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Test message",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
			wantErr: false,
		},
		{
			name: "JSON with empty metadata",
			data: []byte(`{
				"channel_type":"slack",
				"user_id":"user111",
				"session_id":"session222",
				"content":"",
				"timestamp":"2009-02-13T23:31:32Z",
				"metadata":{}
			}`),
			want: &InboundMessage{
				ChannelType: ChannelTypeSlack,
				UserID:      "user111",
				SessionID:   "session222",
				Content:     "",
				Timestamp:   time.Unix(1234567892, 0),
				Metadata:    map[string]any{},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
		{
			name:    "empty JSON",
			data:    []byte(`{}`),
			wantErr: false, // Empty object is valid, fields will be zero values
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
		{
			name: "null data",
			data: []byte(`null`),
			want: &InboundMessage{
				ChannelType: "",
				UserID:      "",
				SessionID:   "",
				Content:     "",
				Timestamp:   time.Time{},
				Metadata:    nil,
			},
			wantErr: false, // JSON null unmarshals to zero values, no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &InboundMessage{}
			err := msg.FromJSON(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// If we have expected values, verify them
			if tt.want != nil && tt.want.ChannelType != "" {
				assert.Equal(t, tt.want.ChannelType, msg.ChannelType)
				assert.Equal(t, tt.want.UserID, msg.UserID)
				assert.Equal(t, tt.want.SessionID, msg.SessionID)
				assert.Equal(t, tt.want.Content, msg.Content)
				// Use Equal for time to handle nanosecond differences
				assert.True(t, tt.want.Timestamp.Equal(msg.Timestamp) ||
					tt.want.Timestamp.Unix() == msg.Timestamp.Unix())
			}
		})
	}
}

func TestOutboundMessage_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		message *OutboundMessage
		wantErr bool
	}{
		{
			name: "valid message with metadata",
			message: &OutboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "Response message",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"status": "success"},
			},
			wantErr: false,
		},
		{
			name: "valid message without metadata",
			message: &OutboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Hello",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
			wantErr: false,
		},
		{
			name: "message with multiline content",
			message: &OutboundMessage{
				ChannelType: ChannelTypeSlack,
				UserID:      "user111",
				SessionID:   "session222",
				Content:     "Line 1\nLine 2\nLine 3",
				Timestamp:   time.Unix(1234567892, 0),
				Metadata:    map[string]any{},
			},
			wantErr: false,
		},
		{
			name: "message with unicode",
			message: &OutboundMessage{
				ChannelType: ChannelTypeWeb,
				UserID:      "user333",
				SessionID:   "session444",
				Content:     "Привет мир! 你好世界! 🌍",
				Timestamp:   time.Unix(1234567893, 0),
				Metadata:    map[string]any{"lang": "ru"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.message.ToJSON()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got)

			// Verify it's valid JSON
			var v map[string]any
			err = json.Unmarshal(got, &v)
			require.NoError(t, err)

			// Check required fields are present
			assert.Contains(t, v, "channel_type")
			assert.Contains(t, v, "user_id")
			assert.Contains(t, v, "session_id")
			assert.Contains(t, v, "content")
			assert.Contains(t, v, "timestamp")
		})
	}
}

func TestOutboundMessage_FromJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *OutboundMessage
		wantErr bool
	}{
		{
			name: "valid JSON with metadata",
			data: []byte(`{
				"channel_type":"telegram",
				"user_id":"user123",
				"session_id":"session456",
				"content":"Response message",
				"timestamp":"2009-02-13T23:31:30Z",
				"metadata":{"status":"success"}
			}`),
			want: &OutboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "Response message",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"status": "success"},
			},
			wantErr: false,
		},
		{
			name: "valid JSON without metadata",
			data: []byte(`{
				"channel_type":"discord",
				"user_id":"user789",
				"session_id":"session012",
				"content":"Hello",
				"timestamp":"2009-02-13T23:31:31Z"
			}`),
			want: &OutboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Hello",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
			wantErr: false,
		},
		{
			name: "JSON with nested metadata",
			data: []byte(`{
				"channel_type":"api",
				"user_id":"user555",
				"session_id":"session666",
				"content":"API response",
				"timestamp":"2009-02-13T23:31:34Z",
				"metadata":{"nested":{"key":"value"},"array":[1,2,3]}
			}`),
			want: &OutboundMessage{
				ChannelType: ChannelTypeAPI,
				UserID:      "user555",
				SessionID:   "session666",
				Content:     "API response",
				Timestamp:   time.Unix(1234567894, 0),
				Metadata:    map[string]any{"nested": map[string]any{"key": "value"}, "array": []any{1.0, 2.0, 3.0}},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
		{
			name: "null data",
			data: []byte(`null`),
			want: &OutboundMessage{
				ChannelType: "",
				UserID:      "",
				SessionID:   "",
				Content:     "",
				Timestamp:   time.Time{},
				Metadata:    nil,
			},
			wantErr: false, // JSON null unmarshals to zero values, no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &OutboundMessage{}
			err := msg.FromJSON(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// If we have expected values, verify them
			if tt.want != nil && tt.want.ChannelType != "" {
				assert.Equal(t, tt.want.ChannelType, msg.ChannelType)
				assert.Equal(t, tt.want.UserID, msg.UserID)
				assert.Equal(t, tt.want.SessionID, msg.SessionID)
				assert.Equal(t, tt.want.Content, msg.Content)
				assert.True(t, tt.want.Timestamp.Equal(msg.Timestamp) ||
					tt.want.Timestamp.Unix() == msg.Timestamp.Unix())
			}
		})
	}
}

func TestInboundMessage_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name    string
		message *InboundMessage
	}{
		{
			name: "full message with metadata",
			message: &InboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "Test content",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"key": "value", "count": float64(42), "nested": map[string]any{"inner": "data"}},
			},
		},
		{
			name: "message without metadata",
			message: &InboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Simple message",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
		},
		{
			name: "message with empty metadata",
			message: &InboundMessage{
				ChannelType: ChannelTypeSlack,
				UserID:      "user111",
				SessionID:   "session222",
				Content:     "Message with empty metadata",
				Timestamp:   time.Unix(1234567892, 0),
				Metadata:    map[string]any{},
			},
		},
		{
			name: "message with special characters",
			message: &InboundMessage{
				ChannelType: ChannelTypeWeb,
				UserID:      "user333",
				SessionID:   "session444",
				Content:     "Special: \n\t\"'\\ 😀 привет",
				Timestamp:   time.Unix(1234567893, 0),
				Metadata:    map[string]any{"emoji": "😎", "text": "привет мир"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := tt.message.ToJSON()
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Deserialize
			result := &InboundMessage{}
			err = result.FromJSON(data)
			require.NoError(t, err)

			// Verify equality
			assert.Equal(t, tt.message.ChannelType, result.ChannelType)
			assert.Equal(t, tt.message.UserID, result.UserID)
			assert.Equal(t, tt.message.SessionID, result.SessionID)
			assert.Equal(t, tt.message.Content, result.Content)
			assert.True(t, tt.message.Timestamp.Equal(result.Timestamp))

			// Check metadata
			if tt.message.Metadata == nil {
				// After JSON roundtrip, nil metadata stays nil
				assert.Nil(t, result.Metadata)
			} else if len(tt.message.Metadata) == 0 {
				// Empty map may become nil after JSON roundtrip (standard Go JSON behavior)
				// Both nil and empty map are acceptable
				assert.Equal(t, 0, len(result.Metadata))
			} else {
				require.NotNil(t, result.Metadata)
				assert.Equal(t, len(tt.message.Metadata), len(result.Metadata))
				for k, v := range tt.message.Metadata {
					assert.Contains(t, result.Metadata, k)
					// JSON unmarshaling converts int to float64, handle this
					if intVal, ok := v.(int); ok {
						if floatVal, ok := result.Metadata[k].(float64); ok {
							assert.Equal(t, float64(intVal), floatVal)
						} else {
							assert.Equal(t, v, result.Metadata[k])
						}
					} else {
						assert.Equal(t, v, result.Metadata[k])
					}
				}
			}
		})
	}
}

func TestOutboundMessage_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name    string
		message *OutboundMessage
	}{
		{
			name: "full message with metadata",
			message: &OutboundMessage{
				ChannelType: ChannelTypeAPI,
				UserID:      "user123",
				SessionID:   "session456",
				Content:     "API response",
				Timestamp:   time.Unix(1234567890, 0),
				Metadata:    map[string]any{"status": float64(200), "data": map[string]any{"id": float64(123)}},
			},
		},
		{
			name: "message without metadata",
			message: &OutboundMessage{
				ChannelType: ChannelTypeTelegram,
				UserID:      "user789",
				SessionID:   "session012",
				Content:     "Telegram response",
				Timestamp:   time.Unix(1234567891, 0),
				Metadata:    nil,
			},
		},
		{
			name: "message with array metadata",
			message: &OutboundMessage{
				ChannelType: ChannelTypeDiscord,
				UserID:      "user111",
				SessionID:   "session222",
				Content:     "Array test",
				Timestamp:   time.Unix(1234567892, 0),
				Metadata:    map[string]any{"items": []interface{}{"a", "b", "c"}, "count": float64(3)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := tt.message.ToJSON()
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Deserialize
			result := &OutboundMessage{}
			err = result.FromJSON(data)
			require.NoError(t, err)

			// Verify equality
			assert.Equal(t, tt.message.ChannelType, result.ChannelType)
			assert.Equal(t, tt.message.UserID, result.UserID)
			assert.Equal(t, tt.message.SessionID, result.SessionID)
			assert.Equal(t, tt.message.Content, result.Content)
			assert.True(t, tt.message.Timestamp.Equal(result.Timestamp))

			// Check metadata
			if tt.message.Metadata == nil {
				// After JSON roundtrip, nil metadata stays nil
				assert.Nil(t, result.Metadata)
			} else if len(tt.message.Metadata) == 0 {
				// Empty map may become nil after JSON roundtrip (standard Go JSON behavior)
				// Both nil and empty map are acceptable
				assert.Equal(t, 0, len(result.Metadata))
			} else {
				require.NotNil(t, result.Metadata)
				assert.Equal(t, len(tt.message.Metadata), len(result.Metadata))
				for k, v := range tt.message.Metadata {
					assert.Contains(t, result.Metadata, k)
					// JSON unmarshaling converts int to float64, handle this
					if intVal, ok := v.(int); ok {
						if floatVal, ok := result.Metadata[k].(float64); ok {
							assert.Equal(t, float64(intVal), floatVal)
						} else {
							assert.Equal(t, v, result.Metadata[k])
						}
					} else {
						assert.Equal(t, v, result.Metadata[k])
					}
				}
			}
		})
	}
}
