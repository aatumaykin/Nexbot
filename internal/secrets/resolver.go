package secrets

import (
	"fmt"
	"strings"
)

const (
	// secretPrefix is the prefix used to identify secret references in text.
	secretPrefix = "$"

	// secretNotFoundPlaceholder is the placeholder used when a secret is not found.
	secretNotFoundPlaceholder = "***SECRET_NOT_FOUND***"
)

// Resolver provides secret resolution functionality for tools.
// It replaces secret references ($SECRET_NAME) with their actual values.
type Resolver struct {
	store *Store
}

// NewResolver creates a new secret resolver.
func NewResolver(store *Store) *Resolver {
	return &Resolver{
		store: store,
	}
}

// Resolve resolves secret references in the given text.
// Secret references have the format $SECRET_NAME.
// If a secret is not found, it is replaced with ***SECRET_NOT_FOUND***.
func (r *Resolver) Resolve(sessionID, text string) string {
	if text == "" {
		return text
	}

	// Find all secret references
	// Pattern: $SECRET_NAME where SECRET_NAME is alphanumeric + underscore
	result := text
	pos := 0

	for pos < len(result) {
		// Find next '$'
		dollarPos := strings.Index(result[pos:], secretPrefix)
		if dollarPos == -1 {
			break // No more references
		}
		dollarPos += pos

		// Extract secret name (alphanumeric + underscore)
		secretName, endPos := extractSecretName(result, dollarPos+1)
		if secretName == "" {
			pos = dollarPos + 1
			continue
		}

		// Get secret value
		secretValue, err := r.store.Get(sessionID, secretName)
		if err != nil {
			// Secret not found, replace with placeholder
			secretValue = secretNotFoundPlaceholder
		}

		// Replace the reference with the value
		result = result[:dollarPos] + secretValue + result[endPos:]

		// Move past the replacement
		pos = dollarPos + len(secretValue)
	}

	return result
}

// extractSecretName extracts a secret name from text starting at the given position.
// Returns the secret name and the position after the name.
// Secret names are alphanumeric + underscore.
func extractSecretName(text string, startPos int) (string, int) {
	if startPos >= len(text) {
		return "", startPos
	}

	endPos := startPos
	for endPos < len(text) {
		c := text[endPos]
		// Allow alphanumeric and underscore
		if !isAlphaNumeric(c) && c != '_' {
			break
		}
		endPos++
	}

	if endPos == startPos {
		return "", startPos // No valid name
	}

	return text[startPos:endPos], endPos
}

// isAlphaNumeric checks if a byte is alphanumeric.
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}

// MaskSecrets masks secret values in text for logging purposes.
// Replaces secret values with ***.
func MaskSecrets(text string, secretNames []string) string {
	if text == "" || len(secretNames) == 0 {
		return text
	}

	result := text
	for _, name := range secretNames {
		// Create the reference pattern
		ref := fmt.Sprintf("%s%s", secretPrefix, name)

		// Replace all occurrences with ***
		result = replaceAll(result, ref, "***")
	}

	return result
}
