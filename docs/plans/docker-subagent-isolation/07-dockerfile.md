# Этап 7: Dockerfile

## Цель

Создание минимального и безопасного Docker-образа для сабагента.

## Файлы

### `Dockerfile.subagent`

```dockerfile
# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Кэширование зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходников
COPY . .

# Сборка с оптимизациями
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /subagent ./cmd/subagent

# Stage 2: Runtime
FROM alpine:3.19

# Build arg для debug инструментов (v13)
ARG INCLUDE_DEBUG_TOOLS=false

# Минимальные зависимости
RUN apk --no-cache add ca-certificates procps && \
    if [ "$INCLUDE_DEBUG_TOOLS" = "true" ]; then \
        apk --no-cache add curl; \
    fi

# Создание непривилегированного пользователя
RUN adduser -D -u 1000 subagent

WORKDIR /workspace

# Создание директорий с правильными правами
RUN mkdir -p /workspace/skills && chown -R subagent:subagent /workspace

RUN mkdir -p /usr/local/share/subagent/prompts && \
    chown -R subagent:subagent /usr/local/share/subagent

# Копирование промптов
COPY prompts/identity.md /usr/local/share/subagent/prompts/
COPY prompts/security.md /usr/local/share/subagent/prompts/

# Копирование бинарника
COPY --from=builder /subagent /usr/local/bin/subagent
RUN chmod +x /usr/local/bin/subagent

# Переключение на непривилегированного пользователя
USER subagent

# Environment variables
ENV SKILLS_PATH=/workspace/skills

# HEALTHCHECK через process check (pgrep)
# HTTP сервера нет, поэтому проверяем процесс
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD pgrep -f "subagent" || exit 1

# Entrypoint
ENTRYPOINT ["/usr/local/bin/subagent"]
```

## Структура файлов в образе

```
/
├── usr/
│   ├── local/
│   │   ├── bin/
│   │   │   └── subagent          # Бинарник
│   │   └── share/
│   │       └── subagent/
│   │           └── prompts/
│   │               ├── identity.md
│   │               └── security.md
│   └── share/
│       └── ca-certificates/      # SSL сертификаты
└── workspace/
    └── skills/                   # Read-only mount из хоста
```

## Размер образа

```
REPOSITORY          TAG       SIZE
nexbot/subagent     latest    ~15MB
```

Оптимизации:
- Multi-stage build
- `-ldflags="-s -w"` — удаление debug info
- Alpine base
- Только необходимые пакеты

## Сборка

```bash
# Локальная сборка (production)
docker build -f Dockerfile.subagent -t nexbot/subagent:latest .

# С debug инструментами (v13)
docker build -f Dockerfile.subagent --build-arg INCLUDE_DEBUG_TOOLS=true -t nexbot/subagent:debug .

# С конкретным тегом
docker build -f Dockerfile.subagent -t nexbot/subagent:v1.0.0 .

# С build args
docker build -f Dockerfile.subagent \
    --build-arg GO_VERSION=1.26 \
    -t nexbot/subagent:latest .
```

## Тестирование образа

```bash
# Запуск интерактивно
docker run --rm -it nexbot/subagent:latest

# С mount skills
docker run --rm -it \
    -v ~/.nexbot/skills:/workspace/skills:ro \
    nexbot/subagent:latest

# Проверка health
docker ps
docker inspect <container-id> | jq '.[0].State.Health'
```

## Безопасность

### Непривилегированный пользователь

```dockerfile
RUN adduser -D -u 1000 subagent
USER subagent
```

### Read-only mount для skills

```bash
-v ~/.nexbot/skills:/workspace/skills:ro
```

### Resource limits (в PoolConfig)

```go
MemoryLimit: "128m",
CPULimit:    0.5,
PidsLimit:   50,
```

### Security Options (MVP)

```go
// В client.go CreateContainer:
HostConfig: &container.HostConfig{
    Resources: container.Resources{
        Memory:    memoryLimit,
        NanoCPUs:  int64(cpuLimit * 1e9),
        PidsLimit: &pidsLimit,
    },
    Mounts:         mounts,
    SecurityOpt:    []string{"no-new-privileges"},
    ReadonlyRootfs: true,
    Tmpfs:          map[string]string{"/tmp": "rw,size=50m"},  // Для web_fetch temp files
},
```

### tmpfs для /tmp

При `ReadonlyRootfs=true` web_fetch не может писать временные файлы. Решение: tmpfs mount:

```go
Tmpfs: map[string]string{"/tmp": "rw,size=50m"},
```

### Нет network expose

Образ не экспонирует порты. Все общение через stdin/stdout.

## Ключевые решения

1. **Multi-stage build** — минимальный размер образа
2. **Alpine base** — минимальные зависимости
3. **Непривилегированный пользователь** — безопасность
4. **Process HEALTHCHECK** — pgrep вместо HTTP
5. **procps пакет** — для pgrep команды
6. **ca-certificates** — для HTTPS запросов
