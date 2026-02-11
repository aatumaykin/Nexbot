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

// Test 6: Обработка agent запроса
func TestHandleAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, 10, log)
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

	// Подписаться на inbound сообщения
	inboundCh := messageBus.SubscribeInbound(ctx)

	// Запуск сервера
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start(ctx, socketPath)
	}()

	// Дать время на запуск
	time.Sleep(100 * time.Millisecond)

	// Подготовить запрос
	request := Request{
		Type:      "agent",
		SessionID: "telegram:user123",
		Content:   "agent request",
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
	case msg := <-inboundCh:
		if msg.ChannelType != bus.ChannelTypeTelegram {
			t.Errorf("Unexpected channel type: %s", msg.ChannelType)
		}
		if msg.UserID != "user123" {
			t.Errorf("Unexpected user ID: %s", msg.UserID)
		}
		if msg.Content != "agent request" {
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

// Test 7: Обработка неизвестного типа запроса
func TestHandleUnknownRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, 10, log)
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

	// Подготовить запрос с неизвестным типом
	request := Request{
		Type:      "unknown_type",
		SessionID: "telegram:user123",
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
			t.Error("Expected error response for unknown request type")
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

// Test 12: Обработка agent запроса с остановленным message bus (ошибка публикации)
func TestHandleAgentPublishError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, 10, log)
	// Не запускаем message bus, чтобы PublishInbound вернул ошибку

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
		Type:      "agent",
		SessionID: "telegram:user123",
		Content:   "agent request",
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

// Test 13: Обработка запроса с невалидным JSON
func TestHandleConnectionInvalidJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, 10, log)
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

	// Создать невалидный JSON
	invalidJSON := []byte("{invalid json")

	// Создать pipe для имитации соединения
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Отправить невалидный запрос от клиента и получить ответ
	responseCh := make(chan []byte, 1)
	go func() {
		_, _ = client.Write(invalidJSON)
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
			t.Error("Expected error response for invalid JSON")
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
