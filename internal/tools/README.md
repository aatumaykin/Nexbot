# Tools

## Назначение

Tools обеспечивает систему инструментов (tools system) для выполнения действий агентом. Инструменты могут быть вызваны LLM через tool calling.

## Основные компоненты

### Tool
Интерфейс инструмента:
- `Name()` — имя инструмента
- `Description()` — описание
- `Parameters() map[string]interface{}` — параметры (JSON Schema)
- `Execute(ctx context.Context, params map[string]interface{}) (string, error)` — выполнение

### Registry
Реестр зарегистрированных инструментов с функциями:
- `Register(tool Tool) error` — регистрация инструмента
- `ByName(name string) (Tool, error)` — получение по имени
- `All() []Tool` — список всех инструментов
- `ToSchema() []llm.ToolDefinition` — конвертация в LLM tool definitions

### FileTool
Инструмент для работы с файлами:
- `ReadFile(path string) (string, error)`
- `WriteFile(path string, content string) error`
- `ListFiles(dir string) ([]string, error)`

### ShellTool
Инструмент для выполнения shell команд:
- `ExecuteCommand(cmd string) (string, error)`

### FetchTool
Инструмент для загрузки веб-страниц по URL:
- `url` (string, required) — URL для загрузки (должен начинаться с http:// или https://)
- `format` (string, enum: "text", "html", default: "text") — формат вывода
- Возвращает JSON с метаданными: `url`, `status`, `contentType`, `length`, `content`
- Поддерживает удаление HTML тегов при format="text"
- Ограничения: timeout (по умолчанию 30s), max_response_size (по умолчанию 5MB)

## Использование

### Реализация интерфейса

```go
type SearchTool struct{}

func (t *SearchTool) Name() string {
    return "search"
}

func (t *SearchTool) Description() string {
    return "Поиск информации в файловой системе"
}

func (t *SearchTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "query": map[string]interface{}{
                "type":        "string",
                "description": "Поисковый запрос",
            },
        },
    }
}

func (t *SearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
    query := params["query"].(string)
    // Выполнение поиска
    return fmt.Sprintf("Результаты поиска для: %s", query), nil
}
```

### Регистрация инструмента

```go
import (
    "github.com/aatumaykin/nexbot/internal/tools"
)

func main() {
    tool := &SearchTool{}
    registry := tools.NewRegistry()

    // Регистрация
    err := registry.Register(tool)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Конвертация в LLM schema

```go
// Конвертация в tool definitions для LLM
schemas := registry.ToSchema()

for _, schema := range schemas {
    fmt.Printf("%s: %s\n", schema.Name, schema.Description)
}
```

## Конфигурация

### Tool

- `Name()` — уникальное имя
- `Description()` — описание
- `Parameters()` — JSON Schema параметров
- `Execute()` — выполнение

## Зависимости

- `github.com/aatumaykin/nexbot/internal/llm` — LLM tool definitions

## Примечания

- JSON Schema для параметров используется LLM
- Execute вызывается с map[string]interface{}
- Tools используются в recursive tool calling

## См. также

- `internal/skills` — навыки
- `internal/agent/loop` — tool calling
- `internal/llm` — LLM tool definitions
