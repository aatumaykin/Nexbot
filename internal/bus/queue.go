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
	metrics    Metrics

	inboundSubscribers    map[int64]chan InboundMessage
	outboundSubscribers   map[int64]chan OutboundMessage
	eventSubscribers      map[int64]chan Event
	resultSubscribers     map[int64]chan MessageSendResult
	subscriberID          int64
	subscriberChannelSize int
}

// New creates a new MessageBus with the specified capacity for both queues
func New(capacity, subscriberChannelSize int, logger *logger.Logger) *MessageBus {
	return &MessageBus{
		logger:                logger,
		inboundCh:             make(chan InboundMessage, capacity),
		outboundCh:            make(chan OutboundMessage, capacity),
		eventCh:               make(chan Event, capacity),
		resultCh:              make(chan MessageSendResult, 500),
		tracker:               NewResultTracker(logger),
		inboundSubscribers:    make(map[int64]chan InboundMessage),
		outboundSubscribers:   make(map[int64]chan OutboundMessage),
		eventSubscribers:      make(map[int64]chan Event),
		resultSubscribers:     make(map[int64]chan MessageSendResult),
		subscriberID:          0,
		subscriberChannelSize: subscriberChannelSize,
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

// publishMessage publishes a message of any type to a channel.
// This is a generic function to eliminate code duplication between
// PublishInbound, PublishOutbound, and PublishEvent.
func publishMessage[T any](
	ctx context.Context,
	mu *sync.RWMutex,
	started bool,
	ch chan<- T,
	msg T,
	logDebug func(),
	logWarn func(),
) error {
	mu.RLock()
	defer mu.RUnlock()

	if !started {
		return ErrNotStarted
	}

	select {
	case ch <- msg:
		logDebug()
		return nil
	default:
		logWarn()
		return ErrQueueFull
	}
}

// PublishInbound publishes an inbound message to the queue
func (mb *MessageBus) PublishInbound(msg InboundMessage) error {
	return publishMessage(
		mb.ctx,
		&mb.mu,
		mb.started,
		mb.inboundCh,
		msg,
		func() {
			mb.logger.DebugCtx(mb.ctx, "inbound message published",
				logger.Field{Key: "session_id", Value: msg.SessionID},
				logger.Field{Key: "user_id", Value: msg.UserID})
		},
		func() {
			mb.logger.WarnCtx(mb.ctx, "inbound queue full",
				logger.Field{Key: "capacity", Value: cap(mb.inboundCh)})
		},
	)
}

// PublishOutbound publishes an outbound message to the queue
func (mb *MessageBus) PublishOutbound(msg OutboundMessage) error {
	return publishMessage(
		mb.ctx,
		&mb.mu,
		mb.started,
		mb.outboundCh,
		msg,
		func() {
			mb.logger.DebugCtx(mb.ctx, "outbound message published",
				logger.Field{Key: "session_id", Value: msg.SessionID},
				logger.Field{Key: "user_id", Value: msg.UserID})
		},
		func() {
			mb.logger.WarnCtx(mb.ctx, "outbound queue full",
				logger.Field{Key: "capacity", Value: cap(mb.outboundCh)})
		},
	)
}

// MessageInfo provides details about a message for logging
type MessageInfo interface {
	GetSessionID() string
	GetUserID() string
	GetType() string
}

// MessageInfo implementations
func (m InboundMessage) GetSessionID() string { return m.SessionID }
func (m InboundMessage) GetUserID() string    { return m.UserID }
func (m InboundMessage) GetType() string      { return "inbound" }

func (m OutboundMessage) GetSessionID() string { return m.SessionID }
func (m OutboundMessage) GetUserID() string    { return m.UserID }
func (m OutboundMessage) GetType() string      { return string(m.Type) }

func (e Event) GetSessionID() string { return e.SessionID }
func (e Event) GetUserID() string    { return e.UserID }
func (e Event) GetType() string      { return string(e.Type) }

func (m MessageSendResult) GetSessionID() string { return "" }
func (m MessageSendResult) GetUserID() string    { return "" }
func (m MessageSendResult) GetType() string      { return "send_result" }

// SubscribeInbound subscribes to inbound messages
func (mb *MessageBus) SubscribeInbound(ctx context.Context) <-chan InboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan InboundMessage, mb.subscriberChannelSize)
	mb.subscriberID++
	id := mb.subscriberID
	mb.inboundSubscribers[id] = ch
	mb.metrics.InboundSubscribersCount++

	mb.logger.DebugCtx(ctx, "inbound subscriber added",
		logger.Field{Key: "subscriber_id", Value: id},
		logger.Field{Key: "channel_capacity", Value: cap(ch)})

	return ch
}

// SubscribeOutbound subscribes to outbound messages
func (mb *MessageBus) SubscribeOutbound(ctx context.Context) <-chan OutboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan OutboundMessage, mb.subscriberChannelSize)
	mb.subscriberID++
	id := mb.subscriberID
	mb.outboundSubscribers[id] = ch
	mb.metrics.OutboundSubscribersCount++

	mb.logger.DebugCtx(ctx, "outbound subscriber added",
		logger.Field{Key: "subscriber_id", Value: id},
		logger.Field{Key: "channel_capacity", Value: cap(ch)})

	return ch
}

// distributeMessages distributes messages of any type to all subscribers
// This is a generic function to eliminate code duplication between
// distributeInbound, distributeOutbound, and distributeEvents
func distributeMessages[T any, M MessageInfo](
	ctx context.Context,
	log *logger.Logger,
	mu *sync.RWMutex,
	metrics *Metrics,
	ch <-chan T,
	getSubscribers func() map[int64]chan T,
	getMessageInfo func(T) M,
	logMsg string,
	onDropped func(),
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
			msgInfo := getMessageInfo(msg)
			for subID, subCh := range getSubscribers() {
				select {
				case subCh <- msg:
				default:
					// Subscriber channel is full, log with details and increment metrics
					log.WarnCtx(ctx, logMsg,
						logger.Field{Key: "subscriber_id", Value: subID},
						logger.Field{Key: "message_type", Value: msgInfo.GetType()},
						logger.Field{Key: "session_id", Value: msgInfo.GetSessionID()},
						logger.Field{Key: "user_id", Value: msgInfo.GetUserID()},
						logger.Field{Key: "channel_capacity", Value: cap(subCh)},
						logger.Field{Key: "channel_len", Value: len(subCh)})
					if onDropped != nil {
						onDropped()
					}
				}
			}
			mu.RUnlock()
		}
	}
}

// distributeInbound distributes inbound messages to all subscribers
func (mb *MessageBus) distributeInbound() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, &mb.metrics, mb.inboundCh, func() map[int64]chan InboundMessage {
		return mb.inboundSubscribers
	}, func(m InboundMessage) InboundMessage { return m }, "inbound subscriber channel full, skipping message", func() {
		mb.metrics.InboundMessagesDropped++
	})
}

// distributeOutbound distributes outbound messages to all subscribers
func (mb *MessageBus) distributeOutbound() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, &mb.metrics, mb.outboundCh, func() map[int64]chan OutboundMessage {
		return mb.outboundSubscribers
	}, func(m OutboundMessage) OutboundMessage { return m }, "outbound subscriber channel full, skipping message", func() {
		mb.metrics.OutboundMessagesDropped++
	})
}

// IsStarted returns true if the message bus is started
func (mb *MessageBus) IsStarted() bool {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return mb.started
}

// PublishEvent publishes a lifecycle event to the queue
func (mb *MessageBus) PublishEvent(event Event) error {
	return publishMessage(
		mb.ctx,
		&mb.mu,
		mb.started,
		mb.eventCh,
		event,
		func() {
			mb.logger.DebugCtx(mb.ctx, "event published",
				logger.Field{Key: "event_type", Value: event.Type},
				logger.Field{Key: "session_id", Value: event.SessionID},
				logger.Field{Key: "user_id", Value: event.UserID})
		},
		func() {
			mb.logger.WarnCtx(mb.ctx, "event queue full",
				logger.Field{Key: "capacity", Value: cap(mb.eventCh)})
		},
	)
}

// publishMessageWithTimeout publishes a message with custom timeout handling.
// This is used for PublishSendResult which has special force-publish logic.
func publishMessageWithTimeout[T any](
	ctx context.Context,
	mu *sync.RWMutex,
	started bool,
	ch chan<- T,
	msg T,
	onSuccess func(),
	onTimeout func(),
) error {
	mu.RLock()
	defer mu.RUnlock()

	if !started {
		return ErrNotStarted
	}

	select {
	case ch <- msg:
		onSuccess()
		return nil
	case <-time.After(100 * time.Millisecond):
		onTimeout()
		return nil
	}
}

// PublishSendResult публикует результат отправки сообщения
func (mb *MessageBus) PublishSendResult(result MessageSendResult) error {
	return publishMessageWithTimeout(
		mb.ctx,
		&mb.mu,
		mb.started,
		mb.resultCh,
		result,
		func() {
			mb.tracker.Complete(result.CorrelationID, result)
			mb.logger.DebugCtx(mb.ctx, "send result published",
				logger.Field{Key: "correlation_id", Value: result.CorrelationID},
				logger.Field{Key: "success", Value: result.Success})
		},
		func() {
			mb.logger.WarnCtx(mb.ctx, "result channel full, forcing publish",
				logger.Field{Key: "correlation_id", Value: result.CorrelationID},
				logger.Field{Key: "queue_size", Value: len(mb.resultCh)})
			mb.resultCh <- result
			mb.tracker.Complete(result.CorrelationID, result)
		},
	)
}

// SubscribeEvent subscribes to lifecycle events
func (mb *MessageBus) SubscribeEvent(ctx context.Context) <-chan Event {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan Event, mb.subscriberChannelSize)
	mb.subscriberID++
	id := mb.subscriberID
	mb.eventSubscribers[id] = ch
	mb.metrics.EventSubscribersCount++

	mb.logger.DebugCtx(ctx, "event subscriber added",
		logger.Field{Key: "subscriber_id", Value: id},
		logger.Field{Key: "channel_capacity", Value: cap(ch)})

	return ch
}

// distributeEvents distributes events to all subscribers
func (mb *MessageBus) distributeEvents() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, &mb.metrics, mb.eventCh, func() map[int64]chan Event {
		return mb.eventSubscribers
	}, func(e Event) Event { return e }, "event subscriber channel full, skipping event", func() {
		mb.metrics.EventsDropped++
	})
}

// SubscribeSendResults подписывается на результаты отправки
func (mb *MessageBus) SubscribeSendResults(ctx context.Context) <-chan MessageSendResult {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if !mb.started {
		return nil
	}

	ch := make(chan MessageSendResult, mb.subscriberChannelSize)
	mb.subscriberID++
	id := mb.subscriberID
	mb.resultSubscribers[id] = ch
	mb.metrics.ResultSubscribersCount++

	mb.logger.DebugCtx(ctx, "result subscriber added",
		logger.Field{Key: "subscriber_id", Value: id},
		logger.Field{Key: "channel_capacity", Value: cap(ch)})

	return ch
}

// GetResultTracker возвращает трекер результатов
func (mb *MessageBus) GetResultTracker() *ResultTracker {
	return mb.tracker
}

// GetMetrics возвращает метрики message bus
func (mb *MessageBus) GetMetrics() Metrics {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return mb.metrics
}

// distributeResults distributes send results to all subscribers
func (mb *MessageBus) distributeResults() {
	distributeMessages(mb.ctx, mb.logger, &mb.mu, &mb.metrics, mb.resultCh, func() map[int64]chan MessageSendResult {
		return mb.resultSubscribers
	}, func(r MessageSendResult) MessageSendResult { return r }, "result subscriber channel full, skipping result", func() {
		mb.metrics.ResultsDropped++
	})
}
