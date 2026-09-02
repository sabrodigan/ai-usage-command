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
			{
				ProviderID:   "copilot",
				DisplayName:  "GitHub Copilot",
				Status:       "error",
				ErrorMessage: "no GitHub token configured",
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
	if !strings.Contains(tableStr, "🔴 no GitHub token configured") {
		t.Errorf("table output missing red ball for error status:\n%s", tableStr)
	}
	if strings.Contains(tableStr, "❌") {
		t.Errorf("table output should not contain red X:\n%s", tableStr)
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

func TestVisualWidth_SymbolsAndArrows(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"➔", 1},
		{"✔", 1},
		{"✓", 1},
		{"—", 1},
		{"•", 1},
		{"·", 1},
		{"↑", 1},
		{"↓", 1},
		{"│", 1},
		{"─", 1},
		{"💳", 2},
		{"📊", 2},
		{"⏳", 2},
		{"📅", 2},
		{"💰", 2},
		{"📡", 2},
		{"🔀", 2},
		{"🔖", 2},
		{"🟢", 2},
		{"⚡", 2},
		{"\033[H", 0},
		{"\033[0m", 0},
		{"\033[32mOK\033[0m", 2},
		{"💳 Billing Window : 2026-09-01 ➔ 2026-09-30", 43},
		{"📊 Providers    : 3 Active / 7 Total", 36},
		{"🔖 Filter Mode    : None", 24},
		{"🔀 Sort Mode    : Name (A-Z)", 28},
	}

	for _, c := range cases {
		got := VisualWidth(c.input)
		if got != c.want {
			t.Errorf("VisualWidth(%q) = %d; want %d", c.input, got, c.want)
		}
	}
}

func TestHeaderBox_BillingWindowProvidersAlignment(t *testing.T) {
	now := time.Date(2026, 9, 2, 18, 15, 0, 0, time.Local)
	snap := &core.UsageSnapshot{
		Timestamp: now,
		BillingCycle: core.BillingWindow{
			Start: time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local),
			End:   time.Date(2026, 9, 30, 23, 59, 59, 0, time.Local),
		},
		TotalProviders: 7,
		ActiveOkCount:  3,
		TotalCostUSD:   64.02,
		IsLive:         true,
	}

	state := &DashboardState{
		RefreshInterval: 3 * time.Second,
		LastSnapshot:    snap,
		LastFetched:     now,
		Sort:            SortByUsagePercent,
	}

	var buf bytes.Buffer
	renderDashboardView(&buf, state)
	content := buf.String()

	// Strip the leading \033[H
	if strings.HasPrefix(content, "\033[H") {
		content = content[3:]
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 8 {
		t.Fatalf("expected at least 8 lines in header box, got %d", len(lines))
	}

	// Line 0: Top border
	// Line 1: Title
	// Line 2: Divider
	// Line 3: Local Time / Next Refresh
	// Line 4: Billing Window / Providers
	// Line 5: Total Est Cost / Status
	// Line 6: Filter Mode / Sort Mode
	// Line 7: Bottom border
	boxWidth := 96
	for i := 0; i <= 7; i++ {
		l := lines[i]
		vw := VisualWidth(l)
		if vw != boxWidth {
			t.Errorf("line %d visual width = %d; want %d: %q", i, vw, boxWidth, l)
		}
		if !strings.HasPrefix(l, "┌") && !strings.HasPrefix(l, "├") && !strings.HasPrefix(l, "└") {
			if !strings.HasSuffix(l, "  │") {
				t.Errorf("line %d does not end with closing border '  │': %q", i, l)
			}
		}
	}

	// Verify the Billing Window and Providers line
	bwLine := lines[4]
	if !strings.Contains(bwLine, "💳 Billing Window :") {
		t.Errorf("expected line 4 to contain '💳 Billing Window :', got %q", bwLine)
	}
	if !strings.Contains(bwLine, "📊 Providers    :") {
		t.Errorf("expected line 4 to contain '📊 Providers    :', got %q", bwLine)
	}

	filterLine := lines[6]
	if !strings.Contains(filterLine, "🔖 Filter Mode    :") {
		t.Errorf("expected line 6 to contain '🔖 Filter Mode    :', got %q", filterLine)
	}
	if strings.Contains(filterLine, "🏷️") {
		t.Errorf("expected no narrow label emoji on filter line: %q", filterLine)
	}

	// Verify provider column visual alignment matches refresh, status, and sort columns
	refreshCol := VisualWidth(lines[3][:strings.Index(lines[3], "⏳ Next Refresh")])
	providersCol := VisualWidth(lines[4][:strings.Index(lines[4], "📊 Providers")])
	statusCol := VisualWidth(lines[5][:strings.Index(lines[5], "📡 Status")])
	sortCol := VisualWidth(lines[6][:strings.Index(lines[6], "🔀 Sort Mode")])

	if providersCol != refreshCol || providersCol != statusCol || providersCol != sortCol {
		t.Errorf("misaligned visual columns: refresh at %d, providers at %d, status at %d, sort at %d",
			refreshCol, providersCol, statusCol, sortCol)
	}

	// Every sort mode must keep the filter/sort row exactly box-width (no overflow at bottom-right).
	for sortMode := SortByUsagePercent; sortMode <= SortByName; sortMode++ {
		state.Sort = sortMode
		var b bytes.Buffer
		renderDashboardView(&b, state)
		out := b.String()
		if strings.HasPrefix(out, "\033[H") {
			out = out[3:]
		}
		row := strings.Split(out, "\n")[6]
		if vw := VisualWidth(row); vw != boxWidth {
			t.Errorf("sort %s filter/sort line visual width = %d; want %d: %q", sortMode, vw, boxWidth, row)
		}
	}
}

