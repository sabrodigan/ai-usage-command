package analytics

import (
	"testing"
	"time"

	"ai-usage/internal/core"
)

func TestAnalytics_GenerateInsights(t *testing.T) {
	now := time.Now().UTC()
	snapshot := &core.UsageSnapshot{
		Timestamp: now,
		BillingCycle: core.BillingWindow{
			Start: now.AddDate(0, 0, -10),
			End:   now.AddDate(0, 0, 20),
		},
		TotalProviders: 2,
		ActiveOkCount:  2,
		Providers: []core.ProviderUsage{
			{
				ProviderID:    "claude",
				DisplayName:   "Anthropic Claude",
				ModelOrTier:   "Claude 3.5 Sonnet",
				Unit:          core.UnitTokens,
				Consumed:      1_800_000,
				Quota:         2_000_000,
				Remaining:     200_000,
				PercentUsed:   90.0,
				EstimatedCost: 27.00,
				BillingStart:  now.AddDate(0, 0, -10),
				BillingEnd:    now.AddDate(0, 0, 20),
				Status:        "ok",
			},
			{
				ProviderID:    "antigravity",
				DisplayName:   "Antigravity Agent",
				ModelOrTier:   "Agent Pro Runtime",
				Unit:          core.UnitTokens,
				Consumed:      200_000,
				Quota:         10_000_000,
				Remaining:     9_800_000,
				PercentUsed:   2.0,
				EstimatedCost: 0.00,
				BillingStart:  now.AddDate(0, 0, -10),
				BillingEnd:    now.AddDate(0, 0, 20),
				Status:        "ok",
			},
		},
	}

	insights := GenerateInsights(snapshot)
	if len(insights) < 2 {
		t.Fatalf("expected at least 2 insights, got %d", len(insights))
	}

	hasQuotaAlert := false
	hasCostAlert := false
	for _, in := range insights {
		if in.Category == "Quota Risk" {
			hasQuotaAlert = true
		}
		if in.Category == "Cost Optimization" {
			hasCostAlert = true
		}
	}

	if !hasQuotaAlert {
		t.Errorf("expected Quota Risk insight for high-burn provider")
	}
	if !hasCostAlert {
		t.Errorf("expected Cost Optimization insight for expensive token usage")
	}
}
