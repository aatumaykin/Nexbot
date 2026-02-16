package builders

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/docker"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// dockerLoggerAdapter adapts logger.Logger to docker.Logger interface
type dockerLoggerAdapter struct {
	slog *slog.Logger
}

func newDockerLoggerAdapter(log *logger.Logger) *dockerLoggerAdapter {
	return &dockerLoggerAdapter{slog: log.StdLogger()}
}

func (a *dockerLoggerAdapter) Info(msg string, args ...interface{}) {
	a.slog.Info(msg, args...)
}

func (a *dockerLoggerAdapter) Warn(msg string, args ...interface{}) {
	a.slog.Warn(msg, args...)
}

func (a *dockerLoggerAdapter) Error(msg string, args ...interface{}) {
	a.slog.Error(msg, args...)
}

// expandHome expands ~ to home directory
func expandHome(path string) string {
	if len(path) >= 2 && path[0:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func BuildDockerPool(cfg *config.Config, log *logger.Logger) (*docker.ContainerPool, error) {
	if !cfg.Docker.Enabled {
		log.Info("docker subagent disabled")
		return nil, nil
	}

	log.Info("initializing docker pool")

	poolCfg := docker.PoolConfig{
		PullPolicy:              cfg.Docker.PullPolicy,
		TaskTimeout:             time.Duration(cfg.Docker.TaskTimeout) * time.Second,
		SkillsMountPath:         expandHome(cfg.Docker.SkillsMountPath),
		BinaryPath:              expandHome(cfg.Docker.BinaryPath),
		SubagentPromptsPath:     expandHome(cfg.Docker.SubagentPromptsPath),
		ConfigPath:              cfg.FilePath,
		MemoryLimit:             cfg.Docker.MemoryLimit,
		CPULimit:                cfg.Docker.CPULimit,
		PidsLimit:               cfg.Docker.PidsLimit,
		LLMAPIKeyEnv:            cfg.Docker.LLMAPIKeyEnv,
		Environment:             cfg.Docker.Environment,
		MaxTasksPerMinute:       cfg.Docker.MaxTasksPerMinute,
		CircuitBreakerThreshold: cfg.Docker.CircuitBreakerThreshold,
		CircuitBreakerTimeout:   time.Duration(cfg.Docker.CircuitBreakerTimeout) * time.Second,
		HealthCheckInterval:     time.Duration(cfg.Docker.HealthCheckInterval) * time.Second,
		MaxPendingPerContainer:  cfg.Docker.MaxPendingPerContainer,
		InspectTTL:              time.Duration(cfg.Docker.InspectTTL) * time.Second,
		SecretsTTL:              time.Duration(cfg.Docker.SecretsTTL) * time.Second,
		SecurityOpt:             cfg.Docker.SecurityOpt,
		ReadonlyRootfs:          &cfg.Docker.ReadonlyRootfs,
	}

	// Apply defaults if zero
	if poolCfg.MemoryLimit == "" {
		poolCfg.MemoryLimit = "128m"
	}
	if poolCfg.CPULimit == 0 {
		poolCfg.CPULimit = 0.5
	}
	if poolCfg.PidsLimit == 0 {
		poolCfg.PidsLimit = 50
	}
	if poolCfg.LLMAPIKeyEnv == "" {
		poolCfg.LLMAPIKeyEnv = "ZAI_API_KEY"
	}
	if poolCfg.MaxTasksPerMinute == 0 {
		poolCfg.MaxTasksPerMinute = 60
	}
	if poolCfg.CircuitBreakerThreshold == 0 {
		poolCfg.CircuitBreakerThreshold = 5
	}
	if poolCfg.CircuitBreakerTimeout == 0 {
		poolCfg.CircuitBreakerTimeout = 30 * time.Second
	}
	if poolCfg.HealthCheckInterval == 0 {
		poolCfg.HealthCheckInterval = 30 * time.Second
	}
	if poolCfg.MaxPendingPerContainer == 0 {
		poolCfg.MaxPendingPerContainer = 100
	}
	if poolCfg.InspectTTL == 0 {
		poolCfg.InspectTTL = 5 * time.Second
	}
	if poolCfg.SecretsTTL == 0 {
		poolCfg.SecretsTTL = 300 * time.Second
	}
	if len(poolCfg.SecurityOpt) == 0 {
		poolCfg.SecurityOpt = []string{"no-new-privileges"}
	}

	dockerLog := newDockerLoggerAdapter(log)
	pool, err := docker.NewContainerPool(poolCfg, dockerLog)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
