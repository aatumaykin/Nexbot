package secrets

import (
	"os"
	"path/filepath"
)

// Store provides secure storage for secrets with encryption and file-based persistence.
// Secrets are stored encrypted in the workspace directory, with each session having
// its own isolated storage.
type Store struct {
	secretsDir string
}

// NewStore creates a new secrets store.
// secretsDir is the base directory where secrets will be stored.
func NewStore(secretsDir string) *Store {
	return &Store{
		secretsDir: secretsDir,
	}
}

// Put stores a secret for the given sessionID and name.
// The secret is encrypted before being stored.
// File permissions are set to 0600 for security.
func (s *Store) Put(sessionID, name, value string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}
	if name == "" {
		return ErrInvalidSecretName
	}

	// Encrypt the secret
	ciphertext, err := Encrypt(sessionID, value)
	if err != nil {
		return err
	}

	// Create session directory if it doesn't exist
	sessionDir := filepath.Join(s.secretsDir, sanitizeSessionID(sessionID))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}

	// Write encrypted secret to file
	secretFile := filepath.Join(sessionDir, sanitizeName(name)+".enc")
	if err := os.WriteFile(secretFile, ciphertext, 0600); err != nil {
		return err
	}

	return nil
}

// Get retrieves a secret for the given sessionID and name.
// Returns the decrypted secret value.
func (s *Store) Get(sessionID, name string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidSessionID
	}
	if name == "" {
		return "", ErrInvalidSecretName
	}

	// Read encrypted secret file
	secretFile := filepath.Join(s.secretsDir, sanitizeSessionID(sessionID), sanitizeName(name)+".enc")
	ciphertext, err := os.ReadFile(secretFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	// Decrypt the secret
	plaintext, err := Decrypt(sessionID, ciphertext)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

// Delete removes a secret for the given sessionID and name.
func (s *Store) Delete(sessionID, name string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}
	if name == "" {
		return ErrInvalidSecretName
	}

	secretFile := filepath.Join(s.secretsDir, sanitizeSessionID(sessionID), sanitizeName(name)+".enc")
	if err := os.Remove(secretFile); err != nil {
		if os.IsNotExist(err) {
			return ErrSecretNotFound
		}
		return err
	}

	return nil
}

// List returns all secret names for the given sessionID.
// Does not return the secret values (only names).
func (s *Store) List(sessionID string) ([]string, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	sessionDir := filepath.Join(s.secretsDir, sanitizeSessionID(sessionID))

	// Read directory entries
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No secrets yet
		}
		return nil, err
	}

	// Extract secret names (remove .enc extension)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".enc" {
			names = append(names, name[:len(name)-4]) // Remove .enc
		}
	}

	return names, nil
}

// Clear removes all secrets for the given sessionID.
func (s *Store) Clear(sessionID string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	sessionDir := filepath.Join(s.secretsDir, sanitizeSessionID(sessionID))
	if err := os.RemoveAll(sessionDir); err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to clear
		}
		return err
	}

	return nil
}

// sanitizeSessionID sanitizes the sessionID for use as a directory name.
// Prevents path traversal attacks.
func sanitizeSessionID(sessionID string) string {
	// Replace directory separators with underscore
	// This prevents path traversal while keeping sessionID readable
	sanitized := sessionID
	for _, sep := range []string{"/", "\\", ".."} {
		sanitized = replaceAll(sanitized, sep, "_")
	}
	return sanitized
}

// sanitizeName sanitizes the secret name for use as a filename.
// Prevents path traversal attacks.
func sanitizeName(name string) string {
	sanitized := name
	for _, sep := range []string{"/", "\\", "..", ".", " ", "\n", "\r", "\t"} {
		sanitized = replaceAll(sanitized, sep, "_")
	}
	return sanitized
}

// replaceAll is a simple replacement function to avoid importing strings
func replaceAll(s, old, new string) string {
	result := ""
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}
