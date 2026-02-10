package builders

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceBuilder_NewWorkspaceBuilder(t *testing.T) {
	cfg := &config.Config{}
	log := createTestLogger(t)
	mb := bus.New(100, 10, log)

	builder := NewWorkspaceBuilder(cfg, log, mb)
	require.NotNil(t, builder)
	require.Equal(t, cfg, builder.config)
	require.Equal(t, log, builder.logger)
	require.Equal(t, mb, builder.messageBus)
}
