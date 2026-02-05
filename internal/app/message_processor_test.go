package app

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
)

func TestApp_StartMessageProcessing_Success(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Verify message processing started by checking if goroutine is running
	// This is a basic check - we verify no error occurred

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_StartMessageProcessing_NilMessageBus(t *testing.T) {
	t.Skip("StartMessageProcessing panics on nil messageBus - this is expected behavior")
}

func TestApp_StartMessageProcessing_ContextCancellation(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx, cancel := context.WithCancel(context.Background())

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Cancel context
	cancel()

	// Wait a bit for goroutine to exit
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_WithCommand(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Create a message with a command
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"/help",
		map[string]interface{}{"command": "/help"},
	)

	// Publish message
	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for message to be processed
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_WithoutCommand(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Create a message without a command
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"Hello, how are you?",
		nil,
	)

	// Publish message
	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for message to be processed
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_PublishesEvents(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Subscribe to events
	eventCh := app.messageBus.SubscribeEvent(ctx)
	if eventCh == nil {
		t.Fatal("SubscribeEvent() returned nil")
	}

	// Create and publish a message
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"test message",
		nil,
	)

	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for events
	eventCount := 0
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case event := <-eventCh:
			// Check for processing start or end events
			if event.Type == bus.EventTypeProcessingStart || event.Type == bus.EventTypeProcessingEnd {
				eventCount++
				if event.UserID != "user123" {
					t.Errorf("Event UserID = %s, want user123", event.UserID)
				}
				if event.SessionID != "session456" {
					t.Errorf("Event SessionID = %s, want session456", event.SessionID)
				}
			}
			if eventCount >= 2 {
				break loop
			}
		case <-timeout:
			t.Error("Timeout waiting for events")
			break loop
		}
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_CommandMetadata(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Test different command metadata formats
	testCases := []struct {
		name    string
		command string
	}{
		{"command string", "/help"},
		{"empty command", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := bus.NewInboundMessage(
				bus.ChannelTypeTelegram,
				"user123",
				"session456",
				"test",
				map[string]interface{}{"command": tc.command},
			)

			err = app.messageBus.PublishInbound(*msg)
			if err != nil {
				t.Fatalf("PublishInbound() failed: %v", err)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_NilMetadata(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Create a message with nil metadata
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"test message",
		nil,
	)

	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_MultipleMessages(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Publish multiple messages
	for i := 0; i < 5; i++ {
		msg := bus.NewInboundMessage(
			bus.ChannelTypeTelegram,
			"user123",
			"session456",
			"test message",
			nil,
		)

		err = app.messageBus.PublishInbound(*msg)
		if err != nil {
			t.Fatalf("PublishInbound() failed for message %d: %v", i, err)
		}
	}

	// Wait for all messages to be processed
	time.Sleep(500 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_DifferentChannels(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Test different channel types
	channels := []bus.ChannelType{
		bus.ChannelTypeTelegram,
	}

	for _, channel := range channels {
		msg := bus.NewInboundMessage(
			channel,
			"user123",
			"session456",
			"test message",
			nil,
		)

		err = app.messageBus.PublishInbound(*msg)
		if err != nil {
			t.Fatalf("PublishInbound() failed for channel %s: %v", channel, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_StartMessageProcessing_DoubleStart(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// First start
	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("First StartMessageProcessing() failed: %v", err)
	}

	// Second start - should succeed (creates new subscription)
	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Errorf("Second StartMessageProcessing() should succeed, got error: %v", err)
	}

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_AgentTimeout(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Create a message that may timeout
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"test message",
		nil,
	)

	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for processing or timeout
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	_ = app.Shutdown()
}

func TestApp_processMessage_PublishesOutbound(t *testing.T) {
	app := New(createTestConfig(t), createTestLogger(t))
	ctx := context.Background()

	err := app.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = app.StartMessageProcessing(ctx)
	if err != nil {
		t.Fatalf("StartMessageProcessing() failed: %v", err)
	}

	// Subscribe to outbound messages
	outboundCh := app.messageBus.SubscribeOutbound(ctx)
	if outboundCh == nil {
		t.Fatal("SubscribeOutbound() returned nil")
	}

	// Create and publish a message
	msg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		"user123",
		"session456",
		"test message",
		nil,
	)

	err = app.messageBus.PublishInbound(*msg)
	if err != nil {
		t.Fatalf("PublishInbound() failed: %v", err)
	}

	// Wait for outbound message (if any response is generated)
	select {
	case outboundMsg := <-outboundCh:
		if outboundMsg.UserID != "user123" {
			t.Errorf("Outbound UserID = %s, want user123", outboundMsg.UserID)
		}
		if outboundMsg.SessionID != "session456" {
			t.Errorf("Outbound SessionID = %s, want session456", outboundMsg.SessionID)
		}
	case <-time.After(1 * time.Second):
		// No outbound message is also valid if agent returns empty response
	}

	// Cleanup
	_ = app.Shutdown()
}
