package workers

import (
	"context"
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	tests := []struct {
		name       string
		workers    int
		bufferSize int
		wantErr    bool
	}{
		{
			name:       "valid pool",
			workers:    3,
			bufferSize: 10,
			wantErr:    false,
		},
		{
			name:       "single worker",
			workers:    1,
			bufferSize: 5,
			wantErr:    false,
		},
		{
			name:       "many workers",
			workers:    100,
			bufferSize: 50,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workers, tt.bufferSize, log, messageBus)
			assert.NotNil(t, pool)
			assert.Equal(t, tt.workers, pool.WorkerCount())
			assert.NotNil(t, pool.Results())
		})
	}
}
