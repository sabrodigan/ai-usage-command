package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"ai-usage/internal/core"
	"ai-usage/internal/scanner"
)

func ordinal(n int) string {
	switch n {
	case 1, 21, 31:
		return fmt.Sprintf("%dst", n)
	case 2, 22:
		return fmt.Sprintf("%dnd", n)
	case 3, 23:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// RenderScanAll outputs a complete breakdown of all detected credentials and tools on the machine.
func RenderScanAll(w io.Writer, scanned []scanner.ScannedCredential, globalCycleDay int) {
	fmt.Fprintf(w, "\n┌────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│  🔍 MACHINE AI ENVIRONMENT & CREDENTIALS SCANNER                       │\n")
	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────┘\n\n")

	if len(scanned) == 0 {
		fmt.Fprintln(w, "No AI provider credentials or local session configs detected on this machine.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tPROVIDER\tSOURCE\tKEY PREVIEW\tCYCLE RESET\tDEFAULT QUOTA\tTIER")
	fmt.Fprintln(tw, "------\t--------\t------\t-----------\t-----------\t-------------\t----")

	for _, s := range scanned {
		statusStr := "✔ CONFIGURED"
		switch s.Status {
		case scanner.StatusNew:
			statusStr = "✨ NEW"
		case scanner.StatusUpdatedKey:
			statusStr = "🔄 NEW KEY"
		}

		cycleDay := s.SuggestedCycleDay
		if s.ExistingCycleDay > 0 {
			cycleDay = s.ExistingCycleDay
		}
		if cycleDay <= 0 {
			cycleDay = globalCycleDay
		}

		quota := s.DefaultQuota
		if s.ExistingQuota > 0 {
			quota = s.ExistingQuota
		}
		quotaStr := core.FormatNumber(quota)
		if s.Unit == core.UnitRequests {
			quotaStr = fmt.Sprintf("%.0f reqs", quota)
		} else if s.Unit == core.UnitTokens {
			quotaStr = quotaStr + " tok"
		}

		sourceStr := s.SourceType
		if len(sourceStr) > 26 {
			sourceStr = sourceStr[:23] + "..."
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			statusStr,
			s.DisplayName,
			sourceStr,
			s.KeySnippet,
			ordinal(cycleDay),
			quotaStr,
			s.ModelTier,
		)
	}
	tw.Flush()
	fmt.Fprintln(w)
}

// RenderScanNewCards renders distinct, prominent cards for newly discovered or updated credentials.
func RenderScanNewCards(w io.Writer, newCreds []scanner.ScannedCredential, globalCycleDay int, now time.Time) {
	if len(newCreds) == 0 {
		fmt.Fprintln(w, "\n✔ All detected AI credentials on this machine are already configured in your usage tracker.")
		fmt.Fprintln(w, "  Run 'usage providers' to inspect active providers, or 'usage scan' to see full machine inventory.")
		fmt.Fprintln(w)
		return
	}

	fmt.Fprintf(w, "\n🔍 Scanned machine against your known providers: Found %d new or updated credential(s):\n\n", len(newCreds))

	for i, c := range newCreds {
		tag := "✨ NEW PROVIDER"
		if c.Status == scanner.StatusUpdatedKey {
			tag = "🔄 UPDATED KEY DETECTED"
		}

		cycleDay := c.SuggestedCycleDay
		if cycleDay <= 0 {
			cycleDay = globalCycleDay
		}
		if cycleDay <= 0 {
			cycleDay = 1
		}

		win := core.CalculateBillingCycle(now, cycleDay)
		winStr := fmt.Sprintf("%s ➔ %s", win.Start.Local().Format("01/02"), win.End.Local().Format("01/02"))

		quotaStr := core.FormatNumber(c.DefaultQuota)
		if c.Unit == core.UnitRequests {
			quotaStr = fmt.Sprintf("%.0f requests", c.DefaultQuota)
		} else if c.Unit == core.UnitTokens {
			quotaStr = quotaStr + " tokens"
		}

		fmt.Fprintf(w, "┌─── [%d/%d] %s: %s (%s) %s\n",
			i+1,
			len(newCreds),
			tag,
			c.DisplayName,
			c.ProviderID,
			strings.Repeat("─", max(2, 60-len(c.DisplayName)-len(c.ProviderID)-len(tag))),
		)
		fmt.Fprintf(w, "│  📍 Source       : %s (%s)\n", c.SourcePath, c.SourceType)
		fmt.Fprintf(w, "│  🔑 Key Preview  : %s\n", c.KeySnippet)
		fmt.Fprintf(w, "│  🤖 Model / Tier : %s\n", c.ModelTier)
		fmt.Fprintf(w, "│  💳 Cycle Date   : %s of each month (Current Window: %s)\n", ordinal(cycleDay), winStr)
		fmt.Fprintf(w, "│  📊 Rec. Quota   : %s\n", quotaStr)
		if c.Endpoint != "" {
			fmt.Fprintf(w, "│  🌐 API Endpoint : %s\n", c.Endpoint)
		}
		fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────┘\n")
	}
	fmt.Fprintln(w)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
