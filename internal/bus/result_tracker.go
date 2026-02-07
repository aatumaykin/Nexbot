package bus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// ResultTracker отслеживает результаты отправки сообщений
// Позволяет преобразовать асинхронную отправку в синхронное ожидание
type ResultTracker struct {
	mu      sync.Mutex
	pending map[string]chan MessageSendResult
	logger  *logger.Logger
}

// NewResultTracker создает новый ResultTracker
func NewResultTracker(logger *logger.Logger) *ResultTracker {
	rt := &ResultTracker{
		pending: make(map[string]chan MessageSendResult),
		logger:  logger,
	}

	// Запускаем cleanup для удаления зависших запросов
	go rt.cleanupLoop()

	return rt
}

// Register регистрирует запрос ожидания результата
func (rt *ResultTracker) Register(correlationID string) chan MessageSendResult {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	ch := make(chan MessageSendResult, 1)
	rt.pending[correlationID] = ch
	rt.logger.DebugCtx(context.Background(), "registered send result tracker",
		logger.Field{Key: "correlation_id", Value: correlationID},
		logger.Field{Key: "pending_count", Value: len(rt.pending)})

	return ch
}

// Wait ожидает результат отправки с таймаутом
func (rt *ResultTracker) Wait(ctx context.Context, correlationID string, timeout time.Duration) (*MessageSendResult, error) {
	rt.mu.Lock()
	ch, ok := rt.pending[correlationID]
	rt.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no pending request found for correlation_id: %s", correlationID)
	}

	// Ждем результат или таймаут
	select {
	case result := <-ch:
		rt.mu.Lock()
		delete(rt.pending, correlationID)
		rt.mu.Unlock()
		return &result, nil
	case <-time.After(timeout):
		rt.mu.Lock()
		delete(rt.pending, correlationID)
		rt.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for send result: %s", timeout)
	case <-ctx.Done():
		rt.mu.Lock()
		delete(rt.pending, correlationID)
		rt.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Complete завершает запрос с результатом
func (rt *ResultTracker) Complete(correlationID string, result MessageSendResult) {
	rt.mu.Lock()
	ch, ok := rt.pending[correlationID]
	rt.mu.Unlock()

	if !ok {
		rt.logger.DebugCtx(context.Background(), "no pending request for result",
			logger.Field{Key: "correlation_id", Value: correlationID})
		return
	}

	rt.logger.DebugCtx(context.Background(), "completing send result",
		logger.Field{Key: "correlation_id", Value: correlationID},
		logger.Field{Key: "success", Value: result.Success})

	// Неблокирующая отправка
	select {
	case ch <- result:
	default:
		rt.logger.WarnCtx(context.Background(), "failed to send result: channel blocked",
			logger.Field{Key: "correlation_id", Value: correlationID})
	}
}

// cleanupLoop периодически очищает старые запросы
func (rt *ResultTracker) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rt.mu.Lock()
		count := len(rt.pending)
		rt.mu.Unlock()

		if count > 0 {
			rt.logger.DebugCtx(context.Background(), "cleanup: pending results",
				logger.Field{Key: "count", Value: count})
		}
	}
}
