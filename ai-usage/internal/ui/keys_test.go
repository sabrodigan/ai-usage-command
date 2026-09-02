package ui

import (
	"bytes"
	"strings"
	"testing"

	"ai-usage/internal/config"
)

func TestRunKeysManager_Interactive(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"codex": {
				DisplayName: "OpenAI Codex",
				APIKey:      "sk-proj-test1234567890abcdef",
				ModelTier:   "GPT-4o",
			},
			"claude": {
				DisplayName: "Anthropic Claude",
				APIKey:      "sk-ant-api03-testkey987654321",
				ModelTier:   "Claude 3.5 Sonnet",
			},
			"antigravity": {
				DisplayName: "Antigravity Agent",
				ModelTier:   "Agent Pro",
			},
		},
	}

	// Simulate user entering:
	// "2" (reveal second key - claude), then "0" (exit)
	in := bytes.NewBufferString("2\n0\n")
	var out bytes.Buffer

	RunKeysManager(in, &out, cfg)
	output := out.String()

	if !strings.Contains(output, "SECURE AI CREDENTIALS & API KEY MANAGER") {
		t.Errorf("expected keys manager header in output")
	}
	if !strings.Contains(output, "Anthropic Claude") || !strings.Contains(output, "OpenAI Codex") {
		t.Errorf("expected provider names in output")
	}
	if !strings.Contains(output, "REVEALED") {
		t.Errorf("expected key to be revealed after input 1")
	}
	if !strings.Contains(output, "Secure credentials session closed") {
		t.Errorf("expected exit message")
	}
}

func TestMaskSecret(t *testing.T) {
	if MaskSecret("") != "" {
		t.Errorf("expected empty string for empty key")
	}
	if MaskSecret("short") != "••••••••" {
		t.Errorf("expected bullet mask for short key, got %s", MaskSecret("short"))
	}
	long := "sk-ant-api03-testkey9876543210"
	masked := MaskSecret(long)
	if !strings.HasPrefix(masked, "sk-ant") || !strings.HasSuffix(masked, "43210"[len("43210")-4:]) {
		t.Errorf("expected masked prefix/suffix, got %s", masked)
	}
}
