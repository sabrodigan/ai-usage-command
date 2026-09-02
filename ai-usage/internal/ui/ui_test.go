package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ai-usage/internal/core"
)

func TestUI_RenderProgressBar(t *testing.T) {
	bar0 := RenderProgressBar(0, 10)
	if bar0 != "[··········]" {
		t.Errorf("expected 0%% bar, got %s", bar0)
	}

	bar50 := RenderProgressBar(50, 10)
	if bar50 != "[■■■■■·····]" {
		t.Errorf("expected 50%% bar, got %s", bar50)
	}

	bar100 := RenderProgressBar(100, 10)
	if bar100 != "[■■■■■■■■■■]" {
		t.Errorf("expected 100%% bar, got %s", bar100)
	}
}

func TestUI_RenderTableAndJSONAndCSV(t *testing.T) {
	snapshot := &core.UsageSnapshot{
		Timestamp: time.Now().UTC(),
		BillingCycle: core.BillingWindow{
			Start: time.Now().UTC().AddDate(0, 0, -15),
			End:   time.Now().UTC().AddDate(0, 0, 15),
		},
		TotalProviders: 1,
		ActiveOkCount:  1,
		TotalTokens:    1_000_000,
		TotalCostUSD:   15.00,
		Providers: []core.ProviderUsage{
			{
				ProviderID:    "claude",
				DisplayName:   "Anthropic Claude",
				ModelOrTier:   "Claude 3.5 Sonnet",
				Unit:          core.UnitTokens,
				Consumed:      1_000_000,
				Quota:         5_000_000,
				Remaining:     4_000_000,
				PercentUsed:   20.0,
				EstimatedCost: 15.00,
				Status:        "ok",
			},
		},
	}

	// Table render
	var tableBuf bytes.Buffer
	RenderTable(&tableBuf, snapshot)
	tableStr := tableBuf.String()
	if !strings.Contains(tableStr, "Anthropic Claude") || !strings.Contains(tableStr, "20.0%") {
		t.Errorf("table output missing expected content:\n%s", tableStr)
	}

	// JSON render
	var jsonBuf bytes.Buffer
	if err := RenderJSON(&jsonBuf, snapshot); err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"percent_used": 20`) {
		t.Errorf("json output missing percent_used: %s", jsonBuf.String())
	}

	// CSV render
	var csvBuf bytes.Buffer
	if err := RenderCSV(&csvBuf, snapshot); err != nil {
		t.Fatalf("csv render failed: %v", err)
	}
	if !strings.Contains(csvBuf.String(), "Anthropic Claude") {
		t.Errorf("csv output missing provider: %s", csvBuf.String())
	}
}
