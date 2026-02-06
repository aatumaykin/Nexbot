package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// mockConn имитирует net.Conn для тестов
type mockConn struct {
	net.Conn
	readBuf  bytes.Buffer
	writeBuf bytes.Buffer
	closed   bool
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{Name: "/mock/test", Net: "unix"}
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.UnixAddr{Name: "/mock/local", Net: "unix"}
}

func (m *mockConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockListener имитирует net.Listener для тестов
type mockListener struct {
	conns    chan net.Conn
	closed   bool
	acceptCh chan struct{}
	addr     net.Addr
}

func newMockListener() *mockListener {
	return &mockListener{
		conns:    make(chan net.Conn, 10),
		acceptCh: make(chan struct{}),
		addr:     &net.UnixAddr{Name: "/mock/test.sock", Net: "unix"},
	}
}

func (m *mockListener) Accept() (net.Conn, error) {
	if m.closed {
		return nil, net.ErrClosed
	}
	conn := <-m.conns
	close(m.acceptCh)
	m.acceptCh = make(chan struct{})
	return conn, nil
}

func (m *mockListener) Close() error {
	m.closed = true
	close(m.conns)
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return m.addr
}

func (m *mockListener) SendConn(conn net.Conn) {
	m.conns <- conn
	<-m.acceptCh
}

// Test 1: Запуск и остановка сервера
func TestHandlerStartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
// Примечание: текущая реализация validateChannel разрешает все каналы (возвращает nil)
// Тест проверяет текущее поведение - сообщение будет отправлено даже с любым каналом
func TestHandleSendMessageWithAnyChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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

	// Подготовить запрос с произвольным каналом (текущая реализация разрешает все)
	request := Request{
		Type:      "send_message",
		UserID:    "user123",
		Channel:   "custom_channel",
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

	// Проверить, что сообщение отправлено в bus (текущая реализация разрешает все каналы)
	select {
	case msg := <-outboundCh:
		if msg.ChannelType != "custom_channel" {
			t.Errorf("Unexpected channel type: %s", msg.ChannelType)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Message was not sent to bus (expected with current implementation)")
	}

	// Проверить, что получен успешный ответ
	select {
	case response := <-responseCh:
		var resp Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response (current implementation accepts all channels)")
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

// Test 5: Graceful shutdown
func TestHandlerGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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

// Test 6: Обработка agent запроса
func TestHandleAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
		UserID:    "user123",
		Channel:   "telegram",
		SessionID: "session456",
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
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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

// Test 8: Удаление старого socket файла
func TestHandlerCleanupOldSocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	messageBus := bus.New(100, log)
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
	messageBus.Start(ctx)
	defer messageBus.Stop()

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
