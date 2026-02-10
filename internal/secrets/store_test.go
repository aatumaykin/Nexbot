package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "success",
			sessionID: "telegram:123456",
			plaintext: "my_secret_value",
			wantErr:   false,
		},
		{
			name:      "empty sessionID",
			sessionID: "",
			plaintext: "value",
			wantErr:   true,
		},
		{
			name:      "empty plaintext",
			sessionID: "session123",
			plaintext: "",
			wantErr:   true,
		},
		{
			name:      "special characters",
			sessionID: "session:123",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:   false,
		},
		{
			name:      "unicode",
			sessionID: "session",
			plaintext: "привет мир",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := Encrypt(tt.sessionID, tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Decrypt
			decrypted, err := Decrypt(tt.sessionID, ciphertext)
			if err != nil {
				t.Errorf("Decrypt() error = %v", err)
				return
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDecryptWithDifferentSessionID(t *testing.T) {
	sessionID1 := "session1"
	sessionID2 := "session2"
	plaintext := "secret_value"

	// Encrypt with sessionID1
	ciphertext, err := Encrypt(sessionID1, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with sessionID2 (should fail)
	_, err = Decrypt(sessionID2, ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with different sessionID")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		ciphertext  []byte
		wantErr     bool
		errContains string
	}{
		{
			name:       "empty ciphertext",
			sessionID:  "session",
			ciphertext: []byte{},
			wantErr:    true,
		},
		{
			name:       "too short",
			sessionID:  "session",
			ciphertext: []byte{0x01, 0x02},
			wantErr:    true,
		},
		{
			name:       "invalid data",
			sessionID:  "session",
			ciphertext: []byte("invalid"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.sessionID, tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	sessionID := "telegram:123456"

	t.Run("Put and Get", func(t *testing.T) {
		name := "API_KEY"
		value := "sk-1234567890abcdef"

		// Put
		err := store.Put(sessionID, name, value)
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		// Get
		got, err := store.Get(sessionID, name)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got != value {
			t.Errorf("Get() = %v, want %v", got, value)
		}

		// Check file permissions
		secretFile := filepath.Join(tmpDir, sanitizeSessionID(sessionID), sanitizeName(name)+".enc")
		info, err := os.Stat(secretFile)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}

		// Check file permissions (0600)
		if info.Mode().Perm() != 0600 {
			t.Errorf("File permissions = %v, want 0600", info.Mode().Perm())
		}
	})

	t.Run("Get not found", func(t *testing.T) {
		_, err := store.Get(sessionID, "NOT_FOUND")
		if err != ErrSecretNotFound {
			t.Errorf("Get() error = %v, want %v", err, ErrSecretNotFound)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Add multiple secrets
		secrets := map[string]string{
			"API_KEY":  "key1",
			"TOKEN":    "token1",
			"PASSWORD": "pass1",
		}

		for name, value := range secrets {
			err := store.Put(sessionID, name, value)
			if err != nil {
				t.Fatalf("Put() error = %v", err)
			}
		}

		// List
		names, err := store.List(sessionID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(names) != len(secrets) {
			t.Errorf("List() count = %v, want %v", len(names), len(secrets))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		name := "TO_DELETE"
		value := "value"

		// Put
		err := store.Put(sessionID, name, value)
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		// Delete
		err = store.Delete(sessionID, name)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deletion
		_, err = store.Get(sessionID, name)
		if err != ErrSecretNotFound {
			t.Errorf("Get() after delete error = %v, want %v", err, ErrSecretNotFound)
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		err := store.Delete(sessionID, "NOT_FOUND")
		if err != ErrSecretNotFound {
			t.Errorf("Delete() error = %v, want %v", err, ErrSecretNotFound)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		// Add secrets
		err := store.Put(sessionID, "KEY1", "value1")
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		err = store.Put(sessionID, "KEY2", "value2")
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		// Clear
		err = store.Clear(sessionID)
		if err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		// Verify cleared
		names, err := store.List(sessionID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(names) != 0 {
			t.Errorf("List() after clear = %v, want empty", names)
		}
	})

	t.Run("Empty sessionID", func(t *testing.T) {
		err := store.Put("", "name", "value")
		if err != ErrInvalidSessionID {
			t.Errorf("Put() error = %v, want %v", err, ErrInvalidSessionID)
		}

		_, err = store.Get("", "name")
		if err != ErrInvalidSessionID {
			t.Errorf("Get() error = %v, want %v", err, ErrInvalidSessionID)
		}
	})

	t.Run("Empty name", func(t *testing.T) {
		err := store.Put(sessionID, "", "value")
		if err != ErrInvalidSecretName {
			t.Errorf("Put() error = %v, want %v", err, ErrInvalidSecretName)
		}

		_, err = store.Get(sessionID, "")
		if err != ErrInvalidSecretName {
			t.Errorf("Get() error = %v, want %v", err, ErrInvalidSecretName)
		}
	})

	t.Run("Session isolation", func(t *testing.T) {
		session1 := "session1"
		session2 := "session2"

		// Put secret for session1
		err := store.Put(session1, "SHARED", "value1")
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		// Try to get from session2
		_, err = store.Get(session2, "SHARED")
		if err != ErrSecretNotFound {
			t.Errorf("Get() from different session error = %v, want %v", err, ErrSecretNotFound)
		}
	})
}

func TestResolver(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	resolver := NewResolver(store)

	sessionID := "telegram:123456"

	// Add secrets
	secrets := map[string]string{
		"API_KEY":  "sk-1234567890",
		"TOKEN":    "my_token_value",
		"PASSWORD": "secret123",
	}

	for name, value := range secrets {
		err := store.Put(sessionID, name, value)
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	tests := []struct {
		name      string
		sessionID string
		text      string
		want      string
	}{
		{
			name:      "single secret",
			sessionID: sessionID,
			text:      "curl -H \"Authorization: Bearer $TOKEN\"",
			want:      "curl -H \"Authorization: Bearer my_token_value\"",
		},
		{
			name:      "multiple secrets",
			sessionID: sessionID,
			text:      "API_KEY=$API_KEY TOKEN=$TOKEN",
			want:      "API_KEY=sk-1234567890 TOKEN=my_token_value",
		},
		{
			name:      "secret not found",
			sessionID: sessionID,
			text:      "value=$UNKNOWN",
			want:      "value=***SECRET_NOT_FOUND***",
		},
		{
			name:      "no secrets",
			sessionID: sessionID,
			text:      "no secrets here",
			want:      "no secrets here",
		},
		{
			name:      "empty text",
			sessionID: sessionID,
			text:      "",
			want:      "",
		},
		{
			name:      "complex command",
			sessionID: sessionID,
			text:      "curl -u $API_KEY:$PASSWORD https://api.example.com",
			want:      "curl -u sk-1234567890:secret123 https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.Resolve(tt.sessionID, tt.text)
			if got != tt.want {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskSecrets(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		secretNames []string
		want        string
	}{
		{
			name:        "single secret",
			text:        "value=$API_KEY",
			secretNames: []string{"API_KEY"},
			want:        "value=***",
		},
		{
			name:        "multiple secrets",
			text:        "$API_KEY and $TOKEN",
			secretNames: []string{"API_KEY", "TOKEN"},
			want:        "*** and ***",
		},
		{
			name:        "no secrets",
			text:        "no secrets here",
			secretNames: []string{},
			want:        "no secrets here",
		},
		{
			name:        "partial match",
			text:        "MY_API_KEY value",
			secretNames: []string{"API_KEY"},
			want:        "MY_API_KEY value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSecrets(tt.text, tt.secretNames)
			if got != tt.want {
				t.Errorf("MaskSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		secretName string
		wantDir    string
		wantFile   string
	}{
		{
			name:       "normal",
			sessionID:  "telegram:123456",
			secretName: "API_KEY",
			wantDir:    "telegram:123456",
			wantFile:   "API_KEY",
		},
		{
			name:       "path traversal",
			sessionID:  "../etc/passwd",
			secretName: "../../secret",
			wantDir:    "__etc_passwd",
			wantFile:   "____secret",
		},
		{
			name:       "special chars",
			sessionID:  "session/with\\slashes",
			secretName: "name with spaces",
			wantDir:    "session_with_slashes",
			wantFile:   "name_with_spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir := sanitizeSessionID(tt.sessionID)
			if gotDir != tt.wantDir {
				t.Errorf("sanitizeSessionID() = %v, want %v", gotDir, tt.wantDir)
			}

			gotFile := sanitizeName(tt.secretName)
			if gotFile != tt.wantFile {
				t.Errorf("sanitizeName() = %v, want %v", gotFile, tt.wantFile)
			}
		})
	}
}
