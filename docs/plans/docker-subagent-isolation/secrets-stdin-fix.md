# План: Устранение утечки секретов через env контейнеров

## Проблема

Секреты (`ZAI_API_KEY` и другие) могут передаваться через переменные окружения контейнера, что небезопасно — они видны через `docker inspect`.

**Текущее состояние:**
```bash
docker inspect <container> | grep -A 20 "Env"
# Видно: ZAI_API_KEY=xxx
```

---

## Анализ путей передачи секретов

| Путь                                        | Небезопасно? | Действие        | Обоснование                                      |
| ------------------------------------------- | ------------ | --------------- | ------------------------------------------------ |
| `LLMAPIKeyEnv` → env контейнера (client.go) | ✅ ДА         | **Удалить**     | Видно в `docker inspect`                         |
| `LLMAPIKeyEnv` → stdin (docker_spawn.go)    | ❌ НЕТ        | Оставить        | Секрет читается на хосте, передается через stdin |
| `Environment` → env контейнера (client.go)  | ✅ ДА         | **Удалить**     | Потенциальная утечка через конфиг                |
| `SubagentRequest.LLMAPIKey` → stdin         | ❌ НЕТ        | Основной путь   | Безопасно, не видно в inspect                    |
| `SubagentRequest.Secrets` → stdin           | ❌ НЕТ        | Основной путь   | Безопасно, не видно в inspect                    |

**Подтверждение:** Subagent использует ТОЛЬКО stdin для получения секретов (проверено в `cmd/nexbot/subagent.go:215-221`):
```go
if req.LLMAPIKey != "" && subagent.llmProvider == nil {
    if err := subagent.InitLLM(req.LLMAPIKey); err != nil {
        // ...
    }
}
```

---

## Изменения

### 1. [ОБЯЗАТЕЛЬНО] Удалить передачу LLMAPIKeyEnv через env контейнера

**Файл: `internal/docker/client.go`**

Удалить блок (строки 134-139):
```go
// УДАЛИТЬ:
if cfg.LLMAPIKeyEnv != "" {
    apiKeyValue := os.Getenv(cfg.LLMAPIKeyEnv)
    if apiKeyValue != "" {
        env = append(env, fmt.Sprintf("%s=%s", cfg.LLMAPIKeyEnv, apiKeyValue))
    }
}
```

---

### 2. [ОБЯЗАТЕЛЬНО] Удалить передачу Environment через env контейнера

**Файл: `internal/docker/client.go`**

Удалить блок (строки 142-151):
```go
// УДАЛИТЬ:
for _, envVar := range cfg.Environment {
    parts := strings.SplitN(envVar, "=", 2)
    if len(parts) == 1 {
        if value := os.Getenv(parts[0]); value != "" {
            env = append(env, fmt.Sprintf("%s=%s", parts[0], value))
        }
    } else {
        env = append(env, envVar)
    }
}
```

После удаления оставить только:
```go
env := []string{"SKILLS_PATH=/workspace/skills"}
```

---

### 3. Почистить конфигурацию

**Файл: `internal/docker/types.go`**
```go
type PoolConfig struct {
    // ...
    LLMAPIKeyEnv string    // Оставить — используется для чтения из env хоста в docker_spawn.go
    Environment  []string  // Удалить — больше не используется
    // ...
}
```

**Файл: `internal/config/schema.go`**
```go
type DockerConfig struct {
    // ...
    LLMAPIKeyEnv string   `toml:"llm_api_key_env"` // Оставить
    Environment  []string `toml:"environment"`     // Удалить
    // ...
}
```

**Файл: `internal/app/builders/docker_builder.go`**
- Удалить передачу `Environment` в `PoolConfig`

**Файл: `config.example.toml`**
- Удалить секцию `environment` или добавить комментарий что она deprecated

---

## Схема передачи секретов (после изменений)

| Секрет       | Источник           | Передача              | Безопасно |
| ------------ | ------------------ | --------------------- | --------- |
| ZAI_API_KEY  | os.Getenv на хосте | stdin (req.LLMAPIKey) | ✅ Да     |
| Task secrets | secretsFilter      | stdin (req.Secrets)   | ✅ Да     |

---

## Преимущества

1. **Безопасность** — секреты не видны в `docker inspect`
2. **Минимальные изменения** — только удаление кода
3. **Без breaking changes** — stdin-протокол уже работает
4. **Единая точка передачи** — все секреты через stdin

---

## Тестирование

### 1. Unit тест
```bash
# Проверить что env содержит только SKILLS_PATH
go test -v ./internal/docker -run TestCreateContainerEnv
```

### 2. Integration тест
```bash
# 1. Запустить Nexbot
./bin/nexbot serve &

# 2. Отправить spawn задачу (через Telegram или API)

# 3. Проверить docker inspect (контейнер должен быть еще жив)
docker ps -q | head -1 | xargs docker inspect | grep -A 5 "Env"
# Ожидается: только SKILLS_PATH=/workspace/skills

# 4. Убедиться что задача выполнена успешно
```

### 3. Security тест
```bash
# После изменений секреты НЕ должны быть видны:
docker inspect <container> 2>/dev/null | grep -i "ZAI_API_KEY\|SECRET\|PASSWORD\|TOKEN"
# Ожидается: пустой вывод
```

---

## Откат

Если после изменений subagent перестанет работать:

```bash
# 1. Откатить изменения
git revert <commit-hash>

# 2. Пересобрать
make build

# 3. Перезапустить сервис
systemctl restart nexbot  # или: pkill nexbot && ./bin/nexbot serve
```

---

## Критические зависимости

Перед реализацией этого плана **ОБЯЗАТЕЛЬНО** исправить:

### CRITICAL-1: simpleSecretsStore без zeroing

**Файл:** `cmd/nexbot/subagent.go:81-96`

**Проблема:** `simpleSecretsStore` хранит секреты в `map[string]string` без zeroing. После `Clear()` старые данные остаются в heap.

**Решение:** Заменить на `security.SecretsStore` (уже есть в `internal/security/secrets.go` с `zeroBytes()` и TTL).

```go
// БЫЛО:
type simpleSecretsStore struct {
    mu   sync.RWMutex
    data map[string]string
}

// СТАЛО:
import "nexbot/internal/security"

func NewSubagent(cfg *SubagentConfig) *Subagent {
    secretsStore: security.NewSecretsStore(5 * time.Minute),
}
```

---

### CRITICAL-2: Concurrent map write в CreateContainer

**Файл:** `internal/docker/pool.go:123`

**Проблема:** `p.containers[id] = container` выполняется без mutex. CreateContainer вызывается из двух мест:
1. `acquire()` (execute.go:214) — mutex освобождён
2. `RecreateUnhealthy()` (health.go:104) через `createAndStartContainer()`

**Решение:** Удалить запись из `CreateContainer`, изменить сигнатуру `createAndStartContainer`, добавить запись в вызывающих методах.

```go
// pool.go CreateContainer - УДАЛИТЬ строку 123:
// p.containers[id] = container  // ← УДАЛИТЬ

// pool.go CreateContainer - ИЗМЕНИТЬ return:
func (p *ContainerPool) CreateContainer(ctx context.Context) (*Container, error) {
    // ... existing code ...
    return container, nil  // Возвращаем *Container вместо записи в map
}

// pool.go acquire() - ДОБАВИТЬ после успешного создания (строка 215):
p.mu.Lock()
p.containers[container.ID] = container
p.mu.Unlock()

// health.go RecreateUnhealthy() - ДОБАВИТЬ после createAndStartContainer (строка 104):
container, err := p.createAndStartContainer(ctx)
if err != nil {
    return fmt.Errorf("failed to recreate container: %w", err)
}
p.mu.Lock()
p.containers[container.ID] = container
p.mu.Unlock()
```

---

### HIGH-1: markContainerDead без mutex

**Файл:** `internal/docker/execute.go:243-248`

**Проблема:** Читает `p.containers` и пишет `c.Status` без `p.mu.Lock()` — race condition с `acquire()`, `Release()`, `Stop()`.

**Решение:** Добавить mutex.

```go
func (p *Pool) markContainerDead(containerID string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if c, exists := p.containers[containerID]; exists {
        c.Status = StatusError
    }
}
```

---

### HIGH-2: Container.Status не thread-safe

**Файл:** `internal/docker/types.go:33`

**Проблема:** `Status` — обычный `string`, изменяется из разных горутин без синхронизации.

**Решение:** Использовать `atomic.Int32` для статуса.

```go
// types.go
type ContainerStatus int32

const (
    StatusIdle ContainerStatus = iota
    StatusBusy
    StatusError
)

type Container struct {
    ID     string
    status atomic.Int32  // Заменить string Status
    // ...
}

func (c *Container) GetStatus() ContainerStatus {
    return ContainerStatus(c.status.Load())
}

func (c *Container) SetStatus(s ContainerStatus) {
    c.status.Store(int32(s))
}
```

---

## Статус

### Критические зависимости (выполнить ДО)
- [ ] CRITICAL-1: Заменить simpleSecretsStore на security.SecretsStore (`cmd/nexbot/subagent.go`)
- [ ] CRITICAL-2a: Удалить запись из CreateContainer (`pool.go:123`)
- [ ] CRITICAL-2b: Добавить запись в acquire() под mutex (`pool.go:215`)
- [ ] CRITICAL-2c: Добавить запись в RecreateUnhealthy под mutex (`health.go:104`)

### Высокоприоритетные зависимости
- [ ] HIGH-1: Добавить mutex в markContainerDead (`execute.go:243-248`)
- [ ] HIGH-2: Заменить Container.Status на atomic.Int32 (`types.go:33`)

### Основные изменения
- [ ] Удалить LLMAPIKeyEnv из CreateContainer в client.go (строки 134-139)
- [ ] Удалить Environment из CreateContainer в client.go (строки 142-151)
- [ ] Удалить Environment из PoolConfig в types.go
- [ ] Удалить Environment из DockerConfig в schema.go
- [ ] Удалить передачу Environment в docker_builder.go
- [ ] Обновить config.example.toml

### Тестирование
- [ ] Unit тест: env содержит только SKILLS_PATH
- [ ] Integration тест: spawn задача выполняется успешно
- [ ] Security тест: `docker inspect` не показывает секреты
- [ ] Race detector тест: `go test -race ./internal/docker`
