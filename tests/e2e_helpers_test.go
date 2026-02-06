package tests

import (
	"sync"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/mymmrac/telego"
)

// testConfig creates a test configuration with default values.
func testConfig() *config.Config {
	return &config.Config{
		Tools: config.ToolsConfig{
			File: config.FileToolConfig{
				Enabled:       true,
				WhitelistDirs: []string{},
			},
		},
	}
}

// MockTelegramBot is a mock implementation of telego.Bot for testing
type MockTelegramBot struct {
	sentMessages []MockSentMessage
	mu           sync.Mutex
}

type MockSentMessage struct {
	ChatID    int64
	Text      string
	ParseMode string
}

// NewMockTelegramBot creates a mock Telegram bot
func NewMockTelegramBot() *MockTelegramBot {
	return &MockTelegramBot{
		sentMessages: make([]MockSentMessage, 0),
	}
}

// SendMessage mocks of SendMessage method
func (m *MockTelegramBot) SendMessage(params telego.SendMessageParams) (*telego.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var chatID int64
	if params.ChatID.ID != 0 {
		chatID = params.ChatID.ID
	}

	m.sentMessages = append(m.sentMessages, MockSentMessage{
		ChatID:    chatID,
		Text:      params.Text,
		ParseMode: params.ParseMode,
	})

	return &telego.Message{
		MessageID: len(m.sentMessages),
		Chat:      telego.Chat{ID: chatID},
		Text:      params.Text,
	}, nil
}

// GetSentMessages returns all sent messages
func (m *MockTelegramBot) GetSentMessages() []MockSentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentMessages
}

// ClearSentMessages clears the message history
func (m *MockTelegramBot) ClearSentMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = make([]MockSentMessage, 0)
}
