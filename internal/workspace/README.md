# Workspace

## Назначение

Workspace обеспечивает управление рабочей директорией (workspace) агента. Хранит bootstrap файлы и настраивает среду для агента.

## Основные компоненты

### Bootstrap Files
Файлы bootstrap с путями:
- `BootstrapIdentity` — IDENTITY.md
- `BootstrapAgents` — AGENTS.md
- `BootstrapUser` — USER.md
- `BootstrapTools` — TOOLS.md
- `BootstrapHeartbeat` — HEARTBEAT.md
- `BootstrapMemory` — memory/

### Constants
Константы путей к bootstrap файлам.

## Использование

### Чтение bootstrap файлов

```go
import (
    "github.com/aatumaykin/nexbot/internal/workspace"
)

func main() {
    workspacePath := "/path/to/workspace"

    // Чтение identity
    identity, err := workspace.ReadBootstrapFile(workspacePath, workspace.BootstrapIdentity)

    // Чтение AGENTS
    agents, err := workspace.ReadBootstrapFile(workspacePath, workspace.BootstrapAgents)

    // Чтение USER
    user, err := workspace.ReadBootstrapFile(workspacePath, workspace.BootstrapUser)

    // Чтение TOOLS
    tools, err := workspace.ReadBootstrapFile(workspacePath, workspace.BootstrapTools)

    // Чтение HEARTBEAT
    heartbeat, err := workspace.ReadBootstrapFile(workspacePath, workspace.BootstrapHeartbeat)

    // Чтение памяти
    memoryDir := workspace.BootstrapMemory
}
```

### Обработка шаблонов

Bootstrap файлы поддерживают шаблоны:
- `{{CURRENT_TIME}}` — текущее время (HH:MM:SS)
- `{{CURRENT_DATE}}` — текущая дата (YYYY-MM-DD)
- `{{WORKSPACE_PATH}}` — путь к рабочей директории

## Конфигурация

### Workspace Path

- Базовый путь к рабочей директории (по умолчанию: `~/.nexbot`)

## Зависимости

- `os` — файловая система
- `path/filepath` — работа с путями

## Примечания

- Bootstrap файлы обязательны для контекста
- Порядок компонентов в контексте: AGENTS → IDENTITY → USER → TOOLS → HEARTBEAT → memory
- Memory файлы хранятся в директории памяти

## См. также

- `internal/agent/context` — использование bootstrap файлов
- `internal/workspace` — реализация
