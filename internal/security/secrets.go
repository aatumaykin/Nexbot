package security

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// zeroBytes securely zeroes a byte slice
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

const (
	MaxSecretsCount    = 50
	MaxSecretKeyLength = 64
	MaxSecretValueSize = 64 * 1024
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
	zeroBytes(s.data)
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
	defer zeroBytes(val)

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
		zeroBytes(val)
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
