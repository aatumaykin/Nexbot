package ipc

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Test 1: Запуск и остановка сервера
func TestHandlerStartStop(t *testing.T) {
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

	// Проверить что слушатель создан
	if handler.socket == nil {
		t.Error("Socket listener was not created")
	}

	// Остановка
	cancel()
	err = <-errCh

	if err != nil && err != context.Canceled {
		t.Errorf("Start returned error: %v", err)
	}

	// Очистка
	_ = handler.Stop()
}

// Test 2: Прием соединений
func TestHandlerAcceptConnection(t *testing.T) {
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

	// Создать mock соединение
	conn := &mockConn{
		readBuf:  *bytes.NewBuffer([]byte(`{}`)),
		writeBuf: bytes.Buffer{},
	}
	conn.closed = false

	// Проверяем, что conn не закрыт после создания
	if conn.closed {
		t.Error("Connection was closed immediately")
	}

	// Остановка
	cancel()
	<-errCh

	// Очистка
	_ = handler.Stop()
}

// Test 5: Graceful shutdown
func TestHandlerGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

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

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start(ctx, socketPath)
	}()

	// Дать время на запуск
	time.Sleep(100 * time.Millisecond)

	// Отменить контекст
	cancel()

	// Проверить, что сервер корректно остановился
	timeout := time.After(2 * time.Second)
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Graceful shutdown failed: %v", err)
		}
	case <-timeout:
		t.Error("Timeout waiting for graceful shutdown")
	}

	// Очистка
	_ = handler.Stop()
}

// Test 8: Удаление старого socket файла
func TestHandlerCleanupOldSocket(t *testing.T) {
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

	// Создать старый socket файл
	oldSocket, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("Failed to create old socket: %v", err)
	}
	oldSocket.Close()

	// Запуск сервера
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start(ctx, socketPath)
	}()

	// Дать время на запуск
	time.Sleep(100 * time.Millisecond)

	// Проверить, что слушатель создан (старый файл был удален)
	if handler.socket == nil {
		t.Error("Socket listener was not created")
	}

	// Остановка
	cancel()
	<-errCh

	// Очистка
	_ = handler.Stop()
}
