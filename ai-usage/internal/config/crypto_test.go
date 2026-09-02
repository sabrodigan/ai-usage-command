package config

import (
	"strings"
	"testing"
)

func TestCrypto_EncryptDecrypt(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	originalKey := "sk-ant-api03-secret-test-key-123456789"

	encrypted, err := EncryptString(originalKey)
	if err != nil {
		t.Fatalf("failed to encrypt key: %v", err)
	}

	if !strings.HasPrefix(encrypted, "enc:v1:") {
		t.Errorf("encrypted string should have enc:v1: prefix, got %s", encrypted)
	}
	if encrypted == originalKey {
		t.Errorf("ciphertext is identical to plaintext")
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt key: %v", err)
	}

	if decrypted != originalKey {
		t.Errorf("expected decrypted %s, got %s", originalKey, decrypted)
	}
}

func TestCrypto_PlainTextFallback(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	raw := "plain-unencrypted-key"
	decrypted, err := DecryptString(raw)
	if err != nil {
		t.Fatalf("unexpected error on plaintext fallback: %v", err)
	}
	if decrypted != raw {
		t.Errorf("expected %s, got %s", raw, decrypted)
	}
}
