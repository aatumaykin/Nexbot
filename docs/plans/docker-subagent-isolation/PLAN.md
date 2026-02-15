# План: Docker-изоляция сабагентов (v17 MVP)

## Обзор

Реализация изолированных сабагентов в Docker контейнерах для безопасной работы с внешними данными (web fetch, API calls). Защита от prompt injection атак через санитизацию.

## Архитектура

```
Оркестратор (основная система)
    │
    │ spawn (stdin JSON)
    ▼
Сабагент (Docker контейнер)
    │
    │ web_fetch, API calls
    ▼
Внешние источники
```

## Этапы

### Этап 0: Git
- Создать ветку `feature/docker-subagent-isolation`

### Этап 1: Prompt Injection Protection (высокий приоритет)
- `internal/subagent/sanitizer/sanitizer.go` — RE2 regex + Unicode NFKC normalize
- 5 критичных паттернов (role, direct, encoded, context, delimiter)
- Configurable RiskThreshold (default: 30)

### Этап 2: Динамические промпты сабагента (высокий приоритет)
- `internal/subagent/prompts/loader.go` — загрузка из файлов с fallback
- `internal/subagent/prompts/builder.go` — сборка system prompt
- `prompts/identity.md`, `prompts/security.md` — файлы промптов

### Этап 3: Передача секретов через stdin (высокий приоритет)
- `internal/security/secrets.go` — единый secrets store (общий для pool и subagent)
- In-memory secrets с TTL 5min + crypto/subtle.ZeroBytes

### Этап 4: Docker пакет (высокий приоритет)
- `internal/docker/client.go` — Docker API client
- `internal/docker/pool.go` — пул контейнеров + pool-level Circuit Breaker
- `internal/docker/rate_limiter.go` — Counter + window (простой)
- `internal/docker/execute.go` — выполнение задач
- `internal/docker/health.go` — health checks
- `internal/docker/retry.go` — retry с backoff
- `internal/docker/graceful.go` — graceful shutdown
- `internal/docker/metrics.go` — 6 базовых Prometheus metrics

### Этап 5: CLI сабагент (высокий приоритет)
- `cmd/subagent/main.go` — entry point
- `cmd/subagent/tools.go` — регистрация инструментов
- (использует `internal/security/secrets.go`)

### Этап 6: Fallback при Docker down (высокий приоритет)
- `internal/tools/spawn.go` — spawn tool с обработкой ошибок
- DockerUnavailableError с UserMessage()

### Этап 7: Dockerfile (высокий приоритет)
- `Dockerfile.subagent` — образ сабагента
- HEALTHCHECK через process check (pgrep)

### Этап 8: Обновление промптов оркестратора (средний приоритет)
- `docs/workspace/AGENTS.md` — секция Delegation to Subagents
- `docs/workspace/TOOLS.md` — описание spawn tool

### Этап 9: Конфигурация (средний приоритет)
- `internal/config/schema.go` — DockerConfig struct
- `config.example.toml` — пример конфигурации

### Этап 10: Интеграция (средний приоритет)
- `internal/app/builders/docker_builder.go` — builder для Docker pool
- `internal/app/builders/tools_builder.go` — регистрация spawn tool

### Этап 11: Makefile (низкий приоритет)
- `makefiles/docker.mk` — docker-build-subagent, docker-push-subagent

### Этап 12: Тесты (средний приоритет)
- Unit tests для sanitizer, circuit breaker, pool
- Integration tests

### Этап 13: CI + Push
- `make ci`
- `git push`

### Этап 14: Prometheus Alerting (низкий приоритет)
- 4 критичных alerting rules

## Ключевые решения (MVP)

| Параметр            | Решение                                         |
| ------------------- | ----------------------------------------------- |
| Режим Docker        | 1 контейнер (MVP)                               |
| Способ связи        | stdin/stdout JSON + versioning v1.0             |
| Secrets             | stdin JSON → []byte + TTL 5min + ZeroBytes      |
| LLM API Key         | stdin JSON (НЕ env)                             |
| Prompt injection    | RE2 + NFKC + 5 критичных паттернов              |
| Circuit Breaker     | Pool-level + Token-based Allow()                |
| Rate Limiting       | Counter + window (простой)                      |
| Pending Queue       | Mutex + maxPending backpressure                 |
| Resource limits     | memory=128m, cpu=0.5, pids=50                   |
| Healthcheck         | Process check (pgrep) + cached Inspect (TTL 5s) |
| Max response size   | 1MB limit                                       |
| Prometheus metrics  | 6 базовых (containers, requests, errors, latency) |
| Alerting rules      | 4 критичных (container_down, high_error, pool_exhausted, cb_open) |

## Зависимости

- `github.com/wasilibs/go-re2` — RE2 regex (linear time guarantee)
- `golang.org/x/text/unicode/norm` — Unicode NFKC normalization
- `github.com/docker/docker` — Docker SDK
- `github.com/prometheus/client_golang` — Prometheus metrics

## Риски

1. **Docker недоступен** → DockerUnavailableError + понятное сообщение
2. **Prompt injection** → RE2 санитизация + изоляция в контейнере
3. **OOM kills** → memory limits + monitoring + auto-recreate
4. **Cascading failures** → Pool-level Circuit Breaker + Rate Limiter

## Изменения v17 (MVP упрощение)

| Категория | Изменение |
| --------- | --------- |
| **Simplified** | 12 injection patterns → 5 критичных |
| **Simplified** | Per-container CB → Pool-level CB |
| **Simplified** | Token bucket rate limiter → Counter + window |
| **Simplified** | AES-256-GCM encryption → []byte + ZeroBytes |
| **Simplified** | 14 Prometheus metrics → 6 базовых |
| **Simplified** | 16 alerting rules → 4 критичных |
| **Removed** | Admin API endpoints (логи достаточно) |
| **Removed** | Unicode homoglyphs detection (v2) |
| **Fixed** | Единый secrets store в `internal/security/secrets.go` |
| **Fixed** | ReadonlyRootfs + tmpfs mount для /tmp |

## Отложено до v2

- AES-256-GCM шифрование
- Unicode homoglyphs detection
- Per-container Circuit Breaker
- Token bucket rate limiter
- Chaos testing
- Admin API

## Файлы плана

- `00-git.md` — Создание ветки
- `01-prompt-injection-protection.md` — Защита от prompt injection (5 паттернов)
- `02-dynamic-prompts.md` — Динамические промпты сабагента
- `03-secrets-stdin.md` — Передача секретов через stdin (без AES)
- `04-docker-package.md` — Docker пакет (pool-level CB, простой rate limiter)
- `05-cli-subagent.md` — CLI сабагент (без дублирования secrets)
- `06-fallback-docker-down.md` — Fallback при Docker down
- `07-dockerfile.md` — Dockerfile (tmpfs для /tmp)
- `08-update-orchestrator-prompts.md` — Обновление промптов оркестратора
- `09-configuration.md` — Конфигурация
- `10-integration.md` — Интеграция в приложение (без admin.go)
- `11-makefile.md` — Makefile команды
- `12-tests.md` — Тесты (unit + integration)
- `13-ci-push.md` — CI + Push
- `14-prometheus-alerts.md` — 4 критичных alerting rules
- `CHECKLIST.md` — Детальный чек-лист реализации
