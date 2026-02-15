# План: Единый бинарник serve/subagent

## Обзор

Переход от двух бинарников (nexbot + subagent) и отдельного Docker-образа к единому бинарнику с подкомандами. Контейнеры сабагентов запускаются на alpine с volume mounts.

## Архитектура

```
ХОСТ                                      КОНТЕЙНЕР
┌────────────────────────────┐            ┌─────────────────────────────┐
│ /usr/local/bin/nexbot      │──mount────▶│ alpine:3.23                 │
│                            │            │                             │
│ ~/.nexbot/                 │            │ command: [nexbot, subagent] │
│ ├── main/                  │            │                             │
│ │   ├── IDENTITY.md        │            │ mounts:                     │
│ │   ├── AGENTS.md          │            │ /usr/local/bin/nexbot       │
│ │   ├── TOOLS.md           │            │ /workspace/subagent/        │
│ │   └── USER.md            │            │ /workspace/skills/          │
│ ├── memory/                │            │                             │
│ ├── subagent/              │            │ env:                        │
│ │   └── AGENTS.md          │            │ SKILLS_PATH=/workspace/     │
│ └── skills/                │            │   skills                    │
└────────────────────────────┘            └─────────────────────────────┘
```

## Ключевые решения

| Решение                    | Обоснование                                     |
| -------------------------- | ----------------------------------------------- |
| Один бинарник              | Не нужно пересобирать Docker-образ при изменениях |
| Cobra subcommand           | Консистентно с текущей структурой cmd/nexbot    |
| alpine:3.23 + volume mounts| Образ ~5MB, бинарник подменяется на хосте       |
| Поддиректории main/subagent| Консистентная структура workspace               |
| Один AGENTS.md для сабагента| Упрощение: identity + security объединены       |

## Структура файлов

### До

```
docs/workspace/
├── IDENTITY.md
├── AGENTS.md
├── TOOLS.md
├── USER.md
└── memory/

prompts/
├── identity.md
└── security.md

cmd/
├── nexbot/
│   ├── main.go
│   └── serve.go
└── subagent/
    └── main.go

Dockerfile.subagent
```

### После

```
docs/workspace/
├── main/
│   ├── IDENTITY.md
│   ├── AGENTS.md
│   ├── TOOLS.md
│   └── USER.md
├── memory/
│   └── MEMORY.md
└── subagent/
    └── AGENTS.md

cmd/
└── nexbot/
    ├── main.go
    ├── serve.go
    └── subagent.go    ← новый

Dockerfile.subagent     ← удалить
```

## Этапы

### Этап 1: Структура файлов (высокий приоритет)

- [ ] Создать `docs/workspace/main/` и перенести bootstrap файлы
- [ ] Создать `docs/workspace/subagent/AGENTS.md` (объединить identity + security)
- [ ] Удалить `prompts/` директорию

### Этап 2: Bootstrap (высокий приоритет)

- [ ] Обновить `internal/workspace/bootstrap.go` для новой структуры поддиректорий
- [ ] Добавить создание `main/` и `subagent/` при первом запуске
- [ ] Встроить default промпты через `go:embed` для fallback

### Этап 3: Subcommand (высокий приоритет)

- [ ] Добавить `cmd/nexbot/subagent.go` с Cobra subcommand
- [ ] Перенести логику из `cmd/subagent/main.go`
- [ ] Удалить `cmd/subagent/` директорию

### Этап 4: Docker SDK (высокий приоритет)

- [ ] Обновить `internal/docker/client.go`:
  - Использовать `alpine:3.23` вместо `nexbot/subagent`
  - Добавить volume mounts для бинарника
  - Добавить volume mounts для subagent/ и skills/
- [ ] Обновить `internal/docker/pool.go` для новой конфигурации
- [ ] Добавить определение пути к бинарнику (os.Executable)

### Этап 5: Конфигурация (средний приоритет)

- [ ] Удалить `DockerConfig.ImageName`, `ImageTag`, `ImageDigest`
- [ ] Добавить `BinaryPath` (auto-detect), `SubagentPromptsPath`, `SkillsMountPath`
- [ ] Обновить `config.example.toml`
- [ ] Обновить defaults в `internal/config/defaults.go`

### Этап 6: Cleanup (низкий приоритет)

- [ ] Удалить `Dockerfile.subagent`
- [ ] Удалить `makefiles/docker.mk` (docker-build-subagent, docker-push-subagent)
- [ ] Упростить `Makefile` — убрать build-subagent targets

### Этап 7: Тесты и CI (средний приоритет)

- [ ] Обновить тесты `internal/docker/`
- [ ] Обновить тесты `internal/workspace/`
- [ ] Проверить `make ci`
- [ ] `git push`

## Зависимости

Без новых зависимостей — используется существующий Docker SDK.

## Риски

| Риск                           | Митигация                                    |
| ------------------------------ | -------------------------------------------- |
| Бинарник не найден             | `os.Executable()` + fallback + понятная ошибка |
| Volume mount не работает       | Проверка прав доступа + логирование          |
| Промпты не найдены в workspace | go:embed fallback defaults                   |

## Отменяется

- Сборка Docker-образа `nexbot/subagent`
- Push образа в registry
- Multi-stage Dockerfile для сабагента
- CI/CD для Docker-образа

## Выигрыш

| Метрика                 | До                    | После               |
| ----------------------- | --------------------- | ------------------- |
| Docker-образы           | 2 (nexbot + subagent) | 1 (alpine)          |
| Размер образа сабагента | ~20MB                 | ~5MB (alpine)       |
| Пересборка при изменениях | Образ + push        | Только бинарник     |
| CI/CD этапы             | build + docker push   | build               |
| Время деплоя            | минуты                | секунды             |
