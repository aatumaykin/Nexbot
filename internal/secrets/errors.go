// Package secrets provides secure storage and management of sensitive data (passwords, tokens).
// Secrets are encrypted using AES-256-GCM with sessionID as the encryption key.
// Each session has isolated storage, and secrets are never exposed to LLM context.
//
// Key features:
//   - AES-256-GCM encryption with sessionID as key
//   - Session-isolated storage
//   - Secret resolution in tools (substitution)
//   - Secure file permissions (0600)
//   - Secrets never logged in plaintext
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrSecretNotFound is returned when a secret does not exist
	ErrSecretNotFound = errors.New("secret not found")

	// ErrSecretAccessDenied is returned when access to a secret is denied
	ErrSecretAccessDenied = errors.New("secret access denied")

	// ErrInvalidSessionID is returned when sessionID is empty
	ErrInvalidSessionID = errors.New("session ID cannot be empty")

	// ErrInvalidSecretName is returned when secret name is empty
	ErrInvalidSecretName = errors.New("secret name cannot be empty")

	// ErrInvalidCiphertext is returned when ciphertext is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// Secret represents an encrypted secret with its metadata.
type Secret struct {
	Name       string
	Ciphertext []byte
}

// deriveKey derives a 256-bit key from sessionID using SHA-256.
// This ensures that the key length is exactly what AES-256 requires (32 bytes).
func deriveKey(sessionID string) ([]byte, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	hash := sha256.Sum256([]byte(sessionID))
	return hash[:], nil
}

// Encrypt encrypts plaintext using AES-256-GCM with sessionID as the key.
// Returns the ciphertext (nonce + encrypted data).
// Format: nonce (12 bytes) + ciphertext
func Encrypt(sessionID, plaintext string) ([]byte, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}
	if plaintext == "" {
		return nil, errors.New("plaintext cannot be empty")
	}

	// Derive key from sessionID
	key, err := deriveKey(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce (12 bytes as recommended by GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and append nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with sessionID as the key.
// Expected format: nonce (12 bytes) + ciphertext
func Decrypt(sessionID string, ciphertext []byte) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidSessionID
	}
	if len(ciphertext) == 0 {
		return "", ErrInvalidCiphertext
	}

	// Derive key from sessionID
	key, err := deriveKey(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check if ciphertext has enough bytes for nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// Split nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (invalid key or corrupted data): %w", err)
	}

	return string(plaintext), nil
}
