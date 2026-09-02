package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ai-usage/internal/core"
)

func TestFilterAndSortProviders(t *testing.T) {
	providers := []core.ProviderUsage{
		{
			ProviderID:    "copilot",
			DisplayName:   "GitHub Copilot",
			ModelOrTier:   "Individual Plan",
			PercentUsed:   41.3,
			Consumed:      1240,
			Quota:         3000,
			Remaining:     1760,
			EstimatedCost: 10.0,
			Status:        "ok",
		},
		{
			ProviderID:    "cursor",
			DisplayName:   "Cursor IDE",
			ModelOrTier:   "Cursor Pro",
			PercentUsed:   27.6,
			Consumed:      138,
			Quota:         500,
			Remaining:     362,
			EstimatedCost: 20.0,
			Status:        "ok",
		},
		{
			ProviderID:    "claude",
			DisplayName:   "Anthropic Claude",
			ModelOrTier:   "Claude 3.5 Sonnet",
			PercentUsed:   85.0,
			Consumed:      4250000,
			Quota:         5000000,
			Remaining:     750000,
			EstimatedCost: 63.75,
			Status:        "ok",
		},
	}

	// 1. Sort by Usage %
	sortedUsage := filterAndSortProviders(providers, "", SortByUsagePercent)
	if sortedUsage[0].ProviderID != "claude" || sortedUsage[1].ProviderID != "copilot" || sortedUsage[2].ProviderID != "cursor" {
		t.Errorf("expected claude, copilot, cursor when sorted by usage, got %v, %v, %v",
			sortedUsage[0].ProviderID, sortedUsage[1].ProviderID, sortedUsage[2].ProviderID)
	}

	// 2. Sort by Cost
	sortedCost := filterAndSortProviders(providers, "", SortByCost)
	if sortedCost[0].ProviderID != "claude" || sortedCost[1].ProviderID != "cursor" || sortedCost[2].ProviderID != "copilot" {
		t.Errorf("expected claude, cursor, copilot when sorted by cost, got %v, %v, %v",
			sortedCost[0].ProviderID, sortedCost[1].ProviderID, sortedCost[2].ProviderID)
	}

	// 3. Sort by Name
	sortedName := filterAndSortProviders(providers, "", SortByName)
	if sortedName[0].DisplayName != "Anthropic Claude" || sortedName[1].DisplayName != "Cursor IDE" || sortedName[2].DisplayName != "GitHub Copilot" {
		t.Errorf("expected alphabetical order, got %v, %v, %v",
			sortedName[0].DisplayName, sortedName[1].DisplayName, sortedName[2].DisplayName)
	}

	// 4. Filtering
	filtered := filterAndSortProviders(providers, "cursor", SortByUsagePercent)
	if len(filtered) != 1 || filtered[0].ProviderID != "cursor" {
		t.Errorf("expected 1 match for filter 'cursor', got %d", len(filtered))
	}
}

func TestRenderDashboardView(t *testing.T) {
	snap := &core.UsageSnapshot{
		Timestamp:      time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		TotalProviders: 2,
		ActiveOkCount:  2,
		TotalCostUSD:   30.0,
		IsLive:         true,
		Providers: []core.ProviderUsage{
			{
				ProviderID:    "gemini",
				DisplayName:   "Google Gemini",
				ModelOrTier:   "Gemini 1.5 Pro",
				Unit:          core.UnitTokens,
				Consumed:      500000,
				Quota:         4000000,
				Remaining:     3500000,
				PercentUsed:   12.5,
				EstimatedCost: 1.75,
				Status:        "ok",
			},
		},
	}

	state := &DashboardState{
		SelectedIdx:     0,
		Sort:            SortByUsagePercent,
		RefreshInterval: 3 * time.Second,
		LastSnapshot:    snap,
		LastFetched:     time.Now(),
		ShowHelp:        true,
		ExpandedDetails: true,
	}

	var buf bytes.Buffer
	renderDashboardView(&buf, state)
	out := buf.String()

	if !strings.Contains(out, "LIVE DATA DASHBOARD") {
		t.Errorf("expected live dashboard header in output")
	}
	if !strings.Contains(out, "KEYBOARD SHORTCUTS & HELP") {
		t.Errorf("expected help shortcuts overlay in output")
	}
	if !strings.Contains(out, "SELECTED PROVIDER INSPECTION (EXPANDED DETAILS)") {
		t.Errorf("expected expanded inspector in output")
	}
	if !strings.Contains(out, "Google Gemini") {
		t.Errorf("expected Google Gemini in output")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Errorf("unexpected truncate result")
	}
	if truncate("very long string here", 8) != "very ..." {
		t.Errorf("unexpected truncate result: %q", truncate("very long string here", 8))
	}
}
