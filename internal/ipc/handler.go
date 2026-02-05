package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/aatumaykin/nexbot/internal/agent/session"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// Request структура запроса от CLI
type Request struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
}

// Response структура ответа CLI
type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Handler обрабатывает IPC запросы
type Handler struct {
	logger     *logger.Logger
	socket     net.Listener
	ctx        context.Context
	sessionMgr *session.Manager
}

// NewHandler создаёт новый IPC Handler
func NewHandler(l *logger.Logger, sessionDir string) (*Handler, error) {
	// Create session manager
	sessionMgr, err := session.NewManager(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return &Handler{
		logger:     l,
		sessionMgr: sessionMgr,
	}, nil
}

// Start запускает IPC сервер
func (h *Handler) Start(ctx context.Context, socketPath string) error {
	h.ctx = ctx

	// Удаляем старый socket если существует
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	h.socket = listener

	// Запускаем обработку подключений в горутине
	go h.acceptConnections()

	h.logger.Info("IPC server started", logger.Field{Key: "socket", Value: socketPath})
	return nil
}

// acceptConnections принимает новые подключения
func (h *Handler) acceptConnections() {
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			conn, err := h.socket.Accept()
			if err != nil {
				select {
				case <-h.ctx.Done():
					return
				default:
					h.logger.Error("failed to accept connection", err)
				}
				continue
			}

			go h.handleConnection(conn)
		}
	}
}

// handleConnection обрабатывает одно подключение
func (h *Handler) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		h.sendErrorResponse(conn, fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	switch req.Type {
	case "send_message":
		h.handleSendMessage(&req, conn)
	case "agent":
		h.handleAgent(&req, conn)
	default:
		h.sendErrorResponse(conn, fmt.Sprintf("unknown request type: %s", req.Type))
	}
}

// handleSendMessage обрабатывает запрос отправки сообщения
func (h *Handler) handleSendMessage(req *Request, conn net.Conn) {
	// Валидация канала
	if err := h.validateChannel(req.Channel); err != nil {
		h.sendErrorResponse(conn, fmt.Sprintf("channel validation failed: %v", err))
		return
	}

	// TODO: отправка сообщения в канал
	h.logger.Info("send_message request",
		logger.Field{Key: "channel", Value: req.Channel},
		logger.Field{Key: "content", Value: req.Content})

	// Отправляем успешный ответ
	resp := Response{
		Success: true,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		h.logger.Error("failed to send response", err)
	}
}

// handleAgent обрабатывает запрос к агенту
func (h *Handler) handleAgent(req *Request, conn net.Conn) {
	// Валидация сессии
	if req.SessionID != "" {
		if !h.validateSession(req.SessionID) {
			h.sendErrorResponse(conn, fmt.Sprintf("session not found: %s", req.SessionID))
			return
		}
	}

	// TODO: обработка запроса к агенту
	h.logger.Info("agent request",
		logger.Field{Key: "session_id", Value: req.SessionID},
		logger.Field{Key: "user_id", Value: req.UserID})

	// Отправляем успешный ответ
	resp := Response{
		Success: true,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		h.logger.Error("failed to send response", err)
	}
}

// validateChannel проверяет валидность канала
func (h *Handler) validateChannel(channelType string) error {
	// TODO: реализовать валидацию канала
	return nil
}

// validateSession проверяет существование сессии
func (h *Handler) validateSession(sessionID string) bool {
	exists, err := h.sessionMgr.Exists(sessionID)
	if err != nil {
		h.logger.Error("failed to check session existence", err)
		return false
	}
	return exists
}

// sendErrorResponse отправляет ошибку клиенту
func (h *Handler) sendErrorResponse(conn net.Conn, errMsg string) {
	resp := Response{
		Success: false,
		Error:   errMsg,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		h.logger.Error("failed to send error response", err)
	}
}

// Stop останавливает IPC сервер
func (h *Handler) Stop() error {
	if h.socket != nil {
		if err := h.socket.Close(); err != nil {
			return fmt.Errorf("failed to close socket: %w", err)
		}
	}
	h.logger.Info("IPC server stopped")
	return nil
}
