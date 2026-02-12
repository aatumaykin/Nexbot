# Руководство для разработчиков

Это руководство поможет вам начать разработку Nexbot.

## Требования

- **Go 1.26+**
- **Git**
- **Make** (для сборки)

## Установка

### Клонирование репозитория

```bash
git clone https://github.com/aatumaykin/nexbot.git
cd nexbot
```

### Установка зависимостей

```bash
go mod download
```

### Проверка установки

```bash
go version  # Должен быть 1.26 или выше
```

## Структура проекта

```
nexbot/
├── cmd/
│   └── nexbot/
│       └── main.go                 # Точка входа
├── internal/
│   ├── agent/
│   │   ├── loop.go                 # Agent loop (core)
│   │   ├── context.go              # System prompt builder
│   │   ├── memory.go               # Memory store
│   │   ├── session.go              # Session manager
│   │   └── tools.go                # Tool registry
│   ├── bus/
│   │   ├── events.go               # Event types
│   │   └── queue.go                # Message queue
│   ├── channels/
│   │   ├── connector.go            # Connector interface
│   │   └── telegram/
│   │       └── connector.go        # Telegram implementation
│   ├── llm/
│   │   ├── provider.go             # LLM provider interface
│   │   ├── zai.go                  # Z.ai implementation
│   │   └── openai.go               # OpenAI implementation
│   ├── skills/
│   │   ├── loader.go               # Skills loader
│   │   ├── parser.go               # SKILL.md parser
│   │   └── metadata.go             # Skill metadata
│   ├── tools/
│   │   ├── registry.go             # Tool registry
│   │   ├── file.go                 # File operations
│   │   └── shell.go                # Shell execution
│   ├── workspace/
│   │   ├── workspace.go            # Workspace manager
│   │   └── bootstrap.go            # Bootstrap files loader
│   ├── config/
│   │   ├── config.go               # TOML config parsing
│   │   └── schema.go               # Config structs
│   └── logger/
│       └── logger.go               # slog wrapper
├── pkg/
│   └── messagebus/                 # Public message bus
├── workspace/                      # Bootstrap files
│   ├── AGENTS.md
│   ├── SOUL.md
│   ├── USER.md
│   ├── TOOLS.md
│   └── IDENTITY.md
├── skills/
│   └── examples/
│       └── example-skill/
│           └── SKILL.md
├── docs/                           # Documentation
├── config.example.toml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Сборка

### Локальная сборка

```bash
make build
```

Создаст бинарник `nexbot` в текущей директории.

### Сборка для всех платформ

```bash
make build-all
```

Создаст бинарники для:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

### Установка в /usr/local/bin

```bash
make install
```

### Установка в ~/bin

```bash
make install-user
```

## Тестирование

### Запуск всех тестов

```bash
make test
```

### Запуск тестов с покрытием

```bash
make test-cover
```

### Запуск тестов конкретного пакета

```bash
go test ./internal/agent/...
go test ./internal/tools/...
```

### Запуск конкретного теста

```bash
go test -run TestWorkspace ./internal/workspace/...
```

### Запуск тестов с verbose выводом

```bash
go test -v ./...
```

## Линтеры и форматирование

### Форматирование кода

```bash
make fmt
```

Или вручную:

```bash
go fmt ./...
```

### Запуск линтера

```bash
make lint
```

### Запуск всех CI проверок

```bash
make ci
```

Это включает:
- `make fmt` — форматирование
- `make lint` — линтеры
- `make test` — тесты

## Разработка

### Добавление нового инструмента

1. Создайте файл в `internal/tools/`:

```go
package tools

import (
    "context"
    "fmt"
)

// NewMyTool создаёт новый инструмент
func NewMyTool() *Tool {
    return &Tool{
        Name: "my_tool",
        Description: "Описание инструмента",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "arg1": map[string]interface{}{
                    "type": "string",
                    "description": "Описание аргумента",
                },
            },
            "required": []string{"arg1"},
        },
        Execute: executeMyTool,
    }
}

func executeMyTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    arg1, ok := args["arg1"].(string)
    if !ok {
        return nil, fmt.Errorf("arg1 is required and must be a string")
    }

    // Логика инструмента
    result := fmt.Sprintf("Результат: %s", arg1)

    return result, nil
}
```

2. Зарегистрируйте инструмент в `internal/tools/registry.go`:

```go
import (
    "github.com/aatumaykin/nexbot/internal/tools"
    "github.com/aatumaykin/nexbot/internal/tools/file"
)

func NewRegistry(ctx context.Context, workspace *workspace.Workspace) *Registry {
    r := &Registry{
        ctx:       ctx,
        workspace: workspace,
        tools:     make(map[string]*Tool),
    }

    // Регистрация инструментов
    r.Register(file.NewReadFileTool(workspace, config))
    r.Register(file.NewWriteFileTool(workspace, config))
    r.Register(file.NewListDirTool(workspace, config))
    r.Register(file.NewDeleteFileTool(workspace, config))
    r.Register(tools.NewShellTool(workspace))
    r.Register(tools.NewMyTool())  // Новый инструмент

    return r
}
```

3. Добавьте тесты:

```go
package tools

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyTool(t *testing.T) {
    tool := NewMyTool()

    t.Run("success case", func(t *testing.T) {
        result, err := tool.Execute(context.Background(), map[string]interface{}{
            "arg1": "test",
        })

        assert.NoError(t, err)
        assert.Contains(t, result, "test")
    })

    t.Run("missing arg1", func(t *testing.T) {
        _, err := tool.Execute(context.Background(), map[string]interface{}{})

        assert.Error(t, err)
        assert.Contains(t, err.Error(), "arg1 is required")
    })
}
```

### Добавление нового канала

1. Создайте интерфейс канала в `internal/channels/connector.go` (если не существует):

```go
package channels

import (
    "context"
)

type InboundMessage struct {
    ChannelID string
    UserID    string
    Content   string
    Metadata  map[string]interface{}
}

type OutboundMessage struct {
    ChannelID string
    UserID    string
    Content   string
    Metadata  map[string]interface{}
}

type Connector interface {
    Start(ctx context.Context, inboundCh chan<- InboundMessage) error
    Stop() error
    SendMessage(ctx context.Context, msg OutboundMessage) error
}
```

2. Реализуйте канал в `internal/channels/yourchannel/connector.go`:

```go
package yourchannel

import (
    "context"
    "github.com/aatumaykin/nexbot/internal/channels"
)

type Connector struct {
    config Config
}

type Config struct {
    Token     string
    Enabled   bool
}

func NewConnector(config Config) *Connector {
    return &Connector{
        config: config,
    }
}

func (c *Connector) Start(ctx context.Context, inboundCh chan<- channels.InboundMessage) error {
    // Подключение к каналу
    // Обработка входящих сообщений
    // Отправка в inboundCh
    return nil
}

func (c *Connector) Stop() error {
    // Очистка ресурсов
    return nil
}

func (c *Connector) SendMessage(ctx context.Context, msg channels.OutboundMessage) error {
    // Отправка сообщения в канал
    return nil
}
```

3. Зарегистрируйте канал в `cmd/nexbot/main.go` или в менеджере каналов.

### Добавление нового навыка (Skill)

1. Создайте директорию для навыка:

```bash
mkdir -p skills/my-skill
```

2. Создайте `SKILL.md`:

```markdown
---
name: my-skill
description: Описание навыка
tools: [read_file, shell_exec]
---

# My Skill

Подробное описание навыка на русском языке.

## Примеры использования

Примеры того, как использовать этот навык.

## Параметры

Описание параметров и их значений.
```

3. Поместите файл в `~/.nexbot/skills/my-skill/SKILL.md` или `skills/my-skill/SKILL.md`.

### Добавление нового LLM провайдера

1. Создайте файл в `internal/llm/yourprovider.go`:

```go
package llm

import (
    "context"
    "encoding/json"
    "net/http"
)

type YourProvider struct {
    client    *http.Client
    apiKey    string
    baseURL   string
    model     string
}

func NewYourProvider(apiKey, baseURL, model string) *YourProvider {
    return &YourProvider{
        client:  &http.Client{},
        apiKey:  apiKey,
        baseURL: baseURL,
        model:   model,
    }
}

func (p *YourProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // Реализация запроса к API
    // Парсинг ответа
    // Возврат ChatResponse
    return nil, nil
}

func (p *YourProvider) SupportsToolCalling() bool {
    return true
}

func (p *YourProvider) GetDefaultModel() string {
    return p.model
}
```

2. Добавьте выбор провайдера в конфигурацию и валидацию.

## Отладка

### Запуск в режиме debug

```bash
nexbot serve --config config.toml
```

Установите уровень логирования в `config.toml`:

```toml
[logging]
level = "debug"
format = "text"
output = "stdout"
```

### Использование delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/nexbot
```

### Логи

Логи записываются в соответствии с конфигурацией:

```toml
[logging]
output = "~/.nexbot/nexbot.log"
```

## Конвенции кода

### Стиль кода

- Используйте `go fmt` для форматирования
- Следуйте Effective Go: https://go.dev/doc/effective_go
- Используйте именованные возвращаемые значения
- Обрабатывайте ошибки явно

### Именование

- Пакеты: короткие, нижний регистр (например, `agent`, `bus`)
- Экспортируемые типы: PascalCase (например, `Agent`, `Message`)
- Неэкспортируемые типы: camelCase (например, `agentLoop`, `message`)
- Константы: UPPER_SNAKE_CASE (например, `MAX_ITERATIONS`)
- Интерфейсы: описательные имена с суффиксом `-er` (например, `Connector`, `Provider`)

### Комментарии

- Пакетный комментарий: объясняет, что делает пакет
- Экспортируемые функции: что делает функция, параметры, возвращаемые значения
- Сложная логика: пояснения "почему", а не "что"

### Структура

- Один файл = одна основная сущность
- Порядок: константы → типы → переменные → интерфейсы → функции
- Сначала публичные API, потом приватные

## Контрибьюция

### Process контрибьюции

1. Форкните репозиторий
2. Создайте ветку: `git checkout -b feature/amazing-feature`
3. Внесите изменения
4. Запустите `make ci` для проверки
5. Сделайте commit: `git commit -m 'Add amazing feature'`
6. Push в ветку: `git push origin feature/amazing-feature`
7. Создайте Pull Request

### Требования к PR

- Все тесты проходят (`make test`)
- Линтеры проходят (`make lint`)
- Код отформатирован (`make fmt`)
- Добавлены тесты для нового функционала
- Обновлена документация (при необходимости)
- Commit message следует конвенциям проекта

### Правила безопасности

- Никогда не коммитите секреты (API ключи, пароли)
- Используйте переменные окружения для секретов
- Маскируйте секреты в логах и сообщениях об ошибках
- Всегда проверяйте валидацию пользовательского ввода

## Релиз

### Версионирование

Используется [Semantic Versioning](https://semver.org/):
- `MAJOR.MINOR.PATCH`
- MAJOR: несовместимые изменения API
- MINOR: новые функции, обратно совместимые
- PATCH: исправления ошибок, обратно совместимые

### Создание релиза

1. Обновите версию в `go.mod`
2. Обновите `CHANGELOG.md` (если есть)
3. Создайте git tag:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
4. Соберите все бинарники:
   ```bash
   make release
   ```
5. Создайте GitHub Release с:
   - Версией
   - Описанием изменений
   - Загруженными бинарниками
   - Checksums

## Документация

### Структура документации

Документация находится в директории `docs/`:
- `README.md` — вводный гайд, установка, быстрый старт
- `CONFIGURATION.md` — полная справка по конфигурации
- `ARCHITECTURE.md` — архитектура системы
- `DEVELOPMENT.md` — руководство для разработчиков (этот файл)
- `EXAMPLES.md` — практические примеры

### Обновление документации

- Обновляйте документацию при изменениях API
- Добавляйте примеры для нового функционала
- Пишите документацию на русском языке
- Проверяйте ссылки и корректность синтаксиса markdown

## Полезные ссылки

- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Standard Library](https://pkg.go.dev/std)
- [Nexbot Documentation](README.md)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Z.ai API](https://z.ai)

## Поддержка

Если у вас есть вопросы или проблемы:
- Проверьте существующие [Issues](https://github.com/aatumaykin/nexbot/issues)
- Создайте новый Issue с описанием проблемы
- Присоединитесь к обсуждениям в Discord/Telegram (если есть)

---

Спасибо за интерес к разработке Nexbot! 🚀
