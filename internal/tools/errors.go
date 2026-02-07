package tools

import (
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// ErrorType - тип ошибки инструмента
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation" // Ошибка валидации аргументов
	ErrorTypePermission ErrorType = "permission" // Ошибка прав доступа
	ErrorTypeTimeout    ErrorType = "timeout"    // Таймаут выполнения
	ErrorTypeNotFound   ErrorType = "not_found"  // Ресурс не найден
	ErrorTypeExecution  ErrorType = "execution"  // Ошибка выполнения
	ErrorTypeRateLimit  ErrorType = "rate_limit" // Лимит частоты запросов
	ErrorTypeStorage    ErrorType = "storage"    // Ошибка хранения
	ErrorTypeDisabled   ErrorType = "disabled"   // Инструмент отключен
	ErrorTypeUnknown    ErrorType = "unknown"    // Неизвестная ошибка
)

// ToolError - структурированная ошибка инструмента
type ToolError struct {
	Type       ErrorType      `json:"type"`                 // Тип ошибки
	Code       string         `json:"code"`                 // Код ошибки
	Message    string         `json:"message"`              // Описание ошибки
	Details    map[string]any `json:"details,omitempty"`    // Дополнительные детали
	Suggestion string         `json:"suggestion,omitempty"` // Предложение по исправлению
	Retryable  bool           `json:"retryable"`            // Можно ли повторить
	ExitCode   int            `json:"exit_code,omitempty"`  // Exit code (для shell)
	Command    string         `json:"command,omitempty"`    // Команда (для shell)
	Path       string         `json:"path,omitempty"`       // Путь (для file)
	Args       string         `json:"args,omitempty"`       // Аргументы (общее)
}

// Error возвращает текстовое описание ошибки
func (e *ToolError) Error() string {
	return e.Message
}

// IsRetryable проверяет, можно ли повторить выполнение
func (e *ToolError) IsRetryable() bool {
	return e.Retryable
}

// ToLLMContext возвращает структурированное описание для LLM
func (e *ToolError) ToLLMContext() string {
	var details string
	if len(e.Details) > 0 {
		details = "\nDetails:"
		for k, v := range e.Details {
			details += fmt.Sprintf("\n- %s: %v", k, v)
		}
	}

	var suggestion string
	if e.Suggestion != "" {
		suggestion = fmt.Sprintf("\nSuggestion: %s", e.Suggestion)
	}

	var command string
	if e.Command != "" {
		command = fmt.Sprintf("\nCommand: %q", e.Command)
	}

	var path string
	if e.Path != "" {
		path = fmt.Sprintf("\nPath: %q", e.Path)
	}

	var args string
	if e.Args != "" {
		args = fmt.Sprintf("\nArgs: %q", e.Args)
	}

	var exitCode string
	if e.ExitCode != 0 {
		exitCode = fmt.Sprintf("\nExit Code: %d", e.ExitCode)
	}

	actions := e.getSuggestedActions()

	return fmt.Sprintf(`Tool Error Details:
 - Type: %s
 - Code: %s
 - Message: %s
 - Retryable: %t%s%s%s%s%s%s

Suggested Actions:
%s`,
		e.Type,
		e.Code,
		e.Message,
		e.Retryable,
		details,
		suggestion,
		command,
		path,
		args,
		exitCode,
		actions)
}

// getSuggestedActions возвращает предложения по исправлению
func (e *ToolError) getSuggestedActions() string {
	switch e.Type {
	case ErrorTypeValidation:
		return "- Check the arguments\n- Verify required fields\n- Check data format"
	case ErrorTypePermission:
		return "- Check file permissions\n- Verify user rights\n- Ensure directory is accessible"
	case ErrorTypeTimeout:
		return "- Increase timeout\n- Break into smaller tasks\n- Optimize operation"
	case ErrorTypeNotFound:
		return "- Verify path is correct\n- Check file exists\n- Ensure directory is valid"
	case ErrorTypeExecution:
		if e.ExitCode > 0 {
			return fmt.Sprintf("- Command failed with exit code %d\n- Check command syntax\n- Verify dependencies", e.ExitCode)
		}
		return "- Check execution context\n- Verify dependencies\n- Review error details"
	case ErrorTypeRateLimit:
		return "- Wait and retry\n- Reduce request frequency\n- Check API quotas"
	case ErrorTypeStorage:
		return "- Check storage path\n- Verify write permissions\n- Ensure disk space available"
	case ErrorTypeDisabled:
		return "- Tool is disabled in configuration\n- Enable the tool if needed\n- Check configuration"
	default:
		return "- Review error details\n- Try alternative approach\n- Check system status"
	}
}

// LogFields возвращает поля для структурированного логирования
func (e *ToolError) LogFields() []logger.Field {
	fields := []logger.Field{
		{Key: "error_type", Value: e.Type},
		{Key: "error_code", Value: e.Code},
		{Key: "error_message", Value: e.Message},
		{Key: "retryable", Value: e.Retryable},
	}

	if e.ExitCode != 0 {
		fields = append(fields, logger.Field{Key: "exit_code", Value: e.ExitCode})
	}

	if e.Command != "" {
		fields = append(fields, logger.Field{Key: "command", Value: e.Command})
	}

	if e.Path != "" {
		fields = append(fields, logger.Field{Key: "path", Value: e.Path})
	}

	if e.Suggestion != "" {
		fields = append(fields, logger.Field{Key: "error_suggestion", Value: e.Suggestion})
	}

	return fields
}

// NewValidationError создает ошибку валидации
func NewValidationError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Type:      ErrorTypeValidation,
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

// NewPermissionError создает ошибку прав доступа
func NewPermissionError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Type:      ErrorTypePermission,
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

// NewTimeoutError создает ошибку таймаута
func NewTimeoutError(code, message string, details map[string]any) *ToolError {
	return &ToolError{
		Type:      ErrorTypeTimeout,
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: true,
	}
}

// NewNotFoundError создает ошибку "не найден"
func NewNotFoundError(code, message, suggestion string) *ToolError {
	return &ToolError{
		Type:       ErrorTypeNotFound,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Retryable:  false,
	}
}

// NewExecutionError создает ошибку выполнения
func NewExecutionError(code, message, suggestion string, exitCode int) *ToolError {
	details := map[string]any{}
	if exitCode != 0 {
		details["exit_code"] = exitCode
	}
	return &ToolError{
		Type:       ErrorTypeExecution,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Details:    details,
		ExitCode:   exitCode,
		Retryable:  exitCode > 0 && exitCode < 128, // Exit code 0-127 может быть retryable
	}
}

// NewRateLimitError создает ошибку лимита
func NewRateLimitError(code, message string, retryAfter time.Duration) *ToolError {
	details := make(map[string]any)
	if retryAfter > 0 {
		details["retry_after"] = retryAfter.String()
	}

	return &ToolError{
		Type:      ErrorTypeRateLimit,
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: true,
	}
}

// NewStorageError создает ошибку хранения
func NewStorageError(code, message string) *ToolError {
	return &ToolError{
		Type:      ErrorTypeStorage,
		Code:      code,
		Message:   message,
		Retryable: true,
	}
}

// NewDisabledError создает ошибку отключенного инструмента
func NewDisabledError(code, message string) *ToolError {
	return &ToolError{
		Type:      ErrorTypeDisabled,
		Code:      code,
		Message:   message,
		Retryable: false,
	}
}
