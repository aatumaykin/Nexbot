# Чек-лист реализации (v17 MVP)

## Этап 0: Git

- [ ] Создать ветку `feature/docker-subagent-isolation`

## Этап 1: Prompt Injection Protection

- [ ] `internal/subagent/sanitizer/sanitizer.go` — 5 критичных паттернов
- [ ] PatternConfig с contextType и riskWeight
- [ ] SanitizerConfig с RiskThreshold (default 30)
- [ ] Validator struct вместо глобальных функций
- [ ] `go.mod`: добавить `github.com/wasilibs/go-re2`

## Этап 2: Динамические промпты сабагента

- [ ] `internal/subagent/prompts/loader.go` (fallback to defaults)
- [ ] `internal/subagent/prompts/builder.go`
- [ ] `prompts/identity.md`
- [ ] `prompts/security.md`

## Этап 3: Передача секретов через stdin

- [ ] `internal/security/secrets.go` — единый secrets store
- [ ] Secret.Value() []byte вместо String()
- [ ] crypto/subtle.ZeroBytes при очистке
- [ ] TTL 5min

## Этап 4: Docker пакет

### 4.1 `internal/docker/types.go`
- [ ] Container struct (без per-container CB)
- [ ] pendingEntry с mutex
- [ ] pendingCount int64
- [ ] ErrorCode constants

### 4.2 `internal/docker/errors.go`
- [ ] SubagentError с ErrorCode

### 4.3 `internal/docker/client.go`
- [ ] PullImage, CreateContainer, StartContainer, etc.
- [ ] DockerClientInterface для mocking

### 4.4 `internal/docker/pool.go`
- [ ] Pool-level Circuit Breaker
- [ ] Token-based Allow()
- [ ] bufferPool sync.Pool
- [ ] validator *sanitizer.Validator
- [ ] readResponses с timeout

### 4.5 `internal/docker/rate_limiter.go`
- [ ] Counter + window (простой)

### 4.6 `internal/docker/execute.go`
- [ ] MaxResponseSize = 1MB
- [ ] Write timeout 5s
- [ ] Scanner timeout 30s

### 4.7 `internal/docker/health.go`
- [ ] Health checks с auto-recreate

### 4.8 `internal/docker/retry.go`
- [ ] Exponential backoff + jitter

### 4.9 `internal/docker/graceful.go`
- [ ] GracefulShutdown с drain mode

### 4.10 `internal/docker/metrics.go`
- [ ] 6 базовых metrics (containers, requests, errors, latency)

## Этап 5: CLI сабагент

### 5.1 `cmd/subagent/main.go`
- [ ] Subagent struct
- [ ] MaxRequestSize = 1MB
- [ ] isCompatibleVersion()
- [ ] validator *sanitizer.Validator
- [ ] Использует `internal/security/secrets.go`

### 5.2 `cmd/subagent/tools.go`
- [ ] registerSubagentTools()

## Этап 6: Fallback при Docker down

- [ ] `internal/tools/spawn.go`
- [ ] DockerUnavailableError с UserMessage()

## Этап 7: Dockerfile

- [ ] `Dockerfile.subagent` с process HEALTHCHECK
- [ ] apk add procps
- [ ] tmpfs mount для /tmp (50MB)
- [ ] prompts/ директория

## Этап 8: Обновить промпты оркестратора

- [ ] `docs/workspace/AGENTS.md` — секция Delegation
- [ ] `docs/workspace/TOOLS.md` — описание spawn tool

## Этап 9: Конфигурация

- [ ] `internal/config/schema.go` — DockerConfig
- [ ] `config.example.toml`

## Этап 10: Интеграция

- [ ] `internal/app/builders/docker_builder.go`
- [ ] `internal/app/builders/tools_builder.go`

## Этап 11: Makefile

- [ ] `makefiles/docker.mk`

## Этап 12: Тесты

- [ ] `internal/subagent/sanitizer/*_test.go`
- [ ] `internal/docker/*_test.go`
- [ ] `internal/security/secrets_test.go`

## Этап 13: CI + Push

- [ ] `make ci`
- [ ] `git push`

## Этап 14: Prometheus Alerting

- [ ] 4 критичных alerting rules
