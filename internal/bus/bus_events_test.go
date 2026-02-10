package bus

import (
	"context"
	"testing"
	"time"
)

func TestMessageBus_PublishEvent(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, 10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	bus2 := New(10, 10, log)
	err = bus2.PublishEvent(*event)
	if err != ErrNotStarted {
		t.Errorf("Expected ErrNotStarted, got %v", err)
	}
}

func TestMessageBus_SubscribeEvent(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, 10, log)
	ctx := context.Background()

	ch := bus.SubscribeEvent(ctx)
	if ch != nil {
		t.Error("SubscribeEvent() should return nil when not started")
	}

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	ch = bus.SubscribeEvent(ctx)
	if ch == nil {
		t.Fatal("SubscribeEvent() returned nil")
	}

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

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
}

func TestMessageBus_EventQueueFull(t *testing.T) {
	log := createTestLogger(t)
	bus := New(2, 10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	bus = New(1, 10, log)
	if err := bus.Start(ctx); err != nil {
		t.Fatal(err)
	}

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

func TestMessageBus_EventTypes(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, 10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	eventCh := bus.SubscribeEvent(ctx)

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

func TestMessageBus_EventInGracefulShutdown(t *testing.T) {
	log := createTestLogger(t)
	bus := New(10, 10, log)
	ctx := context.Background()

	err := bus.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	eventCh := bus.SubscribeEvent(ctx)

	event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
	err = bus.PublishEvent(*event)
	if err != nil {
		t.Fatalf("PublishEvent() failed: %v", err)
	}

	err = bus.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	select {
	case <-eventCh:
	default:
	}

	select {
	case _, ok := <-eventCh:
		if ok {
			t.Error("Event channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
	}
}
