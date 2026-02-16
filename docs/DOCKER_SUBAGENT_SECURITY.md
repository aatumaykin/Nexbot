# Docker Subagent Security

## Overview

Docker subagent provides secure isolation for executing tasks with secrets (API keys, tokens) and processing external content.

## Secret Transmission

### How Secrets Are Passed

**Secrects are transmitted ONLY via stdin:**

```
Host Process                    Docker Container
     |                               |
     |---> SubagentRequest           |
     |    - LLMAPIKey                 |
     |    - Secrets[]                 |
     |                               v
     |                          Read from stdin
     |                               |
     |                    Initialize LLM with key
     |                    Store secrets in SecretsStore
     |                               |
     |                    Process task
     |                               |
     |                    Clear secrets (zeroing)
     |                               v
     |                          Response
```

### What's NOT Transmitted via Env

**Container environment variables contain ONLY:**
- `SKILLS_PATH=/workspace/skills`

**NOT present:**
- `ZAI_API_KEY`
- `OPENAI_API_KEY`
- Any task secrets
- `Environment` configuration (removed)

### Why Not Use Environment Variables?

**Security Risk:**
```bash
# If secrets were in env:
docker inspect <container> | grep -A 20 "Env"
# Would show: ZAI_API_KEY=sk-...  <-- LEAKED!

# After our changes:
docker inspect <container> | grep -A 5 "Env"
# Shows only: SKILLS_PATH=/workspace/skills
```

**Verification:**
```bash
# Test: No secrets visible in docker inspect
docker ps -q | head -1 | xargs docker inspect | grep -i "ZAI_API_KEY\|SECRET\|TOKEN"
# Expected: Empty output
```

## Implementation Details

### 1. Stdin Protocol

**Host side** (`internal/docker/docker_spawn.go`):
```go
req := &SubagentRequest{
    LLMAPIKey: os.Getenv(cfg.LLMAPIKeyEnv),  // Read from host env
    Secrets:  secretsFilter(task),
}

// Write request to container stdin
json.NewEncoder(containerStdin).Encode(req)
```

**Container side** (`cmd/nexbot/subagent.go`):
```go
var req SubagentRequest
if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
    log.Fatal(err)
}

// Initialize LLM with key from stdin
if req.LLMAPIKey != "" {
    subagent.InitLLM(req.LLMAPIKey)
}

// Store secrets with TTL and zeroing
for k, v := range req.Secrets {
    secretsStore.Set(k, v, 5*time.Minute)
}
```

### 2. SecretsStore with Zeroing

**Security features:**
- Secrets stored in memory with 5-minute TTL
- Automatic zeroing on expiration
- Secure storage via `internal/security.SecretsStore`

```go
// cmd/nexbot/subagent.go
secretsStore := security.NewSecretsStore(5 * time.Minute)

// When secret expires:
// 1. Value is zeroed (bytes set to 0)
// 2. Removed from map
// 3. No trace in memory
```

### 3. Thread-Safety

**Mutex-protected container operations:**
```go
// pool.go - CreateContainer no longer writes to map
func (p *ContainerPool) CreateContainer(ctx context.Context) (*Container, error) {
    // ... create container ...
    return container, nil  // Return container, caller writes to map
}

// pool.go - Caller writes under mutex
func (p *ContainerPool) acquire() (*Container, error) {
    container, err := p.CreateContainer(ctx)
    if err != nil {
        return nil, err
    }

    p.mu.Lock()
    p.containers[container.ID] = container  // Safe write
    p.mu.Unlock()
}
```

**Atomic status updates:**
```go
// types.go - Container status is atomic
type Container struct {
    ID     string
    status atomic.Int32  // Thread-safe
}

func (c *Container) GetStatus() ContainerStatus {
    return ContainerStatus(c.status.Load())
}

func (c *Container) SetStatus(s ContainerStatus) {
    c.status.Store(int32(s))
}
```

## Configuration

### Docker Config (config.toml)

```toml
[docker]
enabled = true

# API key environment variable on HOST (read by nexbot, NOT passed to container)
llm_api_key_env = "ZAI_API_KEY"

# Secrets TTL (in seconds)
secrets_ttl_seconds = 300  # 5 minutes

# ... other settings ...
```

**Important:**
- `llm_api_key_env` specifies variable to read on HOST
- This value is NOT passed to container env
- It's read from host os.Getenv() and sent via stdin

### Removed Configuration

**Deprecated (removed in v0.17+):**
```toml
# NO LONGER SUPPORTED:
[docker]
# environment = ["VAR=value"]  # ❌ REMOVED
```

**Reason:**
- Environment variables are visible in `docker inspect`
- Security risk for secret leakage
- Use stdin protocol instead

## Testing

### Security Tests

**Unit test:**
```bash
go test -v ./internal/docker -run TestCreateContainerEnv
```

Verifies: Container env contains only `SKILLS_PATH`.

**Integration test:**
```bash
go test -v ./internal/docker -run TestSecretsNotInEnv
```

Verifies: No secrets visible in `docker inspect`.

**Race detector:**
```bash
go test -race ./internal/docker
```

Verifies: No race conditions in concurrent access.

### Manual Verification

```bash
# 1. Start a task that spawns a container
# (via Telegram or API)

# 2. Find the running container
CONTAINER_ID=$(docker ps -q | head -1)

# 3. Check container environment
docker inspect $CONTAINER_ID | jq '.[0].Config.Env'
# Expected: ["SKILLS_PATH=/workspace/skills"]

# 4. Verify NO secrets:
docker inspect $CONTAINER_ID | grep -i "ZAI_API_KEY\|SECRET\|TOKEN"
# Expected: Empty output
```

## Best Practices

### For Users

1. **Set API key as environment variable on HOST:**
   ```bash
   export ZAI_API_KEY="sk-..."
   ```

2. **NEVER pass secrets via Docker config:**
   ```toml
   # ❌ WRONG:
   [docker]
   environment = ["API_KEY=sk-..."]  # Deprecated anyway

   # ✅ CORRECT:
   [docker]
   llm_api_key_env = "ZAI_API_KEY"  # Read from host env
   ```

3. **Verify security after changes:**
   ```bash
   # Check running containers
   docker ps

   # Inspect environment
   docker ps -q | xargs docker inspect | grep -A 5 "Env"
   ```

### For Developers

1. **Always pass secrets via stdin:**
   ```go
   // ✅ CORRECT:
   req.LLMAPIKey = os.Getenv(cfg.LLMAPIKeyEnv)
   json.NewEncoder(stdin).Encode(req)

   // ❌ WRONG:
   env = append(env, fmt.Sprintf("ZAI_API_KEY=%s", apiKey))
   ```

2. **Use atomic types for shared state:**
   ```go
   // ✅ CORRECT:
   status atomic.Int32

   // ❌ WRONG:
   status string  // Not thread-safe
   ```

3. **Protect map writes with mutex:**
   ```go
   p.mu.Lock()
   p.containers[id] = container
   p.mu.Unlock()
   ```

## Troubleshooting

### Container Starts but Tasks Fail

**Symptom:** Container starts, but LLM initialization fails.

**Cause:** API key not reaching container.

**Check:**
1. Verify `llm_api_key_env` is set in config.toml
2. Verify environment variable is set on host:
   ```bash
   echo $ZAI_API_KEY
   ```
3. Check logs for stdin read errors

### Secrets Visible in Docker Inspect

**Symptom:** API keys visible in `docker inspect`.

**Cause:** Old configuration or outdated binary.

**Fix:**
1. Update config.toml (remove `environment` section)
2. Rebuild binary: `make build`
3. Restart nexbot

## References

- **Implementation:** `internal/docker/docker_spawn.go`, `cmd/nexbot/subagent.go`
- **SecretsStore:** `internal/security/secrets.go`
- **Tests:** `internal/docker/secrets_test.go`, `internal/docker/client_test.go`
- **Plan:** `docs/plans/docker-subagent-isolation/secrets-stdin-fix.md`

## Version History

- **v0.17+**: Secrets transmitted only via stdin (no env leakage)
- **v0.16**: Mixed stdin + env (deprecated)
- **v0.15**: Env-only (insecure)
