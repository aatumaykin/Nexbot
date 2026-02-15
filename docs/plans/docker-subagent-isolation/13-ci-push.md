# Этап 13: CI + Push

## Цель

Финальная проверка и публикация изменений.

## Действия

### 1. Запуск CI проверок

```bash
make ci
```

Включает:
- `make lint` — статический анализ
- `make test` — все тесты
- `make build` — сборка
- `make vet` — go vet

### 2. Проверка кода

```bash
# Форматирование
make fmt

# Линтер
make lint

# Тесты
make test

# Тесты с coverage
make test-coverage
```

### 3. Git операции

```bash
# Проверить статус
git status

# Добавить файлы
git add .

# Коммит
git commit -m "feat: add Docker subagent isolation

- Add prompt injection protection with RE2 regex
- Implement container pool with Circuit Breaker
- Add stdin-based secrets transmission
- Add health checks and graceful shutdown
- Add spawn tool for delegation"

# Push в remote
git push -u origin feature/docker-subagent-isolation
```

### 4. Pull Request

```bash
# Создать PR
gh pr create --title "feat: Docker subagent isolation" --body "
## Summary

- Prompt injection protection with RE2 regex and NFKC normalization
- Docker container pool with Circuit Breaker and Rate Limiting
- Secure secrets transmission via stdin JSON
- Health checks with auto-recreate
- Graceful shutdown with drain mode
- spawn tool for task delegation

## Changes

### New Files
- `internal/subagent/sanitizer/sanitizer.go` - Injection protection
- `internal/subagent/prompts/loader.go` - Dynamic prompts
- `internal/docker/` - Docker pool implementation
- `cmd/subagent/` - Subagent CLI
- `internal/tools/spawn.go` - Spawn tool
- `Dockerfile.subagent` - Container image

### Modified Files
- `internal/config/schema.go` - Docker config
- `internal/app/` - Integration

## Test Plan

- [x] Unit tests for sanitizer
- [x] Unit tests for Circuit Breaker
- [x] Unit tests for Rate Limiter
- [x] Unit tests for secrets store
- [ ] Integration tests with Docker (manual)
- [ ] End-to-end test (manual)

## Checklist

- [x] Code follows project conventions
- [x] All tests pass
- [x] Linter passes
- [x] Documentation updated
"
```

### 5. После merge

```bash
# Переключиться на main
git checkout main

# Pull изменения
git pull

# Собрать и опубликовать Docker образ
make docker-build-subagent
make docker-push-subagent IMAGE_TAG=v1.0.0
```

## CI Pipeline

### GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: make lint

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: make test

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: make build

  docker:
    runs-on: ubuntu-latest
    needs: [lint, test, build]
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - run: |
          make docker-build-subagent-multiarch \
            DOCKER_REGISTRY=ghcr.io/${{ github.repository_owner }} \
            IMAGE_TAG=latest
```

## Release Checklist

### Pre-release

- [ ] Все тесты проходят
- [ ] Линтер не выдает ошибок
- [ ] Документация обновлена
- [ ] CHANGELOG.md обновлен
- [ ] Версия обновлена в коде

### Release

```bash
# Создать тег
git tag -a v1.0.0 -m "Release v1.0.0: Docker subagent isolation"

# Push тег
git push origin v1.0.0

# Собрать Docker образ с тегом
make docker-build-subagent IMAGE_TAG=v1.0.0
make docker-push-subagent IMAGE_TAG=v1.0.0
```

### Post-release

- [ ] GitHub Release создан
- [ ] Docker образ опубликован
- [ ] Документация обновлена
- [ ] Announcement в чате/блоге

## Rollback Plan

Если что-то пошло не так:

```bash
# Откатить коммит
git revert HEAD

# Или сбросить ветку
git reset --hard HEAD~1

# Пересобрать и передеплоить
make build
make docker-build-subagent
make docker-push-subagent
```

## Ключевые решения

1. **make ci** — единая команда для всех проверок
2. **GitHub Actions** — автоматический CI/CD
3. **Conventional commits** — понятные коммиты
4. **Docker multi-arch** — поддержка amd64/arm64
5. **Rollback plan** — план отката при проблемах
