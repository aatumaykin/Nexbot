package tools

import (
	"fmt"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// ToolError - структурированная ошибка выполнения инструмента
type ToolError struct {
	Code       string         `json:"code"`                 // Код ошибки для программной обработки
	Message    string         `json:"message"`              // Человекочитаемое сообщение
	Details    map[string]any `json:"details,omitempty"`    // Дополнительные детали
	Suggestion string         `json:"suggestion,omitempty"` // Предложение по исправлению
}

// Error реализует интерфейс error
func (e *ToolError) Error() string {
	return e.Message
}

// ToLLMContext возвращает структурированное описание для LLM
func (e *ToolError) ToLLMContext() string {
	var result string
	result = fmt.Sprintf("Tool Error:\n - Code: %s\n - Message: %s", e.Code, e.Message)

	if e.Suggestion != "" {
		result += fmt.Sprintf("\n - Suggestion: %s", e.Suggestion)
	}

	if len(e.Details) > 0 {
		result += "\n - Details:"
		for key, value := range e.Details {
			result += fmt.Sprintf("\n     - %s: %v", key, value)
		}
	}

	return result
}

// LogFields возвращает поля для структурированного логирования
func (e *ToolError) LogFields() []logger.Field {
	fields := []logger.Field{
		{Key: "error_code", Value: e.Code},
		{Key: "error_message", Value: e.Message},
	}
	if e.Suggestion != "" {
		fields = append(fields, logger.Field{Key: "error_suggestion", Value: e.Suggestion})
	}
	return fields
}

// NewNotFoundError создает ошибку "не найдено"
func NewNotFoundError(code, message, suggestion string) *ToolError {
	return &ToolError{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
	}
}

// NewTimeoutError создает ошибку "таймаут"
func NewTimeoutError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewExecutionError создает ошибку выполнения
func NewExecutionError(code, message, suggestion string, exitCode int) *ToolError {
	return &ToolError{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Details:    map[string]any{"exit_code": exitCode},
	}
}

// NewValidationError создает ошибку валидации
func NewValidationError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewPermissionError создает ошибку доступа
func NewPermissionError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
