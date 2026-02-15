# Этапы реализации

## Описание
Секция описывает 7 этапов реализации с чек-листами конкретных задач, приоритетами и указанием конкретных файлов и функций для изменения.

## Ключевые элементы

### Этап 1: Структура файлов (высокий приоритет)
- Создать `docs/workspace/main/` и перенести bootstrap файлы
- Создать `docs/workspace/subagent/AGENTS.md`
- Удалить `prompts/` директорию

### Этап 2: Bootstrap (высокий приоритет)
- Файл: `internal/workspace/bootstrap.go`
- Добавить создание main/ и subagent/ при первом запуске
- Встроить default промпты через `go:embed` для fallback

### Этап 3: Subcommand (высокий приоритет)
- Файл: `cmd/nexbot/subagent.go` (новый)
- Перенести логику из `cmd/subagent/main.go`
- Удалить `cmd/subagent/` директорию

### Этап 4: Docker SDK (высокий приоритет)
- Файлы: `internal/docker/client.go`, `internal/docker/pool.go`
- Использовать `alpine:3.23` вместо `nexbot/subagent`
- Добавить volume mounts для бинарника (`os.Executable`)
- Добавить volume mounts для subagent/ и skills/

### Этап 5: Конфигурация (средний приоритет)
- Удалить: `DockerConfig.ImageName`, `ImageTag`, `ImageDigest`
- Добавить: `BinaryPath`, `SubagentPromptsPath`, `SkillsMountPath`
- Файлы: config.example.toml, internal/config/defaults.go

### Этап 6: Cleanup (низкий приоритет)
- Удалить `Dockerfile.subagent`
- Удалить docker-build-subagent, docker-push-subagent из makefiles
- Упростить Makefile

### Этап 7: Тесты и CI (средний приоритет)
- Обновить тесты `internal/docker/`
- Обновить тесты `internal/workspace/`
- Проверить `make ci`

## Связи с другими секциями
- Реализует все решения из **03-key-decisions.md**
- Реализует структуру из **04-file-structure.md**
- Учитывает риски из **06-risks-dependencies.md**

## Практические выводы
1. Начать с Этапа 1-4 (высокий приоритет)
2. Последовательность: структура → bootstrap → subcommand → docker SDK
3. После высокоприоритетных этапов — конфигурация и тесты
4. Cleanup можно делать параллельно или в конце
5. Обязательно: `git push` после завершения всех этапов
