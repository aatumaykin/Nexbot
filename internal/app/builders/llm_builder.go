package builders

import (
	"fmt"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
)

type LLMBuilder struct {
	config *config.Config
	logger *logger.Logger
}

func NewLLMBuilder(cfg *config.Config, log *logger.Logger) *LLMBuilder {
	return &LLMBuilder{
		config: cfg,
		logger: log,
	}
}

func (b *LLMBuilder) Build() (llm.Provider, error) {
	switch b.config.Agent.Provider {
	case "zai":
		zaiConfig := llm.ZAIConfig{
			APIKey:         b.config.LLM.ZAI.APIKey,
			TimeoutSeconds: b.config.LLM.ZAI.TimeoutSeconds,
		}
		provider := llm.NewZAIProvider(zaiConfig, b.logger)
		b.logger.Info("LLM provider initialized", logger.Field{Key: "provider", Value: "zai"})
		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", b.config.Agent.Provider)
	}
}
