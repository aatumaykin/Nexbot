// Package logger provides a structured logging wrapper around Go's slog package.
// It supports both JSON and text formatted output, multiple log levels (debug, info, warn, error),
// and flexible output destinations (stdout, stderr, or file paths).
//
// Example usage:
//
//	log, err := logger.New(logger.Config{
//	    Level:  "info",
//	    Format: "json",
//	    Output: "stdout",
//	})
//	if err != nil {
//	    log.Fatal("Failed to initialize logger", err)
//	}
//
//	log.Info("Application started", logger.Field{Key: "version", Value: "1.0.0"})
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Config представляет конфигурацию logger
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output string // stdout, stderr, или путь к файлу
}

// Logger представляет обёртку вокруг slog.Logger
type Logger struct {
	slog *slog.Logger
}

// Field представляет поле для structured logging
type Field struct {
	Key   string
	Value any
}

// New создает новый logger с заданной конфигурацией
func New(cfg Config) (*Logger, error) {
	// Парсинг уровня логирования
	level, valid := parseLevel(cfg.Level)
	if !valid {
		return nil, fmt.Errorf("invalid log level: %s (expected: debug, info, warn, error)", cfg.Level)
	}

	// Определение writer для вывода
	var writer io.Writer
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// Путь к файлу - разворачиваем ~ в домашнюю директорию
		filePath := cfg.Output
		if strings.HasPrefix(filePath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			filePath = filepath.Join(homeDir, filePath[2:])
		}
		filePath = filepath.Clean(filePath)
		// Создаём директорию, если она не существует
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", filePath, err)
		}
		writer = file
	}

	// Создание handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		return nil, fmt.Errorf("invalid log format: %s (expected: json, text)", cfg.Format)
	}

	return &Logger{
		slog: slog.New(handler),
	}, nil
}

// parseLevel конвертирует строку уровня в slog.Level
func parseLevel(level string) (slog.Level, bool) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false // Invalid
	}
}

// Debug логирует сообщение на уровне debug
func (l *Logger) Debug(msg string, fields ...Field) {
	l.slog.Debug(msg, l.fieldsToAny(fields...)...)
}

// Info логирует сообщение на уровне info
func (l *Logger) Info(msg string, fields ...Field) {
	l.slog.Info(msg, l.fieldsToAny(fields...)...)
}

// Warn логирует сообщение на уровне warn
func (l *Logger) Warn(msg string, fields ...Field) {
	l.slog.Warn(msg, l.fieldsToAny(fields...)...)
}

// Error логирует сообщение на уровне error с ошибкой
func (l *Logger) Error(msg string, err error, fields ...Field) {
	allFields := append([]Field{{Key: "error", Value: err}}, fields...)
	l.slog.Error(msg, l.fieldsToAny(allFields...)...)
}

// DebugCtx логирует сообщение с контекстом на уровне debug
func (l *Logger) DebugCtx(ctx context.Context, msg string, fields ...Field) {
	l.slog.DebugContext(ctx, msg, l.fieldsToAny(fields...)...)
}

// InfoCtx логирует сообщение с контекстом на уровне info
func (l *Logger) InfoCtx(ctx context.Context, msg string, fields ...Field) {
	l.slog.InfoContext(ctx, msg, l.fieldsToAny(fields...)...)
}

// WarnCtx логирует сообщение с контекстом на уровне warn
func (l *Logger) WarnCtx(ctx context.Context, msg string, fields ...Field) {
	l.slog.WarnContext(ctx, msg, l.fieldsToAny(fields...)...)
}

// ErrorCtx логирует сообщение с контекстом на уровне error с ошибкой
func (l *Logger) ErrorCtx(ctx context.Context, msg string, err error, fields ...Field) {
	allFields := append([]Field{{Key: "error", Value: err}}, fields...)
	l.slog.ErrorContext(ctx, msg, l.fieldsToAny(allFields...)...)
}

// fieldsToAny конвертирует срез Field в срез slog.Attr
func (l *Logger) fieldsToAny(fields ...Field) []any {
	result := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		result = append(result, f.Key, f.Value)
	}
	return result
}

// With возвращает новый logger с добавленными полями
func (l *Logger) With(fields ...Field) *Logger {
	return &Logger{
		slog: l.slog.With(l.fieldsToAny(fields...)...),
	}
}

// StdLogger возвращает стандартный logger для совместимости
func (l *Logger) StdLogger() *slog.Logger {
	return l.slog
}

// Default возвращает стандартный logger для удобства
func Default() *slog.Logger {
	return slog.Default()
}

// SetDefault устанавливает стандартный logger
func SetDefault(l *Logger) {
	slog.SetDefault(l.slog)
}
