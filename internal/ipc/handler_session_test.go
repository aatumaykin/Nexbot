package ipc

import (
	"context"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Test 9: Сессия создается корректно
func TestHandlerSessionCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	if err := messageBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}
	defer func() {
		if err := messageBus.Stop(); err != nil {
			t.Logf("Failed to stop message bus: %v", err)
		}
	}()

	handler, err := NewHandler(log, tempDir, messageBus)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	sessionID := "test_session"

	// Проверить, что сессия не существует до создания
	exists, err := handler.sessionMgr.Exists(sessionID)
	if err != nil {
		t.Fatalf("Failed to check session existence: %v", err)
	}
	if exists {
		t.Error("Session should not exist before creation")
	}

	// Создать сессию
	session, created, err := handler.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if !created {
		t.Error("Session should be marked as newly created")
	}
	if session.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, session.ID)
	}

	// Проверить, что сессия теперь существует
	exists, err = handler.sessionMgr.Exists(sessionID)
	if err != nil {
		t.Fatalf("Failed to check session existence: %v", err)
	}
	if !exists {
		t.Error("Session should exist after creation")
	}
}

// Test 10: validateChannel позволяет все каналы (текущая реализация)
func TestValidateChannel(t *testing.T) {
	// Текущая реализация validateChannel всегда возвращает nil
	// Этот тест проверяет текущее поведение
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	handler, err := NewHandler(log, tempDir, messageBus)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	tests := []struct {
		name    string
		channel string
		wantErr bool
	}{
		{"valid telegram", "telegram", false},
		{"valid discord", "discord", false},
		{"valid slack", "slack", false},
		{"valid web", "web", false},
		{"valid api", "api", false},
		{"invalid channel", "invalid_channel", false}, // Текущая реализация разрешает все
		{"empty channel", "", false},                  // Текущая реализация разрешает пустой
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateChannel(tt.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateChannel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test 11: validateSession проверяет существование сессии
func TestValidateSession(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	handler, err := NewHandler(log, tempDir, messageBus)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	sessionID := "test_session"

	// Проверить несуществующую сессию
	if handler.validateSession(sessionID) {
		t.Error("Session should not exist")
	}

	// Создать сессию
	_, _, err = handler.sessionMgr.GetOrCreate(sessionID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Проверить существующую сессию
	if !handler.validateSession(sessionID) {
		t.Error("Session should exist")
	}
}
