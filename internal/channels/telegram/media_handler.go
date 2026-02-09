package telegram

import (
	"fmt"
	"os"
	"strings"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/mymmrac/telego"
)

// prepareMediaParams is a generic function that prepares parameters for sending media (photo/document)
func prepareMediaParams[T any](
	conn *Connector,
	msg bus.OutboundMessage,
	chatID int64,
	setMediaField func(*T, telego.InputFile),
) (*T, error) {
	// Initialize params with ChatID
	var params T
	chatIDField, ok := any(&params).(interface{ SetChatID(telego.ChatID) })
	if ok {
		chatIDField.SetChatID(telego.ChatID{ID: chatID})
	}

	// Set caption if provided
	captionField, ok := any(&params).(interface{ SetCaption(string) })
	if ok && msg.Content != "" {
		captionField.SetCaption(msg.Content)
	}

	if msg.Media == nil {
		return &params, fmt.Errorf("media data is required")
	}

	media := msg.Media

	// Priority order: LocalPath > FileID > URL
	if media.LocalPath != "" {
		if !conn.isValidFilePath(media.LocalPath) {
			return &params, fmt.Errorf("invalid file path: %s", media.LocalPath)
		}

		// Open file for reading
		file, err := os.Open(media.LocalPath)
		if err != nil {
			return &params, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		inputFile := telego.InputFile{File: file}
		setMediaField(&params, inputFile)
	} else if media.FileID != "" {
		inputFile := telego.InputFile{FileID: media.FileID}
		setMediaField(&params, inputFile)
	} else if media.URL != "" {
		inputFile := telego.InputFile{URL: media.URL}
		setMediaField(&params, inputFile)
	} else {
		return &params, fmt.Errorf("no valid media source provided (local_path, file_id, or url)")
	}

	return &params, nil
}

// isValidFilePath validates a file path
func (c *Connector) isValidFilePath(path string) bool {
	if path == "" {
		return false
	}

	// Check for absolute path
	if strings.HasPrefix(path, "/") {
		return true
	}

	// Check for relative path starting with . or ..
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return true
	}

	// Path with just filename is also valid
	return true
}

// buildInlineKeyboard converts an InlineKeyboard to Telegram's InlineKeyboardMarkup format
func (c *Connector) buildInlineKeyboard(keyboard *bus.InlineKeyboard) *telego.InlineKeyboardMarkup {
	if keyboard == nil {
		return nil
	}

	markup := &telego.InlineKeyboardMarkup{
		InlineKeyboard: make([][]telego.InlineKeyboardButton, len(keyboard.Rows)),
	}

	for i, row := range keyboard.Rows {
		buttons := make([]telego.InlineKeyboardButton, len(row))
		for j, button := range row {
			buttons[j] = telego.InlineKeyboardButton{
				Text:         button.Text,
				CallbackData: button.Data,
			}
		}
		markup.InlineKeyboard[i] = buttons
	}

	return markup
}
