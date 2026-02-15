# Этап 11: Makefile

## Цель

Добавить команды для сборки и публикации Docker-образов.

## Файлы

### `makefiles/docker.mk`

```makefile
# Docker commands for subagent

DOCKER_REGISTRY ?= 
IMAGE_NAME ?= nexbot/subagent
IMAGE_TAG ?= latest

.PHONY: docker-build-subagent
docker-build-subagent:
	@echo "Building subagent Docker image..."
	docker build -f Dockerfile.subagent -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-build-subagent-multiarch
docker-build-subagent-multiarch:
	@echo "Building multi-architecture subagent image..."
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.subagent \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--push .
	@echo "Multi-arch image pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-push-subagent
docker-push-subagent:
	@echo "Pushing subagent image..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "Image pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-tag-subagent
docker-tag-subagent:
	@if [ -z "$(NEW_tag)" ]; then \
		echo "Usage: make docker-tag-subagent NEW_TAG=v1.0.0"; \
		exit 1; \
	fi
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):$(NEW_TAG)
	@echo "Tagged: $(IMAGE_NAME):$(NEW_TAG)"

.PHONY: docker-test-subagent
docker-test-subagent:
	@echo "Testing subagent image..."
	docker run --rm -i $(IMAGE_NAME):$(IMAGE_TAG) <<'EOF'
{"version":"1.0","type":"ping","id":"test-1"}
EOF

.PHONY: docker-inspect-subagent
docker-inspect-subagent:
	@echo "Inspecting subagent image..."
	docker images $(IMAGE_NAME)
	docker inspect $(IMAGE_NAME):$(IMAGE_TAG) | jq '.[0].Size'
```

### Обновление `Makefile`

```makefile
# Include docker commands
include makefiles/docker.mk

# ... existing targets ...

# Build all (including docker)
.PHONY: build-all
build-all: build docker-build-subagent
	@echo "All artifacts built"
```

## Команды

### Сборка

```bash
# Собрать образ
make docker-build-subagent

# С конкретным тегом
make docker-build-subagent IMAGE_TAG=v1.0.0

# Multi-architecture (amd64 + arm64)
make docker-build-subagent-multiarch
```

### Публикация

```bash
# Push в registry
make docker-push-subagent

# С конкретным тегом
make docker-push-subagent IMAGE_TAG=v1.0.0

# С registry
make docker-push-subagent DOCKER_REGISTRY=ghcr.io/myorg
```

### Тегирование

```bash
# Создать новый тег
make docker-tag-subagent NEW_TAG=v1.0.0
```

### Тестирование

```bash
# Быстрый тест образа
make docker-test-subagent

# Инспекция
make docker-inspect-subagent
```

## CI/CD интеграция

### GitHub Actions

```yaml
# .github/workflows/docker.yml
name: Docker Build

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build and push
        run: |
          make docker-build-subagent-multiarch \
            DOCKER_REGISTRY=ghcr.io/${{ github.repository_owner }} \
            IMAGE_TAG=${{ github.ref_name }}
```

## Ключевые решения

1. **Отдельный makefile** — изоляция Docker команд
2. **Переменные для гибкости** — registry, image name, tag
3. **Multi-arch support** — buildx для amd64/arm64
4. **Test target** — быстрая проверка образа
5. **CI-ready** — интеграция с GitHub Actions
