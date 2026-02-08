package telegram

import (
	"context"

	"github.com/mymmrac/telego"
)

// BotInterface defines the Telegram bot API methods used by the connector.
// This interface allows creating mock implementations for testing without
// depending on the concrete telego.Bot implementation.
type BotInterface interface {
	// GetMe returns basic information about the bot.
	GetMe(ctx context.Context) (*telego.User, error)

	// SendMessage sends a text message to a chat.
	SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error)

	// SetMyCommands sets the bot's command list in the bot menu.
	SetMyCommands(ctx context.Context, params *telego.SetMyCommandsParams) error

	// UpdatesViaLongPolling starts long polling for Telegram updates.
	// Returns a channel that will receive updates as they arrive.
	UpdatesViaLongPolling(ctx context.Context, params *telego.GetUpdatesParams, opts ...telego.LongPollingOption) (<-chan telego.Update, error)

	// SendChatAction sends a chat action (e.g., typing) to a chat.
	SendChatAction(ctx context.Context, params *telego.SendChatActionParams) error

	// EditMessageText edits text of a message sent via the bot.
	EditMessageText(ctx context.Context, params *telego.EditMessageTextParams) (*telego.Message, error)

	// DeleteMessage deletes a message sent via the bot.
	DeleteMessage(ctx context.Context, params *telego.DeleteMessageParams) error

	// SendPhoto sends a photo to a chat.
	SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error)

	// SendDocument sends a document to a chat.
	SendDocument(ctx context.Context, params *telego.SendDocumentParams) (*telego.Message, error)
}

// telegoAdapter wraps telego.Bot to implement BotInterface.
// This is a simple adapter that delegates all calls to the underlying bot.
type telegoAdapter struct {
	bot *telego.Bot
}

// NewBotAdapter creates a new BotInterface from a telego.Bot instance.
// This allows using telego.Bot where BotInterface is required,
// enabling both real bot usage and mock implementations in tests.
func NewBotAdapter(bot *telego.Bot) BotInterface {
	return &telegoAdapter{bot: bot}
}

// GetMe returns basic information about the bot.
func (a *telegoAdapter) GetMe(ctx context.Context) (*telego.User, error) {
	return a.bot.GetMe(ctx)
}

// SendMessage sends a text message to a chat.
func (a *telegoAdapter) SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error) {
	return a.bot.SendMessage(ctx, params)
}

// SetMyCommands sets the bot's command list in the bot menu.
func (a *telegoAdapter) SetMyCommands(ctx context.Context, params *telego.SetMyCommandsParams) error {
	return a.bot.SetMyCommands(ctx, params)
}

// UpdatesViaLongPolling starts long polling for Telegram updates.
// Returns a channel that will receive updates as they arrive.
func (a *telegoAdapter) UpdatesViaLongPolling(ctx context.Context, params *telego.GetUpdatesParams, opts ...telego.LongPollingOption) (<-chan telego.Update, error) {
	return a.bot.UpdatesViaLongPolling(ctx, params, opts...)
}

// SendChatAction sends a chat action (e.g., typing) to a chat.
func (a *telegoAdapter) SendChatAction(ctx context.Context, params *telego.SendChatActionParams) error {
	return a.bot.SendChatAction(ctx, params)
}

// EditMessageText edits text of a message sent via the bot.
func (a *telegoAdapter) EditMessageText(ctx context.Context, params *telego.EditMessageTextParams) (*telego.Message, error) {
	return a.bot.EditMessageText(ctx, params)
}

// DeleteMessage deletes a message sent via the bot.
func (a *telegoAdapter) DeleteMessage(ctx context.Context, params *telego.DeleteMessageParams) error {
	return a.bot.DeleteMessage(ctx, params)
}

// SendPhoto sends a photo to a chat.
func (a *telegoAdapter) SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error) {
	return a.bot.SendPhoto(ctx, params)
}

// SendDocument sends a document to a chat.
func (a *telegoAdapter) SendDocument(ctx context.Context, params *telego.SendDocumentParams) (*telego.Message, error) {
	return a.bot.SendDocument(ctx, params)
}
