package bus

import (
	"context"
	"testing"

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

func TestMessageBus_GracefulShutdown(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	inboundCh := bus.SubscribeInbound(ctx)
	outboundCh := bus.SubscribeOutbound(ctx)

	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	if err := bus.PublishInbound(*msg); err != nil {
		t.Fatal(err)
	}

	outMsg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Response", "", nil)
	if err := bus.PublishOutbound(*outMsg); err != nil {
		t.Fatal(err)
	}

	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	select {
	case <-inboundCh:
	default:
	}

	select {
	case <-outboundCh:
	default:
	}

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

	cancel()

	select {
	case _, ok := <-inboundCh:
		if ok {
			t.Error("Inbound channel should be closed after context cancellation")
		}
	default:
	}

	select {
	case _, ok := <-outboundCh:
		if ok {
			t.Error("Outbound channel should be closed after context cancellation")
		}
	default:
	}

	bus.mu.Lock()
	bus.started = false
	bus.mu.Unlock()

	_ = bus.Stop()
}
