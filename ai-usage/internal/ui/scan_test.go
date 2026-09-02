package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ai-usage/internal/core"
	"ai-usage/internal/scanner"
)

func TestRenderScanAll(t *testing.T) {
	var buf bytes.Buffer
	scanned := []scanner.ScannedCredential{
		{
			ProviderID:        "deepseek",
			DisplayName:       "DeepSeek AI",
			SourceType:        "env",
			KeySnippet:        "sk-12...89",
			SuggestedCycleDay: 1,
			DefaultQuota:      10_000_000,
			ModelTier:         "DeepSeek-V3",
			Unit:              core.UnitTokens,
			Status:            scanner.StatusNew,
		},
		{
			ProviderID:        "antigravity",
			DisplayName:       "Antigravity Agent",
			SourceType:        "file (session telemetry)",
			KeySnippet:        "Local Session Active ✔",
			SuggestedCycleDay: 1,
			DefaultQuota:      10_000_000,
			ModelTier:         "Agent Pro Runtime",
			Unit:              core.UnitTokens,
			Status:            scanner.StatusConfigured,
		},
	}

	RenderScanAll(&buf, scanned, 1)
	out := buf.String()

	if !strings.Contains(out, "MACHINE AI ENVIRONMENT & CREDENTIALS SCANNER") {
		t.Errorf("expected scanner header in output")
	}
	if !strings.Contains(out, "DeepSeek AI") || !strings.Contains(out, "✨ NEW") {
		t.Errorf("expected DeepSeek AI with NEW status in output")
	}
	if !strings.Contains(out, "Antigravity Agent") || !strings.Contains(out, "✔ CONFIGURED") {
		t.Errorf("expected Antigravity Agent with CONFIGURED status in output")
	}
}

func TestRenderScanNewCards(t *testing.T) {
	var buf bytes.Buffer
	newCreds := []scanner.ScannedCredential{
		{
			ProviderID:        "openrouter",
			DisplayName:       "OpenRouter",
			SourcePath:        "$OPENROUTER_API_KEY",
			SourceType:        "env",
			KeySnippet:        "sk-or...5678",
			ModelTier:         "Unified Gateway",
			Endpoint:          "https://openrouter.ai/api/v1",
			SuggestedCycleDay: 1,
			DefaultQuota:      10_000_000,
			Unit:              core.UnitTokens,
			Status:            scanner.StatusNew,
		},
	}

	RenderScanNewCards(&buf, newCreds, 1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	out := buf.String()

	if !strings.Contains(out, "OpenRouter") {
		t.Errorf("expected OpenRouter in cards output")
	}
	if !strings.Contains(out, "Cycle Date") || !strings.Contains(out, "1st of each month") {
		t.Errorf("expected Cycle Date in cards output")
	}
	if !strings.Contains(out, "Rec. Quota") || !strings.Contains(out, "10.00M tokens") {
		t.Errorf("expected Rec. Quota in cards output")
	}

	// Empty list case
	var emptyBuf bytes.Buffer
	RenderScanNewCards(&emptyBuf, nil, 1, time.Now())
	if !strings.Contains(emptyBuf.String(), "All detected AI credentials on this machine are already configured") {
		t.Errorf("expected empty message for nil credentials")
	}
}
