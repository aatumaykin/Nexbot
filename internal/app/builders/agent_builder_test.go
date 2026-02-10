package builders

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestAgentBuilder_NewAgentBuilder(t *testing.T) {
	cfg := &config.Config{}
	log := createTestLogger(t)
	provider := llm.NewMockProvider(llm.MockConfig{})
	ws := workspace.New(config.WorkspaceConfig{})

	builder := NewAgentBuilder(cfg, log, provider, ws)
	require.NotNil(t, builder)
	require.Equal(t, cfg, builder.config)
	require.Equal(t, log, builder.logger)
	require.Equal(t, provider, builder.provider)
	require.Equal(t, ws, builder.workspace)
}

func TestAgentBuilder_BuildLoop(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Provider: "zai",
			Model:    "test-model",
		},
		Cron: config.CronConfig{
			Timezone: "UTC",
		},
	}
	log := createTestLogger(t)
	provider := llm.NewMockProvider(llm.MockConfig{})
	ws := workspace.New(config.WorkspaceConfig{})

	builder := NewAgentBuilder(cfg, log, provider, ws)
	agentLoop, err := builder.BuildLoop()

	require.Error(t, err)
	require.Nil(t, agentLoop)
}

func TestAgentBuilder_BuildSubagentManager(t *testing.T) {
	t.Run("Subagent disabled", func(t *testing.T) {
		cfg := &config.Config{
			Subagent: config.SubagentConfig{
				Enabled: false,
			},
		}
		log := createTestLogger(t)
		provider := llm.NewMockProvider(llm.MockConfig{})
		ws := workspace.New(config.WorkspaceConfig{})
		agentLoop := &loop.Loop{}

		builder := NewAgentBuilder(cfg, log, provider, ws)
		manager, spawnFunc, err := builder.BuildSubagentManager(agentLoop)

		require.NoError(t, err)
		require.Nil(t, manager)
		require.Nil(t, spawnFunc)
	})
}
