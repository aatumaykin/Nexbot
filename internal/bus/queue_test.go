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
