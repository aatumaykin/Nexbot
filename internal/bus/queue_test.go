package bus

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func createTestLogger(t *testing.T) *logger.Logger {
	cfg := logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return log
}

func TestNew(t *testing.T) {
	capacity := 100
	log := createTestLogger(t)

	bus := New(capacity, log)

	if bus == nil {
		t.Fatal("New() returned nil")
	}

	if bus.IsStarted() {
		t.Error("New() returned a started bus")
	}
}

func TestMessageBus_Start(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)

	ctx := context.Background()
	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !bus.IsStarted() {
		t.Error("Start() did not set started flag")
	}

	// Test double start
	err = bus.Start(ctx)
	if err != ErrAlreadyStarted {
		t.Errorf("Expected ErrAlreadyStarted, got %v", err)
	}

	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestMessageBus_Stop(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)

	ctx := context.Background()
	err := bus.Stop()
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}

	err = bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if bus.IsStarted() {
		t.Error("Stop() did not clear started flag")
	}
}

func TestMessageBus_PublishInbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(2, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Test publish when not started
	bus2 := New(10, log)
	err = bus2.PublishInbound(*msg)
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}
}

func TestMessageBus_PublishOutbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(2, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishOutbound(*msg)
	if err != nil {
		t.Fatalf("PublishOutbound() failed: %v", err)
	}

	// Test queue full
	bus = New(1, log)
	bus.Start(ctx)
	defer bus.Stop()

	msg1 := NewOutboundMessage(ChannelTypeTelegram, "user1", "session1", "Hello1", nil)
	msg2 := NewOutboundMessage(ChannelTypeTelegram, "user2", "session2", "Hello2", nil)

	err = bus.PublishOutbound(*msg1)
	if err != nil {
		t.Fatalf("PublishOutbound() failed for first message: %v", err)
	}

	err = bus.PublishOutbound(*msg2)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}

	// Test publish when not started
	bus2 := New(10, log)
	err = bus2.PublishOutbound(*msg)
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}
}

func TestMessageBus_SubscribeInbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	// Test subscribe when not started
	ch := bus.SubscribeInbound(ctx)
	if ch != nil {
		t.Error("SubscribeInbound() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	ch = bus.SubscribeInbound(ctx)
	if ch == nil {
		t.Fatal("SubscribeInbound() returned nil")
	}

	// Publish a message
	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Receive the message
	select {
	case receivedMsg := <-ch:
		if receivedMsg.UserID != msg.UserID {
			t.Errorf("Expected UserID %s, got %s", msg.UserID, receivedMsg.UserID)
		}
		if receivedMsg.SessionID != msg.SessionID {
			t.Errorf("Expected SessionID %s, got %s", msg.SessionID, receivedMsg.SessionID)
		}
		if receivedMsg.Content != msg.Content {
			t.Errorf("Expected Content %s, got %s", msg.Content, receivedMsg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}

	// Test multiple subscribers
	ch2 := bus.SubscribeInbound(ctx)
	err = bus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	receivedCount := 0
	select {
	case <-ch:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-ch2:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	if receivedCount != 2 {
		t.Errorf("Expected 2 messages received, got %d", receivedCount)
	}
}

func TestMessageBus_SubscribeOutbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	// Test subscribe when not started
	ch := bus.SubscribeOutbound(ctx)
	if ch != nil {
		t.Error("SubscribeOutbound() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	ch = bus.SubscribeOutbound(ctx)
	if ch == nil {
		t.Fatal("SubscribeOutbound() returned nil")
	}

	// Publish a message
	msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishOutbound(*msg)
	if err != nil {
		t.Fatalf("PublishOutbound() failed: %v", err)
	}

	// Receive the message
	select {
	case receivedMsg := <-ch:
		if receivedMsg.UserID != msg.UserID {
			t.Errorf("Expected UserID %s, got %s", msg.UserID, receivedMsg.UserID)
		}
		if receivedMsg.SessionID != msg.SessionID {
			t.Errorf("Expected SessionID %s, got %s", msg.SessionID, receivedMsg.SessionID)
		}
		if receivedMsg.Content != msg.Content {
			t.Errorf("Expected Content %s, got %s", msg.Content, receivedMsg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}

	// Test multiple subscribers
	ch2 := bus.SubscribeOutbound(ctx)
	err = bus.PublishOutbound(*msg)
	if err != nil {
		t.Fatalf("PublishOutbound() failed: %v", err)
	}

	receivedCount := 0
	select {
	case <-ch:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-ch2:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	if receivedCount != 2 {
		t.Errorf("Expected 2 messages received, got %d", receivedCount)
	}
}

func TestMessageBus_GracefulShutdown(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Subscribe to channels
	inboundCh := bus.SubscribeInbound(ctx)
	outboundCh := bus.SubscribeOutbound(ctx)

	// Publish some messages
	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	bus.PublishInbound(*msg)

	outMsg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Response", nil)
	bus.PublishOutbound(*outMsg)

	// Stop the bus
	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify channels are closed by draining remaining messages
	// The published message might still be in the subscriber channel
	select {
	case <-inboundCh:
		// Drain the message
	default:
	}

	select {
	case <-outboundCh:
		// Drain the message
	default:
	}

	// Now verify channels are closed
	select {
	case _, ok := <-inboundCh:
		if ok {
			t.Error("Inbound channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case _, ok := <-outboundCh:
		if ok {
			t.Error("Outbound channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
	}

	// Verify we can't publish after stop
	err = bus.PublishInbound(*msg)
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}
}

func TestMessageBus_ContextCancellation(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx, cancel := context.WithCancel(context.Background())

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	inboundCh := bus.SubscribeInbound(ctx)
	outboundCh := bus.SubscribeOutbound(ctx)

	// Cancel the context
	cancel()

	// Wait a bit for goroutines to exit
	time.Sleep(100 * time.Millisecond)

	// Verify channels are closed
	select {
	case _, ok := <-inboundCh:
		if ok {
			t.Error("Inbound channel should be closed after context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case _, ok := <-outboundCh:
		if ok {
			t.Error("Outbound channel should be closed after context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
	}

	// Reset started flag
	bus.mu.Lock()
	bus.started = false
	bus.mu.Unlock()

	_ = bus.Stop() // Clean up
}

// TestMessageBus_PublishEvent tests publishing events
func TestMessageBus_PublishEvent(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	// Test publish when not started
	bus2 := New(10, log)
	err = bus2.PublishEvent(*event)
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}
}

// TestMessageBus_SubscribeEvent tests subscribing to events
func TestMessageBus_SubscribeEvent(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	// Test subscribe when not started
	ch := bus.SubscribeEvent(ctx)
	if ch != nil {
		t.Error("SubscribeEvent() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	ch = bus.SubscribeEvent(ctx)
	if ch == nil {
		t.Fatal("SubscribeEvent() returned nil")
	}

	// Publish an event
	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	// Receive event
	select {
	case receivedEvent := <-ch:
		if receivedEvent.Type != event.Type {
			t.Errorf("Expected Type %s, got %s", event.Type, receivedEvent.Type)
		}
		if receivedEvent.ChannelType != event.ChannelType {
			t.Errorf("Expected ChannelType %s, got %s", event.ChannelType, receivedEvent.ChannelType)
		}
		if receivedEvent.UserID != event.UserID {
			t.Errorf("Expected UserID %s, got %s", event.UserID, receivedEvent.UserID)
		}
		if receivedEvent.SessionID != event.SessionID {
			t.Errorf("Expected SessionID %s, got %s", event.SessionID, receivedEvent.SessionID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}

	// Test multiple subscribers
	ch2 := bus.SubscribeEvent(ctx)
	event2 := NewProcessingEndEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event2)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	receivedCount := 0
	select {
	case <-ch:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-ch2:
		receivedCount++
	case <-time.After(100 * time.Millisecond):
	}

	if receivedCount != 2 {
		t.Errorf("Expected 2 events received, got %d", receivedCount)
	}
}

// TestMessageBus_EventQueueFull tests event queue full scenario
func TestMessageBus_EventQueueFull(t *testing.T) {
	log := createTestLogger(t)
	bus := New(1, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	// Subscribe to events (with a tiny channel that will block)
	_ = bus.SubscribeEvent(ctx)

	event1 := NewProcessingStartEvent(ChannelTypeTelegram, "user1", "session1", nil)
	event2 := NewProcessingEndEvent(ChannelTypeTelegram, "user2", "session2", nil)

	err = bus.PublishEvent(*event1)
	if err != nil {
		t.Fatalf("PublishEvent() failed for first event: %v", err)
	}

	err = bus.PublishEvent(*event2)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

// TestMessageBus_EventTypes tests different event types
func TestMessageBus_EventTypes(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer bus.Stop()

	eventCh := bus.SubscribeEvent(ctx)

	// Publish processing start event
	startEvent := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*startEvent)
	if err != nil {
		t.Fatalf("PublishEvent() failed for start event: %v", err)
	}

	select {
	case received := <-eventCh:
		if received.Type != EventTypeProcessingStart {
			t.Errorf("Expected event type %s, got %s", EventTypeProcessingStart, received.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for start event")
	}

	// Publish processing end event
	endEvent := NewProcessingEndEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*endEvent)
	if err != nil {
		t.Fatalf("PublishEvent() failed for end event: %v", err)
	}

	select {
	case received := <-eventCh:
		if received.Type != EventTypeProcessingEnd {
			t.Errorf("Expected event type %s, got %s", EventTypeProcessingEnd, received.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for end event")
	}
}

// TestMessageBus_EventInGracefulShutdown tests event channels are closed on graceful shutdown
func TestMessageBus_EventInGracefulShutdown(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	eventCh := bus.SubscribeEvent(ctx)

	// Publish an event
	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	// Stop the bus
	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Drain the published event
	select {
	case <-eventCh:
		// Drain event
	default:
	}

	// Verify channel is closed
	select {
	case _, ok := <-eventCh:
		if ok {
			t.Error("Event channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
	}
}
