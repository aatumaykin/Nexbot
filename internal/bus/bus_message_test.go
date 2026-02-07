package bus

import (
	"context"
	"testing"
	"time"
)

func TestMessageBus_PublishInbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(2, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

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

	msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", "", nil)
	err = bus.PublishOutbound(*msg)
	if err != nil {
		t.Fatalf("PublishOutbound() failed: %v", err)
	}

	bus = New(1, log)
	if err := bus.Start(ctx); err != nil {
		t.Fatal(err)
	}

	msg1 := NewOutboundMessage(ChannelTypeTelegram, "user1", "session1", "Hello1", "", nil)
	msg2 := NewOutboundMessage(ChannelTypeTelegram, "user2", "session2", "Hello2", "", nil)

	err = bus.PublishOutbound(*msg1)
	if err != nil {
		t.Fatalf("PublishOutbound() failed for first message: %v", err)
	}

	err = bus.PublishOutbound(*msg2)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}

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

	ch := bus.SubscribeInbound(ctx)
	if ch != nil {
		t.Error("SubscribeInbound() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	ch = bus.SubscribeInbound(ctx)
	if ch == nil {
		t.Fatal("SubscribeInbound() returned nil")
	}

	msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", nil)
	err = bus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

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
}

func TestMessageBus_SubscribeOutbound(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, log)
	ctx := context.Background()

	ch := bus.SubscribeOutbound(ctx)
	if ch != nil {
		t.Error("SubscribeOutbound() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	ch = bus.SubscribeOutbound(ctx)
	if ch == nil {
		t.Fatal("SubscribeOutbound() returned nil")
	}

	msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "Hello", "", nil)
	err = bus.PublishOutbound(*msg)
	if err != nil {
		t.Fatalf("PublishOutbound() failed: %v", err)
	}

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
}
