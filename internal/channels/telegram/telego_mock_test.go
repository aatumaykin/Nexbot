package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockBot_EditMessageText(t *testing.T) {
	ctx := context.Background()
	messageID := int(1)
	newText := "edited text"

	t.Run("success", func(t *testing.T) {
		mockBot := new(MockBot)
		expectedMessage := &telego.Message{
			MessageID: messageID,
			Text:      newText,
		}
		mockBot.On("EditMessageText", ctx, mock.Anything).Return(expectedMessage, nil)

		got, err := mockBot.EditMessageText(ctx, &telego.EditMessageTextParams{
			MessageID: messageID,
			Text:      newText,
		})

		require.NoError(t, err)
		assert.Equal(t, expectedMessage, got)
		mockBot.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockBot := new(MockBot)
		testErr := errors.New("edit failed")
		mockBot.On("EditMessageText", ctx, mock.Anything).Return((*telego.Message)(nil), testErr)

		got, err := mockBot.EditMessageText(ctx, &telego.EditMessageTextParams{})

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, testErr, err)
		mockBot.AssertExpectations(t)
	})
}

func TestMockBot_DeleteMessage(t *testing.T) {
	ctx := context.Background()
	chatID := int64(123456789)
	messageID := int(1)

	t.Run("success", func(t *testing.T) {
		mockBot := new(MockBot)
		mockBot.On("DeleteMessage", ctx, mock.Anything).Return(nil)

		err := mockBot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})

		require.NoError(t, err)
		mockBot.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockBot := new(MockBot)
		testErr := errors.New("delete failed")
		mockBot.On("DeleteMessage", ctx, mock.Anything).Return(testErr)

		err := mockBot.DeleteMessage(ctx, &telego.DeleteMessageParams{})

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		mockBot.AssertExpectations(t)
	})
}

func TestMockBot_SendPhoto(t *testing.T) {
	ctx := context.Background()
	chatID := int64(123456789)
	caption := "test photo"

	t.Run("success", func(t *testing.T) {
		mockBot := new(MockBot)
		expectedMessage := &telego.Message{
			MessageID: 2,
			Photo:     []telego.PhotoSize{{FileID: "photo_test_123"}},
		}
		mockBot.On("SendPhoto", ctx, mock.Anything).Return(expectedMessage, nil)

		got, err := mockBot.SendPhoto(ctx, &telego.SendPhotoParams{
			ChatID:  telego.ChatID{ID: chatID},
			Caption: caption,
		})

		require.NoError(t, err)
		assert.Equal(t, expectedMessage, got)
		mockBot.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockBot := new(MockBot)
		testErr := errors.New("send failed")
		mockBot.On("SendPhoto", ctx, mock.Anything).Return((*telego.Message)(nil), testErr)

		got, err := mockBot.SendPhoto(ctx, &telego.SendPhotoParams{})

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, testErr, err)
		mockBot.AssertExpectations(t)
	})
}

func TestMockBot_SendDocument(t *testing.T) {
	ctx := context.Background()
	chatID := int64(123456789)
	caption := "test document"

	t.Run("success", func(t *testing.T) {
		mockBot := new(MockBot)
		expectedMessage := &telego.Message{
			MessageID: 3,
			Document:  &telego.Document{FileID: "doc_test_123"},
		}
		mockBot.On("SendDocument", ctx, mock.Anything).Return(expectedMessage, nil)

		got, err := mockBot.SendDocument(ctx, &telego.SendDocumentParams{
			ChatID:  telego.ChatID{ID: chatID},
			Caption: caption,
		})

		require.NoError(t, err)
		assert.Equal(t, expectedMessage, got)
		mockBot.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockBot := new(MockBot)
		testErr := errors.New("send failed")
		mockBot.On("SendDocument", ctx, mock.Anything).Return((*telego.Message)(nil), testErr)

		got, err := mockBot.SendDocument(ctx, &telego.SendDocumentParams{})

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, testErr, err)
		mockBot.AssertExpectations(t)
	})
}

func TestNewMockBotSuccess_WithNewMethods(t *testing.T) {
	ctx := context.Background()
	mockBot := NewMockBotSuccess()

	t.Run("EditMessageText returns success", func(t *testing.T) {
		msg, err := mockBot.EditMessageText(ctx, &telego.EditMessageTextParams{
			MessageID: 1,
			Text:      "test",
		})
		require.NoError(t, err)
		assert.Equal(t, int(1), msg.MessageID)
		assert.Equal(t, "edited message", msg.Text)
	})

	t.Run("DeleteMessage returns success", func(t *testing.T) {
		err := mockBot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			MessageID: 1,
		})
		require.NoError(t, err)
	})

	t.Run("SendPhoto returns success", func(t *testing.T) {
		msg, err := mockBot.SendPhoto(ctx, &telego.SendPhotoParams{})
		require.NoError(t, err)
		assert.Equal(t, int(2), msg.MessageID)
		assert.NotEmpty(t, msg.Photo)
	})

	t.Run("SendDocument returns success", func(t *testing.T) {
		msg, err := mockBot.SendDocument(ctx, &telego.SendDocumentParams{})
		require.NoError(t, err)
		assert.Equal(t, int(3), msg.MessageID)
		assert.NotNil(t, msg.Document)
	})
}

func TestNewMockBotError_WithNewMethods(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")
	mockBot := NewMockBotError(testErr)

	t.Run("EditMessageText returns error", func(t *testing.T) {
		msg, err := mockBot.EditMessageText(ctx, &telego.EditMessageTextParams{})
		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, testErr, err)
	})

	t.Run("DeleteMessage returns error", func(t *testing.T) {
		err := mockBot.DeleteMessage(ctx, &telego.DeleteMessageParams{})
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("SendPhoto returns error", func(t *testing.T) {
		msg, err := mockBot.SendPhoto(ctx, &telego.SendPhotoParams{})
		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, testErr, err)
	})

	t.Run("SendDocument returns error", func(t *testing.T) {
		msg, err := mockBot.SendDocument(ctx, &telego.SendDocumentParams{})
		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, testErr, err)
	})
}

func TestNewMockBotWithUpdates_WithNewMethods(t *testing.T) {
	ctx := context.Background()
	mockBot, _ := NewMockBotWithUpdates(telego.Update{})

	t.Run("EditMessageText returns success", func(t *testing.T) {
		msg, err := mockBot.EditMessageText(ctx, &telego.EditMessageTextParams{
			MessageID: 1,
			Text:      "test",
		})
		require.NoError(t, err)
		assert.Equal(t, int(1), msg.MessageID)
		assert.Equal(t, "edited message", msg.Text)
	})

	t.Run("DeleteMessage returns success", func(t *testing.T) {
		err := mockBot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			MessageID: 1,
		})
		require.NoError(t, err)
	})

	t.Run("SendPhoto returns success", func(t *testing.T) {
		msg, err := mockBot.SendPhoto(ctx, &telego.SendPhotoParams{})
		require.NoError(t, err)
		assert.Equal(t, int(2), msg.MessageID)
		assert.NotEmpty(t, msg.Photo)
	})

	t.Run("SendDocument returns success", func(t *testing.T) {
		msg, err := mockBot.SendDocument(ctx, &telego.SendDocumentParams{})
		require.NoError(t, err)
		assert.Equal(t, int(3), msg.MessageID)
		assert.NotNil(t, msg.Document)
	})
}
