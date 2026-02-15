# Этап 10: Интеграция (MVP)

## Цель

Интеграция Docker пула в основное приложение.

## Файлы

### 10.1 `internal/app/builders/docker_builder.go`

```go
package builders

import (
    "time"
    
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/docker"
    "github.com/aatumaykin/nexbot/internal/logger"
    "github.com/mitchellh/go-homedir"
)

func BuildDockerPool(cfg *config.Config, log *logger.Logger) (*docker.ContainerPool, error) {
    if !cfg.Docker.Enabled {
        log.Info("docker subagent disabled")
        return nil, nil
    }
    
    log.Info("initializing docker pool", 
        "image", cfg.Docker.ImageName,
        "containers", cfg.Docker.ContainerCount)
    
    skillsPath := cfg.Docker.SkillsPath
    if expanded, err := homedir.Expand(skillsPath); err == nil {
        skillsPath = expanded
    }
    
    poolCfg := docker.PoolConfig{
        ContainerCount:           cfg.Docker.ContainerCount,
        ImageName:                cfg.Docker.ImageName,
        ImageTag:                 cfg.Docker.ImageTag,
        ImageDigest:              cfg.Docker.ImageDigest,
        PullPolicy:               cfg.Docker.PullPolicy,
        TaskTimeout:              time.Duration(cfg.Docker.TaskTimeout) * time.Second,
        SkillsPath:               skillsPath,
        MemoryLimit:              cfg.Docker.MemoryLimit,
        CPULimit:                 cfg.Docker.CPULimit,
        PidsLimit:                cfg.Docker.PidsLimit,
        LLMAPIKeyEnv:             cfg.Docker.LLMAPIKeyEnv,
        MaxTasksPerMinute:        cfg.Docker.MaxTasksPerMinute,
        CircuitBreakerThreshold:  cfg.Docker.CircuitBreakerThreshold,
        CircuitBreakerTimeout:    time.Duration(cfg.Docker.CircuitBreakerTimeout) * time.Second,
        HealthCheckInterval:      time.Duration(cfg.Docker.HealthCheckInterval) * time.Second,
        MaxPendingPerContainer:   cfg.Docker.MaxPendingPerContainer,
        InspectTTL:               time.Duration(cfg.Docker.InspectTTL) * time.Second,
        SecurityOpt:              cfg.Docker.SecurityOpt,
        ReadonlyRootfs:           &cfg.Docker.ReadonlyRootfs,
    }
    
    if poolCfg.ContainerCount == 0 {
        poolCfg.ContainerCount = 1
    }
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
    if len(poolCfg.SecurityOpt) == 0 {
        poolCfg.SecurityOpt = []string{"no-new-privileges"}
    }
    
    pool, err := docker.NewContainerPool(poolCfg, log)
    if err != nil {
        return nil, err
    }
    
    return pool, nil
}
```

### 10.2 `internal/app/builders/tools_builder.go`

```go
package builders

import (
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/docker"
    "github.com/aatumaykin/nexbot/internal/tools"
    "github.com/aatumaykin/nexbot/internal/tools/file"
    "github.com/aatumaykin/nexbot/internal/tools/shell"
    "github.com/aatumaykin/nexbot/internal/logger"
    "github.com/aatumaykin/nexbot/internal/security"
)

func BuildTools(
    cfg *config.Config,
    log *logger.Logger,
    dockerPool *docker.ContainerPool,
    secretsStore *security.SecretsStore,
) (*tools.Registry, error) {
    registry := tools.NewRegistry()
    
    registry.Register(file.NewReadFileTool(cfg, log))
    registry.Register(file.NewWriteFileTool(cfg, log))
    registry.Register(file.NewListDirTool(cfg, log))
    registry.Register(file.NewDeleteFileTool(cfg, log))
    
    if cfg.Tools.Shell.Enabled {
        registry.Register(shell.NewShellTool(cfg, log))
    }
    
    if dockerPool != nil && cfg.Docker.Enabled {
        secretsFilter := docker.NewSecretsFilter(secretsStore)
        spawnTool := tools.NewSpawnTool(dockerPool, secretsFilter, cfg.Docker.LLMAPIKeyEnv)
        registry.Register(spawnTool)
        log.Info("spawn tool registered")
    }
    
    return registry, nil
}
```

### 10.3 `internal/app/app.go`

```go
package app

import (
    "context"
    "time"
    
    "github.com/aatumaykin/nexbot/internal/app/builders"
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/docker"
    "github.com/aatumaykin/nexbot/internal/logger"
    "github.com/aatumaykin/nexbot/internal/security"
    "github.com/aatumaykin/nexbot/internal/tools"
)

type App struct {
    cfg          *config.Config
    log          *logger.Logger
    dockerPool   *docker.ContainerPool
    secretsStore *security.SecretsStore
    tools        *tools.Registry
}

func New(cfg *config.Config, log *logger.Logger) (*App, error) {
    dockerPool, err := builders.BuildDockerPool(cfg, log)
    if err != nil {
        log.Warn("docker pool init failed, subagent disabled", "error", err)
    }
    
    secretsStore := security.NewSecretsStore(5 * time.Minute)
    
    registry, err := builders.BuildTools(cfg, log, dockerPool, secretsStore)
    if err != nil {
        return nil, err
    }
    
    return &App{
        cfg:          cfg,
        log:          log,
        dockerPool:   dockerPool,
        secretsStore: secretsStore,
        tools:        registry,
    }, nil
}

func (a *App) Start(ctx context.Context) error {
    if a.dockerPool != nil {
        if err := a.dockerPool.Start(ctx); err != nil {
            a.log.Warn("docker pool start failed", "error", err)
        } else {
            a.log.Info("docker pool started")
        }
    }
    
    return nil
}

func (a *App) Stop(ctx context.Context) error {
    if a.dockerPool != nil {
        shutdownCfg := docker.ShutdownConfig{
            Timeout:      30 * time.Second,
            DrainTimeout: 10 * time.Second,
            ForceAfter:   5 * time.Second,
        }
        
        if err := a.dockerPool.GracefulShutdown(ctx, shutdownCfg); err != nil {
            a.log.Error("docker pool shutdown error", "error", err)
        }
    }
    
    a.secretsStore.Clear()
    
    return nil
}
```

### 10.4 `cmd/nexbot/main.go`

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/aatumaykin/nexbot/internal/app"
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    cfg, err := config.Load("config.toml")
    if err != nil {
        panic(err)
    }
    
    log := logger.NewLogger(logger.Config{
        Level: cfg.Log.Level,
    })
    
    application, err := app.New(cfg, log)
    if err != nil {
        log.Error("failed to create application", "error", err)
        os.Exit(1)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Info("shutdown signal received")
        cancel()
    }()
    
    if err := application.Start(ctx); err != nil {
        log.Error("application error", "error", err)
        os.Exit(1)
    }
    
    <-ctx.Done()
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    
    if err := application.Stop(shutdownCtx); err != nil {
        log.Error("shutdown error", "error", err)
    }
    
    log.Info("application stopped")
}
```

## Ключевые решения (MVP)

1. **Optional Docker** — приложение работает без Docker
2. **Builder pattern** — изолированная сборка компонентов
3. **Graceful shutdown** — drain + timeout
4. **Единый SecretsStore** — используется везде
5. **Defaults в builder** — fallback значения
6. **Нет Admin API** — логи и метрики достаточно
