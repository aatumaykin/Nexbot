# Структура файлов

## Описание
Секция показывает сравнение структуры файлов до и после рефакторинга. Наглядно демонстрирует, какие директории и файлы будут изменены, добавлены или удалены.

## Ключевые элементы
### До
- `docs/workspace/` — плоская структура с IDENTITY.md, AGENTS.md, TOOLS.md, USER.md
- `prompts/` — отдельная директория с identity.md, security.md
- `cmd/nexbot/` — main.go, serve.go
- `cmd/subagent/` — отдельный main.go для сабагента
- `Dockerfile.subagent` — кастомный Dockerfile

### После
- `docs/workspace/main/` — bootstrap файлы для основного процесса
- `docs/workspace/subagent/AGENTS.md` — объединённый промпт для сабагента
- `docs/workspace/memory/MEMORY.md` — память
- `cmd/nexbot/` — добавлен subagent.go (новый)
- `cmd/subagent/` — удалена вся директория
- `Dockerfile.subagent` — удалён

## Связи с другими секциями
- Зависит от **03-key-decisions.md** — реализует решения
- Связана с **05-implementation-stages.md**:
  - Этап 1: Структура файлов
  - Этап 3: Subcommand
  - Этап 6: Cleanup

## Практические выводы
1. Создать `docs/workspace/main/` и перенести bootstrap файлы
2. Создать `docs/workspace/subagent/AGENTS.md` (объединить identity + security)
3. Добавить `cmd/nexbot/subagent.go`
4. Удалить `cmd/subagent/` директорию
5. Удалить `prompts/` директорию
6. Удалить `Dockerfile.subagent`
7. Обновить `internal/workspace/bootstrap.go` для новой структуры
