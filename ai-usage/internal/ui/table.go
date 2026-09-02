package ui

import (
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"

	"ai-usage/internal/core"
)

// RenderProgressBar builds a clean Unicode / ASCII visual bar.
func RenderProgressBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(math.Round((percent / 100.0) * float64(width)))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return fmt.Sprintf("[%s%s]", strings.Repeat("■", filled), strings.Repeat("·", empty))
}

// RenderTable outputs a formatted summary table to the specified writer.
func RenderTable(w io.Writer, snapshot *core.UsageSnapshot) {
	boxWidth := 96
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(FormatBoxTop(boxWidth))

	if snapshot.IsLive {
		sb.WriteString(FormatBoxLine("🚀 AI MODEL USAGE MONITOR — 🟢 LIVE DATA TELEMETRY", boxWidth))
		sb.WriteString(FormatBoxLine("📡 Sourced in real-time from active session logs, local daemons & verified APIs", boxWidth))
	} else {
		sb.WriteString(FormatBoxLine("🧪 AI MODEL USAGE MONITOR — 🔬 EXAMPLE / SIMULATED BENCHMARK DATA", boxWidth))
		sb.WriteString(FormatBoxLine("💡 Showing representative multi-provider consumption figures for demonstration", boxWidth))
	}

	sb.WriteString(FormatBoxDivider(boxWidth))

	timeStr := fmt.Sprintf("📅 Current Time   : %s", snapshot.Timestamp.Local().Format("2006-01-02 15:04:05 MST"))
	cycleStr := fmt.Sprintf("💳 Default Window : %s ➔ %s", snapshot.BillingCycle.Start.Local().Format("2006-01-02"), snapshot.BillingCycle.End.Local().Format("2006-01-02"))
	provStr := fmt.Sprintf("📊 Total Providers: %d (Active: %d)", snapshot.TotalProviders, snapshot.ActiveOkCount)
	costStr := fmt.Sprintf("💰 Total Est Cost : $%.2f", snapshot.TotalCostUSD)

	sb.WriteString(FormatTwoColumnBoxLine(timeStr, 48, provStr, boxWidth))
	sb.WriteString(FormatTwoColumnBoxLine(cycleStr, 48, costStr, boxWidth))
	sb.WriteString(FormatBoxBottom(boxWidth))
	sb.WriteString("\n")

	_, _ = w.Write([]byte(sb.String()))

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "RANK\tPROVIDER\tMODEL / TIER\tCYCLE WINDOW\tCONSUMED\tQUOTA\tREMAINING\tUSAGE %\tPROGRESS BAR\tCOST (USD)\tSOURCE / STATUS")
	fmt.Fprintln(tw, "----\t--------\t------------\t------------\t--------\t-----\t---------\t-------\t------------\t----------\t---------------")

	staleCount := 0
	for i, p := range snapshot.Providers {
		rank := fmt.Sprintf("#%d", i+1)

		cycleWindowStr := "-"
		if !p.BillingStart.IsZero() && !p.BillingEnd.IsZero() {
			cycleWindowStr = fmt.Sprintf("%s➔%s", p.BillingStart.Local().Format("01/02"), p.BillingEnd.Local().Format("01/02"))
		}

		if p.IsStale {
			staleCount++
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t0 %s\t%s\t%s\t  0.0%%\t[············]\t$  0.00\t⚠️ STALE (%dd inactive)\n",
				rank, p.DisplayName, p.ModelOrTier, cycleWindowStr, p.Unit, p.FormatQuota(), p.FormatQuota(), p.DaysInactive)
			continue
		}

		if p.Status != "ok" {
			errText := p.ErrorMessage
			if len(errText) > 26 {
				errText = errText[:23] + "..."
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t-\t-\t-\t-\t-\t-\t❌ %s\n",
				rank, p.DisplayName, "-", cycleWindowStr, errText)
			continue
		}

		statusStr := "✔ OK"
		if p.DataSource != "" {
			if snapshot.IsLive {
				statusStr = fmt.Sprintf("🟢 %s", p.DataSource)
			} else {
				statusStr = fmt.Sprintf("🧪 %s", p.DataSource)
			}
		}

		bar := RenderProgressBar(p.PercentUsed, 12)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%5.1f%%\t%s\t$%6.2f\t%s\n",
			rank,
			p.DisplayName,
			p.ModelOrTier,
			cycleWindowStr,
			p.FormatConsumed(),
			p.FormatQuota(),
			p.FormatRemaining(),
			p.PercentUsed,
			bar,
			p.EstimatedCost,
			statusStr,
		)
	}

	tw.Flush()
	fmt.Fprintln(w)

	if staleCount > 0 {
		fmt.Fprintf(w, "⚠️  NOTICE: %d provider(s) have had no usage for 30+ days and are marked as STALE.\n", staleCount)
		fmt.Fprintf(w, "   • To update or re-verify a key : usage update-key <id> <new_key>\n")
		fmt.Fprintf(w, "   • To clean up inactive records : usage prune-stale\n\n")
	}
}
