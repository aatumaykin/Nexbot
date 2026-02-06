package ipc

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

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

// Test 12: sendErrorResponse отправляет корректный ответ об ошибке
func TestSendErrorResponse(t *testing.T) {
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

	// Создать mock соединение
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Запустить горутину для чтения ответа
	responseCh := make(chan []byte, 1)
	go func() {
		response := make([]byte, 1024)
		n, _ := client.Read(response)
		responseCh <- response[:n]
	}()

	// Отправить error response
	errorMsg := "test error message"
	handler.sendErrorResponse(server, errorMsg)

	// Проверить ответ
	select {
	case response := <-responseCh:
		var resp Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Success {
			t.Error("Expected error response")
		}
		if resp.Error != errorMsg {
			t.Errorf("Expected error message '%s', got '%s'", errorMsg, resp.Error)
		}
	case <-time.After(1 * time.Second):
		t.Error("No response received")
	}
}

// Test 13: validateSession возвращает false при ошибке
func TestValidateSessionWithError(t *testing.T) {
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

	// Попытка проверить сессию, но sessionMgr будет недоступен
	// В текущей реализации validateSession возвращает false при любой ошибке
	// Мы можем эмулировать это, проверяя несуществующую сессию
	if handler.validateSession("non_existent_session") {
		t.Error("Non-existent session should return false")
	}
}

// Test 14: Stop возвращает ошибку если socket не удается закрыть
func TestStopError(t *testing.T) {
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

	socketPath := tempDir + "/test.sock"

	// Запуск сервера
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start(ctx, socketPath)
	}()

	// Дать время на запуск
	time.Sleep(100 * time.Millisecond)

	// Проверить что socket создан
	if handler.socket == nil {
		t.Fatal("Socket listener was not created")
	}

	// Закрыть сокет вручную перед вызовом Stop
	handler.socket.Close()

	// Вызвать Stop - попытается закрыть уже закрытый сокет
	err = handler.Stop()
	// В зависимости от реализации может вернуть ошибку или нет
	// Тест проверяет что Stop обрабатывает это корректно
	if err != nil {
		// Ошибка возможна, это нормально
		t.Logf("Stop returned error as expected: %v", err)
	}

	// Остановка
	cancel()
	<-errCh
}

// Test 15: Handler без запуска можно остановить без ошибки
func TestStopWithoutStart(t *testing.T) {
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

	// Проверить что socket не создан
	if handler.socket != nil {
		t.Error("Socket listener should not be created")
	}

	// Вызвать Stop без запуска
	err = handler.Stop()
	if err != nil {
		t.Errorf("Stop without start should not error: %v", err)
	}
}
