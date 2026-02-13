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
