package cron

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_JobExecution(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)
	defer func() {
		// Wait a bit for message to be received
		time.Sleep(100 * time.Millisecond)
	}()

	// Add a job that runs every 100ms
	job := Job{
		ID:       "test-job",
		Schedule: "*/1 * * * * *", // Every second
		Command:  "cron test command",
		UserID:   "cron-user",
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	// Wait for job to execute
	select {
	case msg := <-inboundCh:
		assert.Equal(t, ChannelTypeCron, msg.ChannelType)
		assert.Equal(t, "cron-user", msg.UserID)
		assert.Equal(t, "cron test command", msg.Content)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "test-job", msg.Metadata["cron_job_id"])
		assert.Equal(t, job.Schedule, msg.Metadata["cron_schedule"])
		assert.Equal(t, "test_value", msg.Metadata["test_key"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cron job to execute")
	}
}

func TestScheduler_JobExecutionWithMetadata(t *testing.T) {
	log := testLogger()
	msgBus := bus.New(100, log)

	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer stopMessageBus(msgBus)

	scheduler := NewScheduler(log, msgBus, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer stopScheduler(scheduler)

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)
	defer func() {
		time.Sleep(100 * time.Millisecond)
	}()

	job := Job{
		ID:       "metadata-job",
		Schedule: "*/1 * * * * *",
		Command:  "test",
		UserID:   "test-user",
		Metadata: map[string]string{
			"env":  "production",
			"team": "devops",
		},
	}

	_, err = scheduler.AddJob(job)
	require.NoError(t, err)

	select {
	case msg := <-inboundCh:
		assert.Equal(t, "production", msg.Metadata["env"])
		assert.Equal(t, "devops", msg.Metadata["team"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cron job")
	}
}
