# LLM Provider

## Назначение

LLM Provider определяет интерфейс для провайдеров Large Language Model. Разные провайдеры (OpenAI, Anthropic, Z.ai) должны реализовывать этот интерфейс.

## Основные компоненты

### Provider
Интерфейс провайдера:
- `Chat` — отправка запроса chat completion
- `SupportsToolCalling` — поддержка tool calling

### Role
Роль сообщения:
- `RoleSystem` — системное сообщение
- `RoleUser` — сообщение пользователя
- `RoleAssistant` — сообщение ассистента
- `RoleTool` — сообщение инструмента

### Message
Сообщение в диалоге:
- `Role` — роль отправителя
- `Content` — содержимое
- `ToolCallID` — ID вызова инструмента (для RoleTool)

### FinishReason
Причина завершения генерации:
- `FinishReasonStop` — естественное завершение
- `FinishReasonLength` — превышен max tokens
- `FinishReasonToolCalls` — запрошены tool calls
- `FinishReasonError` — ошибка

### ToolCall
Вызов инструмента:
- `ID` — уникальный ID
- `Name` — имя инструмента
- `Arguments` — аргументы (JSON string)

### ToolDefinition
Определение инструмента:
- `Name` — имя инструмента
- `Description` — описание
- `Parameters` — параметры (JSON Schema)

### ChatRequest
Запрос chat completion:
- `Messages` — история сообщений
- `Model` — модель
- `Temperature` — температура
- `MaxTokens` — максимальное количество токенов
- `Tools` — инструменты

### ChatResponse
Ответ от провайдера:
- `Content` — содержимое
- `FinishReason` — причина завершения
- `ToolCalls` — запрошенные tool calls
- `Usage` — использование токенов
- `Model` — используемая модель

### Usage
Использование токенов:
- `PromptTokens` — токены в промпте
- `CompletionTokens` — токены в завершении
- `TotalTokens` — общее количество

## Использование

### Реализация интерфейса

```go
type MyProvider struct {
    // конфигурация
}

func (p *MyProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
    // отправка запроса к LLM API
    // обработка ответа
    return &llm.ChatResponse{
        Content:      "ответ",
        FinishReason: llm.FinishReasonStop,
        Model:        "model-name",
    }, nil
}

func (p *MyProvider) SupportsToolCalling() bool {
    return true
}
```

### Использование провайдера

```go
// Создание провайдера
provider := zai.NewProvider(cfg)

// Запрос chat completion
req := llm.ChatRequest{
    Messages: []llm.Message{
        {Role: llm.RoleUser, Content: "Привет!"},
    },
    Model:       "glm-4.7-flash",
    Temperature: 0.7,
    MaxTokens:   4096,
    Tools: []llm.ToolDefinition{
        {
            Name:        "search",
            Description: "Поиск информации",
            Parameters:  map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "query": map[string]interface{}{
                        "type": "string",
                    },
                },
            },
        },
    },
}

resp, err := provider.Chat(ctx, req)
if err != nil {
    log.Error("LLM call failed", err)
}

// Обработка ответа
switch resp.FinishReason {
case llm.FinishReasonToolCalls:
    for _, toolCall := range resp.ToolCalls {
        // выполнение инструмента
    }
case llm.FinishReasonStop:
    fmt.Println(resp.Content)
}
```

## Конфигурация

### Z.ai Provider

- `BaseURL` — базовый URL API (по умолчанию: https://api.z.ai/api/coding/paas/v4)
- `APIKey` — API ключ
- `TimeoutSeconds` — timeout (по умолчанию: 30)

## Зависимости

- `context` — управление контекстом

## Примечания

- Tool calling поддерживается через JSON Schema
- Tool messages добавляются в историю для recursive tool calling
- Models могут отличаться от запрошенных
- ToolCallID используется для связывания результатов с вызовами

## См. также

- `internal/agent/loop` — использование провайдера в loop
- `internal/tools` — инструменты
