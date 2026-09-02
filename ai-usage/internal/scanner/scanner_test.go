package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-usage/internal/config"
	"ai-usage/internal/core"
)

func TestScanMachineWithDiff(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create fake anthropic env
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key-1234567890")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	// Create fake codex file
	codexDir := filepath.Join(tempDir, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	_ = os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(`{"OPENAI_API_KEY":"sk-test-openai-secret"}`), 0600)

	cfg := &config.Config{
		AnchorBillingDay: 1,
		Providers: map[string]config.ProviderConfig{
			"codex": {
				ID:               "codex",
				DisplayName:      "OpenAI Codex",
				CustomQuota:      4_000_000,
				AnchorBillingDay: 1,
				Enabled:          true,
			},
		},
	}
	_ = cfg.UpdateKey("codex", "sk-test-openai-secret")

	scanned := ScanMachineWithDiff(cfg)
	if len(scanned) == 0 {
		t.Fatalf("expected scanned credentials, got 0")
	}

	var foundCodex, foundClaude *ScannedCredential
	for i := range scanned {
		if scanned[i].ProviderID == "codex" {
			foundCodex = &scanned[i]
		}
		if scanned[i].ProviderID == "claude" {
			foundClaude = &scanned[i]
		}
	}

	if foundCodex == nil {
		t.Errorf("expected codex in scanned items")
	} else if foundCodex.Status != StatusConfigured {
		t.Errorf("expected codex to have status StatusConfigured, got %v", foundCodex.Status)
	}

	if foundClaude == nil {
		t.Errorf("expected claude in scanned items")
	} else if foundClaude.Status != StatusNew {
		t.Errorf("expected claude to have status StatusNew, got %v", foundClaude.Status)
	}

	// Now test updated key status
	_ = cfg.SetProvider(config.ProviderConfig{
		ID:          "claude",
		DisplayName: "Anthropic Claude",
		APIKey:      "sk-old-key",
		CustomQuota: 5_000_000,
		Enabled:     true,
	})
	scanned2 := ScanMachineWithDiff(cfg)
	for _, c := range scanned2 {
		if c.ProviderID == "claude" {
			if c.Status != StatusUpdatedKey {
				t.Errorf("expected claude to have StatusUpdatedKey after key change, got %v", c.Status)
			}
		}
	}
}

func TestFilterNewCredentials(t *testing.T) {
	list := []ScannedCredential{
		{ProviderID: "a", Status: StatusConfigured},
		{ProviderID: "b", Status: StatusNew},
		{ProviderID: "c", Status: StatusUpdatedKey},
		{ProviderID: "d", Status: StatusConfigured},
	}

	filtered := FilterNewCredentials(list)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 new/updated credentials, got %d", len(filtered))
	}
	if filtered[0].ProviderID != "b" || filtered[1].ProviderID != "c" {
		t.Errorf("unexpected filtered credentials: %+v", filtered)
	}
}

func TestImportSingleCredential(t *testing.T) {
	cfg := config.DefaultConfig()

	cred := ScannedCredential{
		ProviderID:        "deepseek",
		DisplayName:       "DeepSeek AI",
		RawKey:            "sk-deepseek-test-key-9988",
		ModelTier:         "DeepSeek-V3",
		Endpoint:          "https://api.deepseek.com",
		SuggestedCycleDay: 15,
		DefaultQuota:      10_000_000,
		Unit:              core.UnitTokens,
	}

	err := ImportSingleCredential(cfg, cred, 8_000_000, 14)
	if err != nil {
		t.Fatalf("ImportSingleCredential returned error: %v", err)
	}

	p, exists := cfg.Providers["deepseek"]
	if !exists {
		t.Fatalf("expected deepseek in cfg.Providers")
	}
	if p.CustomQuota != 8_000_000 {
		t.Errorf("expected quota 8M, got %v", p.CustomQuota)
	}
	if p.AnchorBillingDay != 14 {
		t.Errorf("expected cycle day 14, got %v", p.AnchorBillingDay)
	}
	if p.GetDecryptedKey() != "sk-deepseek-test-key-9988" {
		t.Errorf("expected decrypted key to match, got %v", p.GetDecryptedKey())
	}
	if p.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt")
	}
}

func TestImportCredentials(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfig()
	creds := []ScannedCredential{
		{
			ProviderID:        "groq",
			DisplayName:       "Groq Cloud",
			RawKey:            "gsk-test-key-1122",
			ModelTier:         "Llama 3.3",
			SuggestedCycleDay: 1,
			DefaultQuota:      10_000_000,
		},
		{
			ProviderID:        "mistral",
			DisplayName:       "Mistral AI",
			RawKey:            "mistral-test-key-3344",
			ModelTier:         "Codestral",
			SuggestedCycleDay: 5,
			DefaultQuota:      5_000_000,
		},
	}

	count, err := ImportCredentials(cfg, creds)
	if err != nil {
		t.Fatalf("ImportCredentials returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	if _, ok := cfg.Providers["groq"]; !ok {
		t.Errorf("expected groq to be imported")
	}
	if _, ok := cfg.Providers["mistral"]; !ok {
		t.Errorf("expected mistral to be imported")
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "***"},
		{"short", "***"},
		{"123456789012345", "1234...45"},
		{"sk-ant-api03-abcdefghijklmnop", "sk-ant...mnop"},
	}

	for _, tt := range tests {
		got := maskKey(tt.input)
		if tt.input == "" && got != "***" {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Silence unused package warning
var _ = time.Now
