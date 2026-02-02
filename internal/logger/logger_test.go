package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_WithValidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid json config stdout",
			config: Config{
				Level:  "debug",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "valid text config stderr",
			config: Config{
				Level:  "info",
				Format: "text",
				Output: "stderr",
			},
			wantErr: false,
		},
		{
			name: "valid json config file",
			config: Config{
				Level:  "warn",
				Format: "json",
				Output: "/tmp/nexbot-test.log",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: Config{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: Config{
				Level:  "debug",
				Format: "xml",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid output path",
			config: Config{
				Level:  "debug",
				Format: "json",
				Output: "/nonexistent/directory/file.log",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger without error")
			}
		})
	}
}

func TestLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	logger.Debug("test debug message", Field{Key: "test", Value: "value"})

	if buf.Len() == 0 {
		t.Error("Expected log output, got empty buffer")
	}
}

func TestLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	logger.Info("test info message", Field{Key: "test", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Expected log to contain field 'test', got: %s", output)
	}
}

func TestLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	logger.Warn("test warn message", Field{Key: "key", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test warn message") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	err := &testError{msg: "test error"}
	logger.Error("test error message", err, Field{Key: "context", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test error message") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("Expected log to contain error message, got: %s", output)
	}
}

func TestLogger_DebugCtx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	ctx := context.Background()
	logger.DebugCtx(ctx, "test debug with context", Field{Key: "test", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test debug with context") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_InfoCtx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	ctx := context.Background()
	logger.InfoCtx(ctx, "test info with context", Field{Key: "key", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test info with context") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_WarnCtx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	ctx := context.Background()
	logger.WarnCtx(ctx, "test warn with context", Field{Key: "key", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test warn with context") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_ErrorCtx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	ctx := context.Background()
	err := &testError{msg: "test error"}
	logger.ErrorCtx(ctx, "test error with context", err, Field{Key: "context", Value: "value"})

	output := buf.String()
	if !strings.Contains(output, "test error with context") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := createTestLogger(t, buf, "json")

	loggerWithFields := logger.With(
		Field{Key: "component", Value: "test"},
		Field{Key: "version", Value: "1.0.0"},
	)

	loggerWithFields.Info("message with fields")

	output := buf.String()
	if !strings.Contains(output, "component") {
		t.Errorf("Expected log to contain 'component', got: %s", output)
	}
	if !strings.Contains(output, "version") {
		t.Errorf("Expected log to contain 'version', got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantDebug bool
		wantInfo  bool
		wantWarn  bool
		wantError bool
	}{
		{
			name:      "debug level shows all",
			level:     "debug",
			wantDebug: true,
			wantInfo:  true,
			wantWarn:  true,
			wantError: true,
		},
		{
			name:      "info level skips debug",
			level:     "info",
			wantDebug: false,
			wantInfo:  true,
			wantWarn:  true,
			wantError: true,
		},
		{
			name:      "warn level skips debug and info",
			level:     "warn",
			wantDebug: false,
			wantInfo:  false,
			wantWarn:  true,
			wantError: true,
		},
		{
			name:      "error level shows only errors",
			level:     "error",
			wantDebug: false,
			wantInfo:  false,
			wantWarn:  false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			// Create custom handler that writes to buffer
			level, _ := parseLevel(tt.level)
			handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})
			logger := &Logger{
				slog: slog.New(handler),
			}

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message", nil)

			output := buf.String()

			hasDebug := strings.Contains(output, "debug message")
			hasInfo := strings.Contains(output, "info message")
			hasWarn := strings.Contains(output, "warn message")
			hasError := strings.Contains(output, "error message")

			if hasDebug != tt.wantDebug {
				t.Errorf("Expected debug=%v, got %v", tt.wantDebug, hasDebug)
			}
			if hasInfo != tt.wantInfo {
				t.Errorf("Expected info=%v, got %v", tt.wantInfo, hasInfo)
			}
			if hasWarn != tt.wantWarn {
				t.Errorf("Expected warn=%v, got %v", tt.wantWarn, hasWarn)
			}
			if hasError != tt.wantError {
				t.Errorf("Expected error=%v, got %v", tt.wantError, hasError)
			}
		})
	}
}

func TestLogger_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := &Logger{
		slog: slog.New(handler),
	}

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := &Logger{
		slog: slog.New(handler),
	}

	logger.Info("test message", Field{Key: "key", Value: "value"})

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	// Verify fields
	if result["msg"] != "test message" {
		t.Errorf("Expected msg='test message', got: %v", result["msg"])
	}
}

// Helper function to create test logger with buffer output
func createTestLogger(t *testing.T, buf *bytes.Buffer, format string) *Logger {
	t.Helper()

	// Create a custom handler that writes to buffer
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	return &Logger{
		slog: slog.New(handler),
	}
}

// testError реализует интерфейс error для тестов
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark logging
func BenchmarkLogger_Debug(b *testing.B) {
	logger, _ := New(Config{
		Level:  "debug",
		Format: "json",
	})
	// Create handler with no output (discard)
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger.slog = slog.New(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark debug message", Field{Key: "iteration", Value: i})
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger, _ := New(Config{
		Level:  "info",
		Format: "json",
	})
	// Create handler with no output (discard)
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger.slog = slog.New(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark info message", Field{Key: "iteration", Value: i})
	}
}
