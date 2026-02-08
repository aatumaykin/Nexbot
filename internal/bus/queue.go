package bus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

var (
	// ErrQueueClosed is returned when attempting to publish to a closed queue.
	ErrQueueClosed = errors.New("queue is closed")

	// ErrQueueFull is returned when the message queue is at capacity.
	ErrQueueFull = errors.New("queue is full")

	// ErrAlreadyStarted is returned when attempting to start an already running message bus.
	ErrAlreadyStarted = errors.New("message bus is already started")

	// ErrNotStarted is returned when attempting to operate on a stopped message bus.
	ErrNotStarted = errors.New("message bus is not started")
)

// MessageBus represents an asynchronous message queue for inbound and outbound messages.
// It implements the publish-subscribe pattern, allowing multiple subscribers to receive
// copies of all published messages.
//
// The MessageBus provides:
//   - Thread-safe message publishing and subscribing
//   - Graceful shutdown with context cancellation
//   - Configurable queue capacity
//   - Support for multiple concurrent subscribers
//
// Example usage:
//
//	bus := bus.New(100, logger)
//	if err := bus.Start(ctx); err != nil {
//	    log.Fatal("Failed to start message bus", err)
//	}
//
//	// Subscribe to inbound messages
//	inboundCh := bus.SubscribeInbound(ctx)
//	go func() {
//	    for msg := range inboundCh {
//	        // Process message
//	    }
//	}()
//
//	// Publish an inbound message
//	msg := bus.NewInboundMessage(bus.ChannelTypeTelegram, "user123", "session456", "Hello", nil)
//	if err := bus.PublishInbound(*msg); err != nil {
//	    log.Error("Failed to publish message", err)
//	}
type MessageBus struct {
	mu      sync.RWMutex
	logger  *logger.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	inboundCh  chan InboundMessage
	outboundCh chan OutboundMessage
	eventCh    chan Event
	resultCh   chan MessageSendResult // для result tracking
	tracker    *ResultTracker

	inboundSubscribers  map[int64]chan InboundMessage
	outboundSubscribers map[int64]chan OutboundMessage
	eventSubscribers    map[int64]chan Event
	resultSubscribers   map[int64]chan MessageSendResult
	subscriberID        int64
}

// New creates a new MessageBus with the specified capacity for both queues
func New(capacity int, logger *logger.Logger) *MessageBus {
	return &MessageBus{
		logger:              logger,
		inboundCh:           make(chan InboundMessage, capacity),
		outboundCh:          make(chan OutboundMessage, capacity),
		eventCh:             make(chan Event, capacity),
		resultCh:            make(chan MessageSendResult, 500),
		tracker:             NewResultTracker(logger),
		inboundSubscribers:  make(map[int64]chan InboundMessage),
		outboundSubscribers: make(map[int64]chan OutboundMessage),
		eventSubscribers:    make(map[int64]chan Event),
		resultSubscribers:   make(map[int64]chan MessageSendResult),
		subscriberID:        0,
	}
}

// Start starts the message bus goroutines
func (mb *MessageBus) Start(ctx context.Context) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.started {
		return ErrAlreadyStarted
	}

	mb.ctx, mb.cancel = context.WithCancel(ctx)
	mb.started = true

	// Start goroutines to distribute messages to subscribers
	go mb.distributeInbound()
	go mb.distributeOutbound()
	go mb.distributeEvents()
	go mb.distributeResults()

	mb.logger.Info("message bus started", logger.Field{Key: "capacity", Value: cap(mb.inboundCh)})
	return nil
}

// Stop gracefully stops the message bus and closes all channels
func (mb *MessageBus) Stop() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return ErrNotStarted
	}

	mb.logger.Info("stopping message bus")

	// Cancel context
	if mb.cancel != nil {
		mb.cancel()
	}

	// Close subscribers
	for id, ch := range mb.inboundSubscribers {
		close(ch)
		delete(mb.inboundSubscribers, id)
	}

	for id, ch := range mb.outboundSubscribers {
		close(ch)
		delete(mb.outboundSubscribers, id)
	}

	for id, ch := range mb.eventSubscribers {
		close(ch)
		delete(mb.eventSubscribers, id)
	}

	for id, ch := range mb.resultSubscribers {
		close(ch)
		delete(mb.resultSubscribers, id)
	}

	// Close main channels
	close(mb.inboundCh)
	close(mb.outboundCh)
	close(mb.eventCh)
	close(mb.resultCh)

	mb.started = false

	mb.logger.Info("message bus stopped")
	return nil
}

// PublishInbound publishes an inbound message to the queue
func (mb *MessageBus) PublishInbound(msg InboundMessage) error {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if !mb.started {
		return ErrNotStarted
	}

	select {
	case mb.inboundCh <- msg:
		mb.logger.DebugCtx(mb.ctx, "inbound message published",
			logger.Field{Key: "session_id", Value: msg.SessionID},
			logger.Field{Key: "user_id", Value: msg.UserID})
		return nil
	default:
		mb.logger.WarnCtx(mb.ctx, "inbound queue full",
			logger.Field{Key: "capacity", Value: cap(mb.inboundCh)})
		return ErrQueueFull
	}
}

// PublishOutbound publishes an outbound message to the queue
func (mb *MessageBus) PublishOutbound(msg OutboundMessage) error {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if !mb.started {
		return ErrNotStarted
	}

	select {
	case mb.outboundCh <- msg:
		mb.logger.DebugCtx(mb.ctx, "outbound message published",
			logger.Field{Key: "session_id", Value: msg.SessionID},
			logger.Field{Key: "user_id", Value: msg.UserID})
		return nil
	default:
		mb.logger.WarnCtx(mb.ctx, "outbound queue full",
			logger.Field{Key: "capacity", Value: cap(mb.outboundCh)})
		return ErrQueueFull
	}
}

// SubscribeInbound subscribes to inbound messages
func (mb *MessageBus) SubscribeInbound(ctx context.Context) <-chan InboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan InboundMessage, 10)
	mb.subscriberID++
	id := mb.subscriberID
	mb.inboundSubscribers[id] = ch

	mb.logger.DebugCtx(ctx, "inbound subscriber added",
		logger.Field{Key: "subscriber_id", Value: id})

	return ch
}

// SubscribeOutbound subscribes to outbound messages
func (mb *MessageBus) SubscribeOutbound(ctx context.Context) <-chan OutboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan OutboundMessage, 10)
	mb.subscriberID++
	id := mb.subscriberID
	mb.outboundSubscribers[id] = ch

	mb.logger.DebugCtx(ctx, "outbound subscriber added",
		logger.Field{Key: "subscriber_id", Value: id})

	return ch
}

// distributeMessages distributes messages of any type to all subscribers
// This is a generic function to eliminate code duplication between
// distributeInbound, distributeOutbound, and distributeEvents
func distributeMessages[T any](
	ctx context.Context,
	logger *logger.Logger,
	mu *sync.RWMutex,
	ch <-chan T,
	getSubscribers func() map[int64]chan T,
	logMsg string,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			mu.RLock()
			for _, subCh := range getSubscribers() {
				select {
				case subCh <- msg:
				default:
					// Subscriber channel is full, skip
					logger.WarnCtx(ctx, logMsg)
				}
			}
			mu.RUnlock()
		}
	}
}

// distributeInbound distributes inbound messages to all subscribers
func (mb *MessageBus) distributeInbound() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, mb.inboundCh, func() map[int64]chan InboundMessage {
		return mb.inboundSubscribers
	}, "inbound subscriber channel full, skipping message")
}

// distributeOutbound distributes outbound messages to all subscribers
func (mb *MessageBus) distributeOutbound() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, mb.outboundCh, func() map[int64]chan OutboundMessage {
		return mb.outboundSubscribers
	}, "outbound subscriber channel full, skipping message")
}

// IsStarted returns true if the message bus is started
func (mb *MessageBus) IsStarted() bool {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return mb.started
}

// PublishEvent publishes a lifecycle event to the queue
func (mb *MessageBus) PublishEvent(event Event) error {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if !mb.started {
		return ErrNotStarted
	}

	select {
	case mb.eventCh <- event:
		mb.logger.DebugCtx(mb.ctx, "event published",
			logger.Field{Key: "event_type", Value: event.Type},
			logger.Field{Key: "session_id", Value: event.SessionID},
			logger.Field{Key: "user_id", Value: event.UserID})
		return nil
	default:
		mb.logger.WarnCtx(mb.ctx, "event queue full",
			logger.Field{Key: "capacity", Value: cap(mb.eventCh)})
		return ErrQueueFull
	}
}

// SubscribeEvent subscribes to lifecycle events
func (mb *MessageBus) SubscribeEvent(ctx context.Context) <-chan Event {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan Event, 10)
	mb.subscriberID++
	id := mb.subscriberID
	mb.eventSubscribers[id] = ch

	mb.logger.DebugCtx(ctx, "event subscriber added",
		logger.Field{Key: "subscriber_id", Value: id})

	return ch
}

// distributeEvents distributes events to all subscribers
func (mb *MessageBus) distributeEvents() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, mb.eventCh, func() map[int64]chan Event {
		return mb.eventSubscribers
	}, "event subscriber channel full, skipping event")
}

// PublishSendResult публикует результат отправки сообщения
func (mb *MessageBus) PublishSendResult(result MessageSendResult) error {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if !mb.started {
		return ErrNotStarted
	}

	select {
	case mb.resultCh <- result:
		mb.tracker.Complete(result.CorrelationID, result)
		mb.logger.DebugCtx(mb.ctx, "send result published",
			logger.Field{Key: "correlation_id", Value: result.CorrelationID},
			logger.Field{Key: "success", Value: result.Success})
		return nil
	case <-time.After(100 * time.Millisecond):
		mb.logger.WarnCtx(mb.ctx, "result channel full, forcing publish",
			logger.Field{Key: "correlation_id", Value: result.CorrelationID},
			logger.Field{Key: "queue_size", Value: len(mb.resultCh)})
		mb.resultCh <- result
		mb.tracker.Complete(result.CorrelationID, result)
		return nil
	}
}

// SubscribeSendResults подписывается на результаты отправки
func (mb *MessageBus) SubscribeSendResults(ctx context.Context) <-chan MessageSendResult {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan MessageSendResult, 10)
	mb.subscriberID++
	id := mb.subscriberID
	mb.resultSubscribers[id] = ch

	mb.logger.DebugCtx(ctx, "result subscriber added",
		logger.Field{Key: "subscriber_id", Value: id})

	return ch
}

// GetResultTracker возвращает трекер результатов
func (mb *MessageBus) GetResultTracker() *ResultTracker {
	return mb.tracker
}

// distributeResults distributes send results to all subscribers
func (mb *MessageBus) distributeResults() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, mb.resultCh, func() map[int64]chan MessageSendResult {
		return mb.resultSubscribers
	}, "result subscriber channel full, skipping result")
}
