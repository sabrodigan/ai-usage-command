package analytics

import (
	"fmt"
	"math"
	"time"

	"ai-usage/internal/core"
)

type Recommendation struct {
	ProviderID          string  `json:"provider_id"`
	Category            string  `json:"category"` // "Quota Risk", "Cost Optimization", "Workload Routing", "Subscription"
	Title               string  `json:"title"`
	Message             string  `json:"message"`
	PotentialSavingsUSD float64 `json:"potential_savings_usd,omitempty"`
	Urgency             string  `json:"urgency"` // "HIGH", "MEDIUM", "LOW", "TIP"
}

// GenerateInsights analyzes current snapshot and generates actionable AI recommendations.
func GenerateInsights(snapshot *core.UsageSnapshot) []Recommendation {
	var recs []Recommendation
	now := time.Now().UTC()

	for _, p := range snapshot.Providers {
		if p.Status != "ok" {
			continue
		}

		// 1. Quota Run-Rate Projection
		if !p.BillingStart.IsZero() && !p.BillingEnd.IsZero() && p.Quota > 0 {
			cycleDurationDays := p.BillingEnd.Sub(p.BillingStart).Hours() / 24
			daysElapsed := now.Sub(p.BillingStart).Hours() / 24
			if daysElapsed < 0.5 {
				daysElapsed = 0.5
			}

			dailyBurn := p.Consumed / daysElapsed
			projectedTotal := dailyBurn * cycleDurationDays

			if projectedTotal > p.Quota && p.PercentUsed > 20 {
				daysRemaining := (p.Quota - p.Consumed) / dailyBurn
				if daysRemaining < 0 {
					daysRemaining = 0
				}

				recs = append(recs, Recommendation{
					ProviderID: p.ProviderID,
					Category:   "Quota Risk",
					Title:      fmt.Sprintf("%s Quota Run-Rate Alert", p.DisplayName),
					Message: fmt.Sprintf(
						"At your current burn rate (%.0f %s/day), you are projected to reach %.0f %s (%.0f%% of quota) and exhaust remaining allowance in ~%.1f days (Cycle resets on %s).",
						dailyBurn, p.Unit, projectedTotal, p.Unit, (projectedTotal/p.Quota)*100, daysRemaining, p.BillingEnd.Local().Format("Jan 02"),
					),
					Urgency: "HIGH",
				})
			}
		}

		// 2. Cost Optimization Recommendations
		if p.Unit == core.UnitTokens && p.Consumed > 500_000 && p.EstimatedCost > 10.0 {
			potentialSavings := p.EstimatedCost * 0.70
			recs = append(recs, Recommendation{
				ProviderID: p.ProviderID,
				Category:   "Cost Optimization",
				Title:      fmt.Sprintf("Cost Optimization for %s", p.DisplayName),
				Message: fmt.Sprintf(
					"You have consumed %s on %s ($%.2f est. cost). Routing automated summaries and background workflows to fast tier models (e.g. Gemini 1.5 Flash or Claude 3.5 Haiku) can reduce cost by up to 70%%.",
					p.FormatConsumed(), p.ModelOrTier, p.EstimatedCost,
				),
				PotentialSavingsUSD: math.Round(potentialSavings*100) / 100,
				Urgency:             "MEDIUM",
			})
		}

		// 3. Subscription Underutilization
		if p.Quota > 0 && p.EstimatedCost >= 10.0 && p.PercentUsed < 5.0 && !p.BillingStart.IsZero() {
			daysElapsed := now.Sub(p.BillingStart).Hours() / 24
			if daysElapsed >= 20 {
				recs = append(recs, Recommendation{
					ProviderID: p.ProviderID,
					Category:   "Subscription",
					Title:      fmt.Sprintf("%s Subscription Underutilized", p.DisplayName),
					Message: fmt.Sprintf(
						"You are paying $%.2f/mo for %s, but have only used %.1f%% of quota with less than %d days left in your cycle.",
						p.EstimatedCost, p.DisplayName, p.PercentUsed, int(p.BillingEnd.Sub(now).Hours()/24),
					),
					Urgency: "LOW",
				})
			}
		}
	}

	// 4. Multi-Model Balancing Tip
	hasAntigravity := false
	for _, p := range snapshot.Providers {
		if p.ProviderID == "antigravity" && p.Status == "ok" && p.PercentUsed < 30 {
			hasAntigravity = true
			break
		}
	}
	if hasAntigravity {
		recs = append(recs, Recommendation{
			ProviderID: "antigravity",
			Category:   "Workload Routing",
			Title:      "Leverage Included Antigravity Agent Quota",
			Message:    "Your Antigravity agent interactions are bundled in your Google subscription ($0.00 extra cost). Offloading large refactors and codebase searches here preserves paid tokens on other APIs.",
			Urgency:    "TIP",
		})
	}

	return recs
}
