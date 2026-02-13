package security

import (
	"testing"
	"time"
)

func TestSecret_ValueReturnsCopy(t *testing.T) {
	s := NewSecret("secret-value", 5*time.Minute)
	val, _ := s.Value()
	val[0] = 'X'
	val2, _ := s.Value()
	if val2[0] == 'X' {
		t.Error("Value() should return a copy")
	}
}

func TestSecret_Clear(t *testing.T) {
	s := NewSecret("value", 5*time.Minute)
	s.Clear()
	s.Clear() // No panic
	_, err := s.Value()
	if err == nil {
		t.Error("expected error after clear")
	}
}

func TestSecret_Expired(t *testing.T) {
	s := NewSecret("value", 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	_, err := s.Value()
	if err == nil {
		t.Error("expected error after expiration")
	}
}

func TestSecretsStore_SetAndGet(t *testing.T) {
	store := NewSecretsStore(5 * time.Minute)
	store.SetAll(map[string]string{"KEY": "value"})
	val, err := store.Get("KEY")
	if err != nil || val != "value" {
		t.Errorf("expected value, got %s, err %v", val, err)
	}
}

func TestSecretsStore_GetNotFound(t *testing.T) {
	store := NewSecretsStore(5 * time.Minute)
	_, err := store.Get("NONEXISTENT")
	if err == nil {
		t.Error("expected error for nonexistent key")
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
