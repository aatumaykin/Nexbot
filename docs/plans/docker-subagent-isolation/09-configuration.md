# Этап 9: Конфигурация

## Цель

Добавить конфигурацию Docker в основную конфигурацию приложения.

## Файлы

### 9.1 `internal/config/schema.go`

Добавить в структуру Config:

```go
type Config struct {
    // ... существующие поля ...
    
    Docker DockerConfig `toml:"docker"`
}

type DockerConfig struct {
    // Основные настройки
    Enabled        bool     `toml:"enabled"`
    ImageName      string   `toml:"image_name"`
    ImageTag       string   `toml:"image_tag"`
    ImageDigest    string   `toml:"image_digest"`
    PullPolicy     string   `toml:"pull_policy"`
    ContainerCount int      `toml:"container_count"`
    TaskTimeout    int      `toml:"task_timeout_seconds"`
    WorkspaceMount string   `toml:"workspace_mount"`
    SkillsPath     string   `toml:"skills_path"`
    Environment    []string `toml:"environment"`
    
    // Resource limits
    MemoryLimit string  `toml:"memory_limit"`
    CPULimit    float64 `toml:"cpu_limit"`
    PidsLimit   int64   `toml:"pids_limit"`
    
    // API и безопасность
    LLMAPIKeyEnv string `toml:"llm_api_key_env"`
    
    // Rate limiting и Circuit Breaker
    MaxTasksPerMinute       int `toml:"max_tasks_per_minute"`
    CircuitBreakerThreshold int `toml:"circuit_breaker_threshold"`
    CircuitBreakerTimeout   int `toml:"circuit_breaker_timeout_seconds"`
    
    // Health checks
    HealthCheckInterval     int   `toml:"health_check_interval_seconds"`
    MaxPendingPerContainer  int64 `toml:"max_pending_per_container"`
    InspectTTL              int   `toml:"inspect_ttl_seconds"`
    
    // Secrets
    SecretsTTL int `toml:"secrets_ttl_seconds"`
    
    // Container security
    SecurityOpt    []string `toml:"security_opt"`
    ReadonlyRootfs bool     `toml:"readonly_rootfs"`
}
```

### 9.2 `config.example.toml`

Добавить секцию:

```toml
# Docker Subagent Settings
# Сабагенты работают в изолированных контейнерах для безопасной
# работы с внешними данными (web, API calls)

[docker]
# Включить Docker-изоляцию сабагентов
enabled = true

# Docker образ
image_name = "nexbot/subagent"
image_tag = "latest"
# image_digest = "sha256:..."  # Для immutability
pull_policy = "if-not-present"  # always, if-not-present, never

# Пул контейнеров
container_count = 1  # MVP: один контейнер
task_timeout_seconds = 300

# Mounts
workspace_mount = "~/.nexbot"
skills_path = "~/.nexbot/skills"

# Resource limits (безопасность)
memory_limit = "128m"  # 128 MB
cpu_limit = 0.5        # 0.5 CPU cores
pids_limit = 50        # max processes

# Container security (MVP)
security_opt = ["no-new-privileges"]  # отключает privilege escalation
readonly_rootfs = true  # read-only filesystem + tmpfs для /tmp

# API Key (читается из environment variable)
llm_api_key_env = "ZAI_API_KEY"

# Rate limiting (защита от перегрузки)
max_tasks_per_minute = 60

# Circuit Breaker (защита от cascading failures)
circuit_breaker_threshold = 5   # ошибок до открытия
circuit_breaker_timeout_seconds = 30  # время до half-open

# Health checks
health_check_interval_seconds = 30
max_pending_per_container = 100
inspect_ttl_seconds = 5  # кэширование InspectContainer

# Secrets
secrets_ttl_seconds = 300  # 5 минут
```

### 9.3 `internal/config/defaults.go`

```go
package config

func DefaultDockerConfig() DockerConfig {
    return DockerConfig{
        Enabled:                 false,
        ImageName:               "nexbot/subagent",
        ImageTag:                "latest",
        PullPolicy:              "if-not-present",
        ContainerCount:          1,
        TaskTimeout:             300,
        WorkspaceMount:          "~/.nexbot",
        SkillsPath:              "~/.nexbot/skills",
        MemoryLimit:             "128m",
        CPULimit:                0.5,
        PidsLimit:               50,
        LLMAPIKeyEnv:            "ZAI_API_KEY",
        MaxTasksPerMinute:       60,
        CircuitBreakerThreshold: 5,
        CircuitBreakerTimeout:   30,
        HealthCheckInterval:     30,
        MaxPendingPerContainer:  100,
        InspectTTL:              5,
        SecretsTTL:              300,
        SecurityOpt:             []string{"no-new-privileges"},
        ReadonlyRootfs:          true,
    }
}
```

## Валидация

### `internal/config/validate.go`

```go
package config

import (
    "fmt"
    "strings"
)

func (c *DockerConfig) Validate() error {
    if !c.Enabled {
        return nil
    }
    
    if c.ImageName == "" {
        return fmt.Errorf("docker.image_name is required when docker.enabled=true")
    }
    
    validPolicies := map[string]bool{
        "always":        true,
        "if-not-present": true,
        "never":         true,
    }
    if !validPolicies[c.PullPolicy] {
        return fmt.Errorf("docker.pull_policy must be one of: always, if-not-present, never")
    }
    
    if c.ContainerCount < 1 {
        return fmt.Errorf("docker.container_count must be >= 1")
    }
    
    if c.TaskTimeout < 1 {
        return fmt.Errorf("docker.task_timeout_seconds must be >= 1")
    }
    
    if c.MemoryLimit != "" && !isValidMemoryLimit(c.MemoryLimit) {
        return fmt.Errorf("docker.memory_limit format invalid (e.g., 128m, 1g)")
    }
    
    if c.CPULimit <= 0 || c.CPULimit > 4 {
        return fmt.Errorf("docker.cpu_limit must be between 0 and 4")
    }
    
    if c.MaxTasksPerMinute < 1 {
        return fmt.Errorf("docker.max_tasks_per_minute must be >= 1")
    }
    
    if c.InspectTTL < 0 || c.InspectTTL > 60 {
        return fmt.Errorf("docker.inspect_ttl_seconds must be between 0 and 60 (got %d)", c.InspectTTL)
    }
    
    if c.CircuitBreakerTimeout < 5 || c.CircuitBreakerTimeout > 300 {
        return fmt.Errorf("docker.circuit_breaker_timeout_seconds must be between 5 and 300 (got %d)", c.CircuitBreakerTimeout)
    }
    
    return nil
}

func isValidMemoryLimit(s string) bool {
    s = strings.ToLower(s)
    suffixes := []string{"k", "m", "g"}
    for _, suffix := range suffixes {
        if strings.HasSuffix(s, suffix) {
            num := strings.TrimSuffix(s, suffix)
            for _, c := range num {
                if c < '0' || c > '9' {
                    return false
                }
            }
            return true
        }
    }
    return false
}
```

## Ключевые решения

1. **enabled=false по умолчанию** — Docker опционален
2. **Минимальные defaults** — один контейнер, базовые лимиты
3. **Валидация** — проверка корректности конфигурации
4. **Environment variable для API key** — безопасность
5. **Pull policy** — гибкое управление обновлениями образа
