package config

import (
	"strings"
)

// maskSecret маскирует секрет, оставляя только первые 4 и последние 4 символа
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}

	// Если секрет слишком короткий, маскируем полностью
	if len(secret) < 8 {
		return "***"
	}

	// Оставляем первые 4 и последние 4 символа
	prefix := secret[:4]
	suffix := secret[len(secret)-4:]

	// Заменяем середину звездочками
	masked := strings.Repeat("*", len(secret)-8)

	return prefix + masked + suffix
}

// maskAPIKey маскирует API ключ для отображения в ошибках и логах
func maskAPIKey(apiKey string) string {
	return maskSecret(apiKey)
}

// maskTelegramToken маскирует Telegram токен для отображения в ошибках и логах
func maskTelegramToken(token string) string {
	if token == "" {
		return ""
	}

	// Telegram token имеет формат <bot_id>:<token>
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		// Если формат неверный, маскируем как обычный секрет
		return maskSecret(token)
	}

	// Маскируем только часть токена, оставляя bot_id видимым для диагностики
	botID := parts[0]
	tokenPart := maskSecret(parts[1])

	return botID + ":" + tokenPart
}

// formatValidationError форматирует ошибку валидации с маскированными секретами
// и дружественным описанием проблемы
func formatValidationError(field, message string, secret string) error {
	maskedSecret := ""
	if secret != "" {
		maskedSecret = maskSecret(secret)
	}

	var errorMsg string
	if maskedSecret != "" {
		errorMsg = field + ": " + message + " (value: " + maskedSecret + ")"
	} else {
		errorMsg = field + ": " + message
	}

	return &ValidationError{Field: field, Message: errorMsg}
}

// ValidationError представляет ошибку валидации с дополнительной информацией
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
