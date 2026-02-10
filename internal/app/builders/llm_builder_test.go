package builders

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/require"
)

func createTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
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

func TestLLMBuilder_NewLLMBuilder(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Provider: "zai",
		},
		LLM: config.LLMConfig{
			ZAI: config.ZAIConfig{
				APIKey:         "test-api-key",
				TimeoutSeconds: 30,
			},
		},
	}
	log := createTestLogger(t)

	builder := NewLLMBuilder(cfg, log)
	require.NotNil(t, builder)
	require.Equal(t, cfg, builder.config)
	require.Equal(t, log, builder.logger)
}

func TestLLMBuilder_Build(t *testing.T) {
	t.Run("ZAI provider", func(t *testing.T) {
		cfg := &config.Config{
			Agent: config.AgentConfig{
				Provider: "zai",
			},
			LLM: config.LLMConfig{
				ZAI: config.ZAIConfig{
					APIKey:         "test-api-key",
					TimeoutSeconds: 30,
				},
			},
		}
		log := createTestLogger(t)

		builder := NewLLMBuilder(cfg, log)
		provider, err := builder.Build()

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("Unsupported provider", func(t *testing.T) {
		cfg := &config.Config{
			Agent: config.AgentConfig{
				Provider: "unsupported",
			},
		}
		log := createTestLogger(t)

		builder := NewLLMBuilder(cfg, log)
		_, err := builder.Build()

		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported LLM provider")
	})
}
