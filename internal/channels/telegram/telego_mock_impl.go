package telegram

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/mock"
)

// MockBot is a mock implementation of BotInterface for testing.
// It uses testify/mock to record and verify method calls.
type MockBot struct {
	mock.Mock
}

// GetMe returns basic information about the bot.
func (m *MockBot) GetMe(ctx context.Context) (*telego.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*telego.User), args.Error(1)
}

// SendMessage sends a text message to a chat.
func (m *MockBot) SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*telego.Message), args.Error(1)
}

// SetMyCommands sets the bot's command list in the bot menu.
func (m *MockBot) SetMyCommands(ctx context.Context, params *telego.SetMyCommandsParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// UpdatesViaLongPolling starts long polling for Telegram updates.
// Returns a channel that will receive updates as they arrive.
func (m *MockBot) UpdatesViaLongPolling(ctx context.Context, params *telego.GetUpdatesParams, opts ...telego.LongPollingOption) (<-chan telego.Update, error) {
	args := m.Called(ctx, params, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Convert chan to <-chan for the return type
	return args.Get(0).(chan telego.Update), args.Error(1)
}

// SendChatAction sends a chat action (e.g., typing) to a chat.
func (m *MockBot) SendChatAction(ctx context.Context, params *telego.SendChatActionParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// EditMessageText edits text of a message sent via the bot.
func (m *MockBot) EditMessageText(ctx context.Context, params *telego.EditMessageTextParams) (*telego.Message, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*telego.Message), args.Error(1)
}

// DeleteMessage deletes a message sent via the bot.
func (m *MockBot) DeleteMessage(ctx context.Context, params *telego.DeleteMessageParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// SendPhoto sends a photo to a chat.
func (m *MockBot) SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*telego.Message), args.Error(1)
}

// SendDocument sends a document to a chat.
func (m *MockBot) SendDocument(ctx context.Context, params *telego.SendDocumentParams) (*telego.Message, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*telego.Message), args.Error(1)
}

// AnswerCallbackQuery answers a callback query sent from inline keyboards.
func (m *MockBot) AnswerCallbackQuery(ctx context.Context, params *telego.AnswerCallbackQueryParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// NewMockBotSuccess creates a MockBot that returns success for all operations.
// This is a helper function for tests that don't need to verify specific behavior.
// All expectations are optional (.Maybe()), so only called methods are checked.
func NewMockBotSuccess() *MockBot {
	mockBot := new(MockBot)

	mockBot.On("GetMe", mock.Anything).Return(&telego.User{
		ID:        123456789,
		FirstName: "Test",
		Username:  "test_bot",
	}, nil).Maybe()

	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
		Text:      "test message",
	}, nil).Maybe()

	mockBot.On("SetMyCommands", mock.Anything, mock.Anything).Return(nil).Maybe()

	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(nil).Maybe()

	mockBot.On("EditMessageText", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
		Text:      "edited message",
	}, nil).Maybe()

	mockBot.On("DeleteMessage", mock.Anything, mock.Anything).Return(nil).Maybe()

	mockBot.On("SendPhoto", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 2,
		Photo:     []telego.PhotoSize{{FileID: "test"}},
	}, nil).Maybe()

	mockBot.On("SendDocument", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 3,
		Document:  &telego.Document{FileID: "test"},
	}, nil).Maybe()

	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(nil).Maybe()

	return mockBot
}

// NewMockBotError creates a MockBot that returns the specified error for all operations.
// This is a helper function for tests that need to verify error handling.
func NewMockBotError(err error) *MockBot {
	mockBot := new(MockBot)

	mockBot.On("GetMe", mock.Anything).Return((*telego.User)(nil), err).Maybe()
	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return((*telego.Message)(nil), err).Maybe()
	mockBot.On("SetMyCommands", mock.Anything, mock.Anything).Return(err).Maybe()
	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(err).Maybe()
	mockBot.On("EditMessageText", mock.Anything, mock.Anything).Return((*telego.Message)(nil), err).Maybe()
	mockBot.On("DeleteMessage", mock.Anything, mock.Anything).Return(err).Maybe()
	mockBot.On("SendPhoto", mock.Anything, mock.Anything).Return((*telego.Message)(nil), err).Maybe()
	mockBot.On("SendDocument", mock.Anything, mock.Anything).Return((*telego.Message)(nil), err).Maybe()
	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(err).Maybe()

	return mockBot
}

// NewMockBotWithUpdates creates a MockBot that returns a channel with the specified updates.
// This is a helper function for testing long polling behavior.
//
// Parameters:
//   - updates: The updates to return from UpdatesViaLongPolling
//
// Returns:
//   - *MockBot: The configured mock bot
//   - <-chan telego.Update: The update channel (same as configured in the mock)
func NewMockBotWithUpdates(updates ...telego.Update) (*MockBot, <-chan telego.Update) {
	mockBot := new(MockBot)

	// Create a buffered channel with the updates
	updateCh := make(chan telego.Update, len(updates))
	for _, update := range updates {
		updateCh <- update
	}
	close(updateCh) // Close to indicate no more updates

	// Configure the mock to return this channel
	mockBot.On("UpdatesViaLongPolling", mock.Anything, mock.Anything, mock.Anything).Return(updateCh, nil)

	// Configure other methods to return success (optional)
	mockBot.On("GetMe", mock.Anything).Return(&telego.User{
		ID:        123456789,
		FirstName: "Test",
		Username:  "test_bot",
	}, nil).Maybe()

	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
		Text:      "test message",
	}, nil).Maybe()

	mockBot.On("SetMyCommands", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockBot.On("EditMessageText", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
		Text:      "edited message",
	}, nil).Maybe()
	mockBot.On("DeleteMessage", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockBot.On("SendPhoto", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 2,
		Photo:     []telego.PhotoSize{{FileID: "test"}},
	}, nil).Maybe()
	mockBot.On("SendDocument", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 3,
		Document:  &telego.Document{FileID: "test"},
	}, nil).Maybe()
	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(nil).Maybe()

	return mockBot, updateCh
}
