package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const encPrefix = "enc:v1:"

// getOrCreateMasterKey derives or retrieves a persistent 256-bit machine-local encryption key.
func getOrCreateMasterKey() ([]byte, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	keyFile := filepath.Join(dir, ".vault.key")
	if data, err := os.ReadFile(keyFile); err == nil && len(data) == 32 {
		return data, nil
	}

	// Generate a new cryptographically secure 256-bit key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		// Fallback to machine-derived entropy if rand.Reader fails
		h := sha256.New()
		h.Write([]byte(os.Getenv("USER") + os.Getenv("HOME") + "ai-usage-secure-salt-v1"))
		key = h.Sum(nil)
	}

	_ = os.WriteFile(keyFile, key, 0600)
	return key, nil
}

// EncryptString encrypts plaintext using AES-256-GCM and returns a base64 string with prefix.
func EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if strings.HasPrefix(plaintext, encPrefix) {
		return plaintext, nil // Already encrypted
	}

	key, err := getOrCreateMasterKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts an AES-256-GCM encrypted string.
func DecryptString(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	if !strings.HasPrefix(encrypted, encPrefix) {
		// Legacy / unencrypted string fallback
		return encrypted, nil
	}

	rawB64 := strings.TrimPrefix(encrypted, encPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encrypted payload: %w", err)
	}

	key, err := getOrCreateMasterKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}
