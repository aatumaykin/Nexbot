# Архитектура

## Описание
Секция содержит диаграмму взаимодействия между хост-системой и контейнером сабагента. Показывает схему volume mounts, структуру workspace и переменные окружения для контейнеризированного процесса.

## Ключевые элементы
- **Хост-система**:
  - `/usr/local/bin/nexbot` — единый бинарник
  - `~/.nexbot/` — workspace с поддиректориями main/, subagent/, skills/, memory/
  - Bootstrap файлы: IDENTITY.md, AGENTS.md, TOOLS.md, USER.md

- **Контейнер (alpine:3.23)**:
  - Command: `[nexbot, subagent]` — запуск подкоманды
  - Mounts: бинарник + `/workspace/subagent/` + `/workspace/skills/`
  - Env: `SKILLS_PATH=/workspace/skills`

- **Volume mounts**:
  - Бинарник с хоста монтируется в контейнер (read-only)
  - subagent/ — конфигурация сабагента (AGENTS.md)
  - skills/ — внешние skills для выполнения задач

## Связи с другими секциями
- Зависит от **01-overview.md** — реализует концепцию единого бинарника
- Связана с **04-file-structure.md** — структура workspace
- Связана с **05-implementation-stages.md** — Этап 4 (Docker SDK)

## Практические выводы
1. Обновить `internal/docker/client.go` для использования alpine:3.23
2. Добавить volume mounts для бинарника через `os.Executable()`
3. Настроить переменные окружения в контейнере
4. Убрать зависимость от кастомного Docker-образа
