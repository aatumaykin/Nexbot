package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestMessageDroppingWithFullSubscriberChannel(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 2, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	_ = mb.SubscribeInbound(ctx)
	_ = mb.SubscribeInbound(ctx)

	inboundCh := mb.SubscribeInbound(ctx)
	if inboundCh == nil {
		t.Fatal("failed to subscribe")
	}

	for i := 0; i < 10; i++ {
		msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "test", nil)
		if err := mb.PublishInbound(*msg); err != nil {
			t.Errorf("failed to publish message %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	metrics := mb.GetMetrics()
	if metrics.InboundMessagesDropped == 0 {
		t.Error("expected messages to be dropped, but none were")
	}
}

func TestMetrics(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 10, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	metrics := mb.GetMetrics()
	if metrics.InboundSubscribersCount != 0 {
		t.Errorf("expected 0 inbound subscribers, got %d", metrics.InboundSubscribersCount)
	}

	mb.SubscribeInbound(ctx)
	metrics = mb.GetMetrics()
	if metrics.InboundSubscribersCount != 1 {
		t.Errorf("expected 1 inbound subscriber, got %d", metrics.InboundSubscribersCount)
	}

	metrics.Reset()
	if metrics.InboundSubscribersCount != 1 {
		t.Error("subscriber count should not be reset")
	}
	if metrics.InboundMessagesDropped != 0 {
		t.Error("dropped messages counter should be reset")
	}
}

func TestConcurrentMessageDropping(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 1, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	_ = mb.SubscribeInbound(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			msg := NewInboundMessage(ChannelTypeTelegram, "user123", "session456", "test", nil)
			_ = mb.PublishInbound(*msg)
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	metrics := mb.GetMetrics()
	if metrics.InboundMessagesDropped == 0 {
		t.Error("expected messages to be dropped during concurrent operations")
	}
}

func TestOutboundMessageDropping(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 2, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	_ = mb.SubscribeOutbound(ctx)
	_ = mb.SubscribeOutbound(ctx)

	_ = mb.SubscribeOutbound(ctx)

	for i := 0; i < 10; i++ {
		msg := NewOutboundMessage(ChannelTypeTelegram, "user123", "session456", "test", "corr123", FormatTypePlain, nil)
		if err := mb.PublishOutbound(*msg); err != nil {
			t.Errorf("failed to publish outbound message %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	metrics := mb.GetMetrics()
	if metrics.OutboundMessagesDropped == 0 {
		t.Error("expected outbound messages to be dropped, but none were")
	}
}

func TestEventDropping(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 2, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	_ = mb.SubscribeEvent(ctx)
	_ = mb.SubscribeEvent(ctx)

	_ = mb.SubscribeEvent(ctx)

	for i := 0; i < 10; i++ {
		event := NewProcessingStartEvent(ChannelTypeTelegram, "user123", "session456", nil)
		if err := mb.PublishEvent(*event); err != nil {
			t.Errorf("failed to publish event %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	metrics := mb.GetMetrics()
	if metrics.EventsDropped == 0 {
		t.Error("expected events to be dropped, but none were")
	}
}

func TestResultDropping(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 2, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	_ = mb.SubscribeSendResults(ctx)
	_ = mb.SubscribeSendResults(ctx)

	_ = mb.SubscribeSendResults(ctx)

	for i := 0; i < 10; i++ {
		result := MessageSendResult{
			CorrelationID: "corr123",
			ChannelType:   ChannelTypeTelegram,
			Success:       true,
			Timestamp:     time.Now(),
		}
		if err := mb.PublishSendResult(result); err != nil {
			t.Errorf("failed to publish result %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	metrics := mb.GetMetrics()
	if metrics.ResultsDropped == 0 {
		t.Error("expected results to be dropped, but none were")
	}
}

func TestSubscriberChannelSize(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatal(err)
	}

	mb := New(100, 5, log)
	if err := mb.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mb.Stop()

	inboundCh := mb.SubscribeInbound(ctx)
	if inboundCh == nil {
		t.Fatal("failed to subscribe")
	}

	if cap(inboundCh) != 5 {
		t.Errorf("expected channel capacity of 5, got %d", cap(inboundCh))
	}
}

func TestMetricsGetDroppedMetrics(t *testing.T) {
	metrics := Metrics{
		InboundMessagesDropped:  10,
		OutboundMessagesDropped: 20,
		EventsDropped:           5,
		ResultsDropped:          15,
	}

	dropped := metrics.GetDroppedMetrics()

	if dropped["inbound_messages_dropped"] != 10 {
		t.Errorf("expected inbound_messages_dropped to be 10, got %d", dropped["inbound_messages_dropped"])
	}
	if dropped["outbound_messages_dropped"] != 20 {
		t.Errorf("expected outbound_messages_dropped to be 20, got %d", dropped["outbound_messages_dropped"])
	}
	if dropped["events_dropped"] != 5 {
		t.Errorf("expected events_dropped to be 5, got %d", dropped["events_dropped"])
	}
	if dropped["results_dropped"] != 15 {
		t.Errorf("expected results_dropped to be 15, got %d", dropped["results_dropped"])
	}
}

func TestMessageInfoImplementations(t *testing.T) {
	inboundMsg := InboundMessage{SessionID: "session123", UserID: "user123"}
	if inboundMsg.GetSessionID() != "session123" {
		t.Error("inbound message GetSessionID failed")
	}
	if inboundMsg.GetUserID() != "user123" {
		t.Error("inbound message GetUserID failed")
	}
	if inboundMsg.GetType() != "inbound" {
		t.Error("inbound message GetType failed")
	}

	outboundMsg := OutboundMessage{SessionID: "session123", UserID: "user123", Type: MessageTypeText}
	if outboundMsg.GetSessionID() != "session123" {
		t.Error("outbound message GetSessionID failed")
	}
	if outboundMsg.GetUserID() != "user123" {
		t.Error("outbound message GetUserID failed")
	}
	if outboundMsg.GetType() != "text" {
		t.Error("outbound message GetType failed")
	}

	event := Event{SessionID: "session123", UserID: "user123", Type: EventTypeProcessingStart}
	if event.GetSessionID() != "session123" {
		t.Error("event GetSessionID failed")
	}
	if event.GetUserID() != "user123" {
		t.Error("event GetUserID failed")
	}
	if event.GetType() != "processing_start" {
		t.Error("event GetType failed")
	}

	result := MessageSendResult{}
	if result.GetSessionID() != "" {
		t.Error("result GetSessionID should return empty string")
	}
	if result.GetUserID() != "" {
		t.Error("result GetUserID should return empty string")
	}
	if result.GetType() != "send_result" {
		t.Error("result GetType failed")
	}
}
