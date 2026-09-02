package ui

import (
	"fmt"
	"io"
	"text/tabwriter"

	"ai-usage/internal/analytics"
	"ai-usage/internal/storage"
)

// RenderRecommendations displays intelligence insights and cost suggestions.
func RenderRecommendations(w io.Writer, recs []analytics.Recommendation) {
	fmt.Fprintf(w, "\n┌────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│  🧠 AI USAGE INTELLIGENCE & RECOMMENDATIONS                            │\n")
	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────┘\n\n")

	if len(recs) == 0 {
		fmt.Fprintln(w, "✔ All AI consumption patterns are balanced and within projected limits.")
		return
	}

	var totalSavings float64 = 0
	for i, r := range recs {
		icon := "💡"
		switch r.Urgency {
		case "HIGH":
			icon = "🚨"
		case "MEDIUM":
			icon = "⚡"
		case "TIP":
			icon = "✨"
		}

		fmt.Fprintf(w, "%s [%s] %s\n", icon, r.Category, r.Title)
		fmt.Fprintf(w, "   %s\n", r.Message)
		if r.PotentialSavingsUSD > 0 {
			fmt.Fprintf(w, "   💵 Potential Monthly Savings: ~$%.2f\n", r.PotentialSavingsUSD)
			totalSavings += r.PotentialSavingsUSD
		}
		if i < len(recs)-1 {
			fmt.Fprintln(w)
		}
	}

	if totalSavings > 0 {
		fmt.Fprintf(w, "\n────────────────────────────────────────────────────────────────────────\n")
		fmt.Fprintf(w, "💰 Estimated Total Potential Monthly Savings: ~$%.2f\n", totalSavings)
		fmt.Fprintf(w, "────────────────────────────────────────────────────────────────────────\n\n")
	} else {
		fmt.Fprintln(w)
	}
}

// RenderDailyTrends prints daily usage time-series records from the local SQLite DB.
func RenderDailyTrends(w io.Writer, summaries []storage.ProviderDailySummary) {
	fmt.Fprintf(w, "\n📊 Historical Daily AI Consumption (Local SQLite Database):\n")
	fmt.Fprintf(w, "────────────────────────────────────────────────────────────────────────\n")

	if len(summaries) == 0 {
		fmt.Fprintln(w, "No historical records recorded yet. Run 'usage' to record snapshots.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "DATE\tPROVIDER\tPEAK CONSUMED\tEST. COST\tUSAGE %")
	fmt.Fprintln(tw, "----\t--------\t-------------\t---------\t-------")

	for _, s := range summaries {
		fmt.Fprintf(tw, "%s\t%s\t%.0f\t$%.2f\t%5.1f%%\n",
			s.Date, s.DisplayName, s.Consumed, s.CostUSD, s.PercentUsed)
	}
	tw.Flush()
	fmt.Fprintln(w)
}
