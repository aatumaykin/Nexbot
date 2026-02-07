# Skills

## Назначение

Skills обеспечивает систему навыков (skills system) для расширения функциональности агента. Skills — это внешние модули, расширяющие возможности агента.

## Основные компоненты

### Skill
Интерфейс навыка:
- `Name()` — имя навыка
- `Description()` — описание
- `HandleMessage(msg llm.Message) (llm.Message, error)` — обработка сообщения

### Registry
Реестр зарегистрированных навыков с функциями:
- `Register(skill Skill) error` — регистрация навыка
- `ByName(name string) (Skill, error)` — получение по имени
- `All() []Skill` — список всех навыков
- `ToSchema() []tools.ToolDefinition` — конвертация в инструменты

## Использование

### Реализация интерфейса

```go
type MySkill struct{}

func (s *MySkill) Name() string {
    return "my-skill"
}

func (s *MySkill) Description() string {
    return "Описание навыка"
}

func (s *MySkill) HandleMessage(msg llm.Message) (llm.Message, error) {
    // Обработка сообщения
    return llm.Message{
        Role:    llm.RoleAssistant,
        Content: "Результат обработки",
    }, nil
}
```

### Регистрация навыка

```go
import (
    "github.com/aatumaykin/nexbot/internal/skills"
    "github.com/aatumaykin/nexbot/internal/tools"
)

func main() {
    skill := &MySkill{}
    registry := skills.NewRegistry()

    // Регистрация
    err := registry.Register(skill)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Использование в loop

```go
// Регистрация навыка в loop
skill := &MySkill{}
err := loop.RegisterSkill(skill)
```

## Конфигурация

### Skill

- `Name()` — уникальное имя
- `Description()` — описание
- `HandleMessage()` — обработка

## Зависимости

- `github.com/aatumaykin/nexbot/internal/tools` — инструменты
- `github.com/aatumaykin/nexbot/internal/llm` — сообщения

## Примечания

- Skills совместимы с OpenClaw
- Регистрация происходит через tools.Registry
- Skills могут использоваться для расширения функциональности

## См. также

- `internal/tools` — инструменты
- `internal/agent/loop` — использование навыков
