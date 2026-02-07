package channels

import (
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// ErrorDetails - универсальный интерфейс для детализации ошибок каналов
// Позволяет расширяемость для разных каналов (Telegram, Slack, Web, API и т.д.)
type ErrorDetails interface {
	// Error возвращает текстовое описание ошибки
	Error() string

	// IsRetryable указывает, можно ли повторить отправку
	IsRetryable() bool

	// RetryAfter возвращает задержку перед повторной отправкой
	RetryAfter() time.Duration

	// ToLLMContext возвращает структурированное описание для LLM
	ToLLMContext() string

	// LogFields возвращает поля для структурированного логирования
	LogFields() []logger.Field
}

// TelegramErrorDetails - детализация ошибки Telegram API
type TelegramErrorDetails struct {
	ErrorCode       int       // Код ошибки (400, 429, 403 и т.д.)
	Description     string    // Описание ошибки от Telegram
	RetryAfterSec   int       // Задержка в секундах (для rate limiting)
	OriginalMessage string    // Сообщение, которое вызвало ошибку
	ChatID          int64     // ID чата
	Timestamp       time.Time // Время ошибки
}

// Error возвращает текстовое описание ошибки
func (d *TelegramErrorDetails) Error() string {
	return d.Description
}

// IsRetryable проверяет, можно ли повторить отправку
func (d *TelegramErrorDetails) IsRetryable() bool {
	// Rate limiting (429) и временные ошибки можно повторить
	return d.ErrorCode == 429 || (d.ErrorCode >= 500 && d.ErrorCode < 600)
}

// RetryAfter возвращает задержку перед повторной отправкой
func (d *TelegramErrorDetails) RetryAfter() time.Duration {
	if d.RetryAfterSec > 0 {
		return time.Duration(d.RetryAfterSec) * time.Second
	}
	// Для временных ошибок - дефолтная задержка
	if d.ErrorCode >= 500 && d.ErrorCode < 600 {
		return 5 * time.Second
	}
	return 0
}

// ToLLMContext возвращает структурированное описание для LLM
func (d *TelegramErrorDetails) ToLLMContext() string {
	return fmt.Sprintf(`Telegram Error Details:
- Error Code: %d
- Description: %s
- Retryable: %t
- Retry After: %s
- Original Message: %q
- Chat ID: %d
- Timestamp: %s`,
		d.ErrorCode,
		d.Description,
		d.IsRetryable(),
		d.RetryAfter().String(),
		d.OriginalMessage,
		d.ChatID,
		d.Timestamp.Format(time.RFC3339))
}

// LogFields возвращает поля для структурированного логирования
func (d *TelegramErrorDetails) LogFields() []logger.Field {
	return []logger.Field{
		{Key: "error_code", Value: d.ErrorCode},
		{Key: "error_description", Value: d.Description},
		{Key: "retry_after", Value: d.RetryAfterSec},
		{Key: "chat_id", Value: d.ChatID},
	}
}
