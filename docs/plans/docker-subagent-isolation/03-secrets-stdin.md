# Этап 3: Передача секретов через stdin (MVP)

## Цель

Безопасная передача секретов и LLM API Key через stdin JSON вместо environment variables.

## Файлы

### `internal/security/secrets.go` (единый для pool и subagent)

```go
package security

import (
    "crypto/subtle"
    "fmt"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

const (
    MaxSecretsCount     = 50
    MaxSecretKeyLength  = 64
    MaxSecretValueSize  = 64 * 1024
)

type Secret struct {
    data      []byte
    expiresAt time.Time
    cleared   atomic.Bool
    mu        sync.RWMutex
}

func NewSecret(value string, ttl time.Duration) *Secret {
    return &Secret{
        data:      []byte(value),
        expiresAt: time.Now().Add(ttl),
    }
}

func (s *Secret) Value() ([]byte, error) {
    if s.cleared.Load() {
        return nil, fmt.Errorf("secret cleared")
    }
    if s.IsExpired() {
        return nil, fmt.Errorf("secret expired")
    }
    
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if s.data == nil {
        return nil, fmt.Errorf("secret cleared")
    }
    
    result := make([]byte, len(s.data))
    copy(result, s.data)
    return result, nil
}

func (s *Secret) Clear() {
    if s.cleared.Swap(true) {
        return
    }
    s.mu.Lock()
    subtle.ZeroBytes(s.data)
    s.data = nil
    s.mu.Unlock()
}

func (s *Secret) IsExpired() bool {
    return time.Now().After(s.expiresAt)
}

type SecretsStore struct {
    mu      sync.RWMutex
    secrets map[string]*Secret
    ttl     time.Duration
}

func NewSecretsStore(ttl time.Duration) *SecretsStore {
    if ttl == 0 {
        ttl = 5 * time.Minute
    }
    return &SecretsStore{
        secrets: make(map[string]*Secret),
        ttl:     ttl,
    }
}

func (s *SecretsStore) Get(key string) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    secret, ok := s.secrets[key]
    if !ok {
        return "", fmt.Errorf("secret not found: %s", key)
    }
    
    if secret.IsExpired() {
        return "", fmt.Errorf("secret expired: %s", key)
    }
    
    val, err := secret.Value()
    if err != nil {
        return "", err
    }
    defer subtle.ZeroBytes(val)
    
    return string(val), nil
}

func (s *SecretsStore) GetBytes(key string) ([]byte, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    secret, ok := s.secrets[key]
    if !ok {
        return nil, fmt.Errorf("secret not found: %s", key)
    }
    
    if secret.IsExpired() {
        return nil, fmt.Errorf("secret expired: %s", key)
    }
    
    return secret.Value()
}

func (s *SecretsStore) SetAll(secrets map[string]string) error {
    if len(secrets) > MaxSecretsCount {
        return fmt.Errorf("too many secrets: %d > %d", len(secrets), MaxSecretsCount)
    }
    
    for k, v := range secrets {
        if len(k) > MaxSecretKeyLength {
            return fmt.Errorf("secret key too long: %s", k)
        }
        if len(v) > MaxSecretValueSize {
            return fmt.Errorf("secret value too large: %s", k)
        }
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    newSecrets := make(map[string]*Secret)
    for k, v := range secrets {
        newSecrets[k] = NewSecret(v, s.ttl)
    }
    
    for _, secret := range s.secrets {
        secret.Clear()
    }
    
    s.secrets = newSecrets
    return nil
}

func (s *SecretsStore) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    for _, secret := range s.secrets {
        secret.Clear()
    }
    
    s.secrets = make(map[string]*Secret)
}

func (s *SecretsStore) ResolveSecrets(text string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    result := text
    for key, secret := range s.secrets {
        if secret.IsExpired() {
            continue
        }
        val, err := secret.Value()
        if err != nil {
            continue
        }
        result = strings.ReplaceAll(result, "$"+key, string(val))
        subtle.ZeroBytes(val)
    }
    return result
}

func (s *SecretsStore) Keys() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    keys := make([]string, 0, len(s.secrets))
    for k := range s.secrets {
        keys = append(keys, k)
    }
    return keys
}
```

### `internal/docker/secrets_filter.go`

```go
package docker

import (
    "fmt"
    
    "github.com/aatumaykin/nexbot/internal/security"
)

type SecretsFilter struct {
    store *security.SecretsStore
}

func NewSecretsFilter(store *security.SecretsStore) *SecretsFilter {
    return &SecretsFilter{store: store}
}

func (f *SecretsFilter) FilterForTask(requiredKeys []string) (map[string]string, error) {
    if len(requiredKeys) == 0 {
        return nil, nil
    }
    
    result := make(map[string]string)
    
    for _, key := range requiredKeys {
        val, err := f.store.Get(key)
        if err != nil {
            return nil, fmt.Errorf("required secret '%s' not found: %w", key, err)
        }
        result[key] = val
    }
    
    return result, nil
}
```

## Протокол передачи

### Запрос от оркестратора к сабагенту

```json
{
    "version": "1.0",
    "id": "task-uuid",
    "type": "execute",
    "task": "Fetch data from API using $API_KEY",
    "timeout": 60,
    "deadline": 1739356800,
    "secrets": {
        "API_KEY": "secret-api-key"
    },
    "llm_api_key": "zai-api-key"
}
```

### Ответ от сабагента

```json
{
    "id": "task-uuid",
    "version": "1.0",
    "status": "success",
    "result": "fetched data..."
}
```

## Безопасность

### Почему НЕ environment variables

```bash
# Environment variables видны через docker inspect
docker inspect container-id | jq '.[0].Config.Env'
# Выводит: ["API_KEY=secret-api-key"]
```

### Почему stdin JSON безопасен

```bash
# stdin не виден в docker inspect
docker inspect container-id | jq '.[0].Config'
# Нет секретов в выводе
```

### Почему []byte + ZeroBytes достаточно для MVP

1. **Memory dump требует root на хосте** — если злоумышленник уже на хосте, игра проиграна
2. **TTL 5min** — ограниченное время окна атаки
3. **crypto/subtle.ZeroBytes** — гарантированное обнуление (компилятор не оптимизирует)
4. **copy() при Value()** — caller получает копию, не ссылку

## Ключевые решения

1. **stdin вместо env** — защита от `docker inspect`
2. **TTL 5 минут** — автоочистка секретов
3. **[]byte вместо string** — контроль над памятью
4. **Value() возвращает копию** — caller владеет памятью
5. **crypto/subtle.ZeroBytes** — гарантированное обнуление
6. **Единый SecretsStore** — нет дублирования кода

## Тесты

### `internal/security/secrets_test.go`

```go
package security

import (
    "fmt"
    "strings"
    "testing"
    "time"
)

func TestSecret_ValueReturnsCopy(t *testing.T) {
    s := NewSecret("secret-value", 5*time.Minute)
    
    val, err := s.Value()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    val[0] = 'X'
    
    val2, _ := s.Value()
    if val2[0] == 'X' {
        t.Error("Value() should return a copy")
    }
}

func TestSecret_ClearNoPanic(t *testing.T) {
    s := NewSecret("value", 5*time.Minute)
    
    s.Clear()
    s.Clear() // Should not panic
    
    _, err := s.Value()
    if err == nil {
        t.Error("expected error after clear")
    }
}

func TestSecret_Expiration(t *testing.T) {
    s := NewSecret("value", 100*time.Millisecond)
    
    _, err := s.Value()
    if err != nil {
        t.Fatal("should work initially")
    }
    
    time.Sleep(150 * time.Millisecond)
    
    _, err = s.Value()
    if err == nil {
        t.Error("expected error after expiration")
    }
}

func TestSecretsStore_SetAndGet(t *testing.T) {
    store := NewSecretsStore(5 * time.Minute)
    
    err := store.SetAll(map[string]string{
        "KEY1": "value1",
        "KEY2": "value2",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    val, err := store.Get("KEY1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if val != "value1" {
        t.Errorf("expected value1, got %s", val)
    }
}

func TestSecretsStore_TooManySecrets(t *testing.T) {
    store := NewSecretsStore(5 * time.Minute)
    
    secrets := make(map[string]string)
    for i := 0; i < MaxSecretsCount+1; i++ {
        secrets[fmt.Sprintf("KEY%d", i)] = "value"
    }
    
    err := store.SetAll(secrets)
    if err == nil {
        t.Error("expected error for too many secrets")
    }
}

func TestSecretsStore_SecretTooLarge(t *testing.T) {
    store := NewSecretsStore(5 * time.Minute)
    
    largeValue := strings.Repeat("x", MaxSecretValueSize+1)
    err := store.SetAll(map[string]string{"KEY": largeValue})
    if err == nil {
        t.Error("expected error for oversized secret")
    }
}

func TestSecretsStore_Clear(t *testing.T) {
    store := NewSecretsStore(5 * time.Minute)
    
    store.SetAll(map[string]string{"KEY": "value"})
    store.Clear()
    
    _, err := store.Get("KEY")
    if err == nil {
        t.Error("expected error after clear")
    }
}

func TestSecretsStore_ResolveSecrets(t *testing.T) {
    store := NewSecretsStore(5 * time.Minute)
    
    store.SetAll(map[string]string{
        "API_KEY": "secret123",
    })
    
    text := "Connect using $API_KEY"
    result := store.ResolveSecrets(text)
    
    if result != "Connect using secret123" {
        t.Errorf("expected resolved text, got %s", result)
    }
}
```
