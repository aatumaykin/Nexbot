package ipc

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Test 3: Обработка send_message запроса
func TestHandleSendMessage(t *testing.T) {
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

	// Подписаться на outbound сообщения
	outboundCh := messageBus.SubscribeOutbound(ctx)

	// Запуск сервера
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start(ctx, socketPath)
	}()

	// Дать время на запуск
	time.Sleep(100 * time.Millisecond)

	// Подготовить запрос
	request := Request{
		Type:      "send_message",
		UserID:    "user123",
		Channel:   "telegram",
		SessionID: "session456",
		Content:   "test message",
	}

	reqData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Создать pipe для имитации соединения
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Отправить запрос от клиента
	go func() {
		_, _ = client.Write(reqData)
		// Читаем ответ
		response := make([]byte, 1024)
		_, _ = client.Read(response)
	}()

	// Обработать соединение в горутине
	go handler.handleConnection(server)

	// Дать время на обработку
	time.Sleep(100 * time.Millisecond)

	// Проверить, что сообщение отправлено в bus
	select {
	case msg := <-outboundCh:
		if msg.ChannelType != bus.ChannelTypeTelegram {
			t.Errorf("Unexpected channel type: %s", msg.ChannelType)
		}
		if msg.UserID != "user123" {
			t.Errorf("Unexpected user ID: %s", msg.UserID)
		}
		if msg.Content != "test message" {
			t.Errorf("Unexpected content: %s", msg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("No message sent to bus")
	}

	// Остановка
	cancel()
	<-errCh

	// Очистка
	_ = handler.Stop()
}

// Test 4: Валидация каналов через запрос с невалидным каналом
// Тест проверяет что недопустимый канал возвращает ошибку валидации
func TestHandleSendMessageWithInvalidChannel(t *testing.T) {
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

	// Подготовить запрос с недопустимым каналом
	request := Request{
		Type:      "send_message",
		UserID:    "user123",
		Channel:   "invalid_channel",
		SessionID: "session456",
		Content:   "test message",
	}

	reqData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Создать pipe для имитации соединения
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Отправить запрос от клиента и получить ответ
	responseCh := make(chan []byte, 1)
	go func() {
		_, _ = client.Write(reqData)
		// Читаем ответ
		response := make([]byte, 1024)
		n, _ := client.Read(response)
		responseCh <- response[:n]
	}()

	// Обработать соединение в горутине
	go handler.handleConnection(server)

	// Проверить, что получен ответ с ошибкой валидации
	select {
	case response := <-responseCh:
		var resp Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Success {
			t.Error("Expected error response for invalid channel")
		}
		if resp.Error == "" {
			t.Error("Expected error message for invalid channel")
		}
	case <-time.After(1 * time.Second):
		t.Error("No response received")
	}
}

// Test 12: Обработка send_message с остановленным message bus (ошибка публикации)
func TestHandleSendMessagePublishError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	// Не запускаем message bus, чтобы PublishOutbound вернул ошибку

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

	// Подготовить запрос
	request := Request{
		Type:      "send_message",
		UserID:    "user123",
		Channel:   "telegram",
		SessionID: "session456",
		Content:   "test message",
	}

	reqData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Создать pipe для имитации соединения
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Отправить запрос от клиента и получить ответ
	responseCh := make(chan []byte, 1)
	go func() {
		_, _ = client.Write(reqData)
		// Читаем ответ
		response := make([]byte, 1024)
		n, _ := client.Read(response)
		responseCh <- response[:n]
	}()

	// Обработать соединение в горутине
	go handler.handleConnection(server)

	// Дать время на обработку
	time.Sleep(100 * time.Millisecond)

	// Проверить, что получен ответ с ошибкой
	select {
	case response := <-responseCh:
		var resp Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Success {
			t.Error("Expected error response when message bus is stopped")
		}
	case <-time.After(1 * time.Second):
		t.Error("No response received")
	}

	// Остановка
	cancel()
	<-errCh

	// Очистка
	_ = handler.Stop()
}

// Test 13: Test NewHandler with invalid session directory
func TestNewHandlerInvalidSessionDir(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)

	// Попытка создать handler с недопустимым путем (например, пустая строка может вызвать ошибку)
	// Или с путем к файлу вместо директории
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	_, err = NewHandler(log, tempFile.Name(), messageBus)
	if err == nil {
		t.Error("Expected error when creating handler with file path instead of directory")
	}
}
