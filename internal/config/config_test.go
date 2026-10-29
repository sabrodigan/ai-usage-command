package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfig_DefaultAndMutation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := DefaultConfig()
	if cfg.AnchorBillingDay != 1 {
		t.Errorf("expected default anchor 1, got %d", cfg.AnchorBillingDay)
	}
	if len(cfg.Providers) < 4 {
		t.Errorf("expected at least 4 default providers, got %d", len(cfg.Providers))
	}

	// Test adding a provider with plaintext key (should be stored encrypted)
	rawKey := "sk-test-secret-12345"
	err := cfg.SetProvider(ProviderConfig{
		ID:          "custom_deepseek",
		DisplayName: "DeepSeek API",
		APIKey:      rawKey,
		CustomQuota: 10_000_000,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, exists := cfg.Providers["custom_deepseek"]
	if !exists {
		t.Fatalf("custom provider not saved")
	}

	// Stored key must be encrypted
	if !strings.HasPrefix(p.APIKey, "enc:v1:") {
		t.Errorf("expected stored key to be encrypted, got %s", p.APIKey)
	}

	// Decrypted key must match original
	if p.GetDecryptedKey() != rawKey {
		t.Errorf("expected decrypted key %s, got %s", rawKey, p.GetDecryptedKey())
	}

	// Test UpdateKey
	newKey := "sk-test-updated-999"
	if err := cfg.UpdateKey("custom_deepseek", newKey); err != nil {
		t.Fatalf("failed to update key: %v", err)
	}
	if cfg.Providers["custom_deepseek"].GetDecryptedKey() != newKey {
		t.Errorf("expected updated key %s, got %s", newKey, cfg.Providers["custom_deepseek"].GetDecryptedKey())
	}

	// Test removing a provider
	removed := cfg.RemoveProvider("custom_deepseek")
	if !removed {
		t.Errorf("expected provider to be removed")
	}
	if _, exists := cfg.Providers["custom_deepseek"]; exists {
		t.Errorf("provider still exists after removal")
	}
}

func TestConfig_StalenessAndPrune(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := DefaultConfig()
	// Add an old provider with last activity 45 days ago
	oldDate := time.Now().UTC().AddDate(0, 0, -45)
	cfg.Providers["old_service"] = ProviderConfig{
		ID:           "old_service",
		DisplayName:  "Old Service",
		Enabled:      true,
		CreatedAt:    oldDate,
		LastActivity: oldDate,
	}

	p := cfg.Providers["old_service"]
	stale, days := p.IsStale()
	if !stale || days < 40 {
		t.Errorf("expected provider to be stale >= 40 days, got stale=%v, days=%d", stale, days)
	}

	// Test PruneStaleProviders
	pruned := cfg.PruneStaleProviders(30)
	if len(pruned) != 1 || pruned[0] != "old_service" {
		t.Errorf("expected old_service to be pruned, got %v", pruned)
	}
	if _, exists := cfg.Providers["old_service"]; exists {
		t.Errorf("old_service should not exist after pruning")
	}
}
