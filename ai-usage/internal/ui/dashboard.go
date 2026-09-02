package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"ai-usage/internal/core"
	"golang.org/x/sys/unix"
)

// SortMode represents how providers are ordered in the dashboard.
type SortMode int

const (
	SortByUsagePercent SortMode = iota
	SortByConsumed
	SortByQuota
	SortByRemaining
	SortByCost
	SortByName
)

func (s SortMode) String() string {
	switch s {
	case SortByUsagePercent:
		return "Usage % (Highest First)"
	case SortByConsumed:
		return "Consumed (Highest First)"
	case SortByQuota:
		return "Quota (Largest First)"
	case SortByRemaining:
		return "Remaining Quota (Lowest First)"
	case SortByCost:
		return "Estimated Cost (Highest First)"
	case SortByName:
		return "Provider Name (A-Z)"
	default:
		return "Default"
	}
}

// DashboardState manages the live TUI state.
type DashboardState struct {
	SelectedIdx     int
	Sort            SortMode
	FilterQuery     string
	RefreshInterval time.Duration
	ShowHelp        bool
	ExpandedDetails bool
	LastSnapshot    *core.UsageSnapshot
	LastFetched     time.Time
	IsFetching      bool
	NextFetchTime   time.Time

	mu sync.Mutex
}

// EnableRawMode switches the terminal to non-canonical / raw mode and returns a restore function.
func EnableRawMode() (func(), error) {
	fd := int(os.Stdin.Fd())
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return func() {}, err
	}

	oldState := *termios
	termios.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return func() {}, err
	}

	restore := func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, &oldState)
	}
	return restore, nil
}

// RunDashboard runs the interactive live dashboard loop.
func RunDashboard(
	fetchFunc func(ctx context.Context) *core.UsageSnapshot,
	initialInterval time.Duration,
	initialFilter string,
) error {
	if initialInterval <= 0 {
		initialInterval = 3 * time.Second
	}

	state := &DashboardState{
		SelectedIdx:     0,
		Sort:            SortByUsagePercent,
		FilterQuery:     initialFilter,
		RefreshInterval: initialInterval,
		ShowHelp:        false,
	}

	// Switch to alternate screen and hide cursor
	os.Stdout.WriteString("\033[?1049h\033[?25l")
	defer func() {
		os.Stdout.WriteString("\033[?25h\033[?1049l")
	}()

	restoreTerm, _ := EnableRawMode()
	defer restoreTerm()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigChan
		cancel()
	}()

	refreshChan := make(chan struct{}, 1)

	// Fetch worker
	triggerFetch := func() {
		state.mu.Lock()
		if state.IsFetching {
			state.mu.Unlock()
			return
		}
		state.IsFetching = true
		state.mu.Unlock()

		go func() {
			fetchCtx, fetchCancel := context.WithTimeout(ctx, 8*time.Second)
			defer fetchCancel()

			snap := fetchFunc(fetchCtx)

			state.mu.Lock()
			state.LastSnapshot = snap
			state.LastFetched = time.Now()
			state.IsFetching = false
			state.NextFetchTime = time.Now().Add(state.RefreshInterval)
			state.mu.Unlock()

			select {
			case refreshChan <- struct{}{}:
			default:
			}
		}()
	}

	// Initial fetch
	triggerFetch()

	// Keyboard input worker
	keyChan := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 16)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case keyChan <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	// UI render ticker (10 Hz for smooth clock and countdown)
	uiTicker := time.NewTicker(200 * time.Millisecond)
	defer uiTicker.Stop()

	// Auto-fetch ticker
	fetchTicker := time.NewTicker(1 * time.Second)
	defer fetchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-uiTicker.C:
			renderDashboardView(os.Stdout, state)

		case <-fetchTicker.C:
			state.mu.Lock()
			interval := state.RefreshInterval
			lastFetch := state.LastFetched
			isFetching := state.IsFetching
			state.mu.Unlock()

			if !isFetching && time.Since(lastFetch) >= interval {
				triggerFetch()
			}

		case <-refreshChan:
			renderDashboardView(os.Stdout, state)

		case keys := <-keyChan:
			if len(keys) == 0 {
				continue
			}

			// Handle Escape sequences (Arrow keys)
			if len(keys) >= 3 && keys[0] == 27 && keys[1] == '[' {
				switch keys[2] {
				case 'A': // Up Arrow
					moveSelection(state, -1)
				case 'B': // Down Arrow
					moveSelection(state, 1)
				}
				renderDashboardView(os.Stdout, state)
				continue
			}

			switch keys[0] {
			case 'q', 'Q', 3: // 'q' or Ctrl+C
				return nil
			case 27: // Esc
				state.mu.Lock()
				if state.ShowHelp {
					state.ShowHelp = false
				} else if state.FilterQuery != "" {
					state.FilterQuery = ""
				}
				state.mu.Unlock()
			case 'r', 'R': // Force refresh
				triggerFetch()
			case 'j', 'J': // Next row
				moveSelection(state, 1)
			case 'k', 'K': // Prev row
				moveSelection(state, -1)
			case 's', 'S': // Cycle sort mode
				state.mu.Lock()
				state.Sort = (state.Sort + 1) % 6
				state.mu.Unlock()
			case 'h', 'H', '?': // Toggle help
				state.mu.Lock()
				state.ShowHelp = !state.ShowHelp
				state.mu.Unlock()
			case ' ', 13, 10: // Space or Enter (toggle details)
				state.mu.Lock()
				state.ExpandedDetails = !state.ExpandedDetails
				state.mu.Unlock()
			case '+', '=', ']': // Increase interval
				state.mu.Lock()
				if state.RefreshInterval < 60*time.Second {
					state.RefreshInterval += 1 * time.Second
				}
				state.mu.Unlock()
			case '-', '[': // Decrease interval
				state.mu.Lock()
				if state.RefreshInterval > 1*time.Second {
					state.RefreshInterval -= 1 * time.Second
				}
				state.mu.Unlock()
			case '/', 'f', 'F': // Filter prompt
				restoreTerm()
				promptForFilter(state)
				restoreTerm, _ = EnableRawMode()
			}

			renderDashboardView(os.Stdout, state)
		}
	}
}

func moveSelection(state *DashboardState, delta int) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.LastSnapshot == nil || len(state.LastSnapshot.Providers) == 0 {
		return
	}

	filtered := filterAndSortProviders(state.LastSnapshot.Providers, state.FilterQuery, state.Sort)
	count := len(filtered)
	if count == 0 {
		state.SelectedIdx = 0
		return
	}

	newIdx := state.SelectedIdx + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= count {
		newIdx = count - 1
	}
	state.SelectedIdx = newIdx
}

func promptForFilter(state *DashboardState) {
	// Show cursor temporarily for typing
	os.Stdout.WriteString("\033[?25h")
	fmt.Print("\n\033[K🔍 Enter provider filter (or press Enter to clear): ")

	var line string
	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			break
		}
		ch := buf[0]
		if ch == '\n' || ch == '\r' {
			break
		}
		if ch == 127 || ch == 8 { // Backspace
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Print("\b \b")
			}
			continue
		}
		if ch >= 32 && ch <= 126 {
			line += string(ch)
			fmt.Print(string(ch))
		}
	}

	state.mu.Lock()
	state.FilterQuery = strings.TrimSpace(line)
	state.SelectedIdx = 0
	state.mu.Unlock()

	os.Stdout.WriteString("\033[?25l")
}

func filterAndSortProviders(providers []core.ProviderUsage, query string, sortMode SortMode) []core.ProviderUsage {
	var filtered []core.ProviderUsage
	q := strings.ToLower(strings.TrimSpace(query))

	for _, p := range providers {
		if q != "" {
			match := strings.Contains(strings.ToLower(p.DisplayName), q) ||
				strings.Contains(strings.ToLower(p.ProviderID), q) ||
				strings.Contains(strings.ToLower(p.ModelOrTier), q)
			if !match {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		pi := filtered[i]
		pj := filtered[j]

		// Error items generally placed at bottom unless sorting by name
		if sortMode != SortByName {
			if (pi.Status == "ok") != (pj.Status == "ok") {
				return pi.Status == "ok"
			}
		}

		switch sortMode {
		case SortByUsagePercent:
			return pi.PercentUsed > pj.PercentUsed
		case SortByConsumed:
			return pi.Consumed > pj.Consumed
		case SortByQuota:
			return pi.Quota > pj.Quota
		case SortByRemaining:
			return pi.Remaining < pj.Remaining
		case SortByCost:
			return pi.EstimatedCost > pj.EstimatedCost
		case SortByName:
			return pi.DisplayName < pj.DisplayName
		default:
			return pi.PercentUsed > pj.PercentUsed
		}
	})

	return filtered
}

func renderDashboardView(w io.Writer, state *DashboardState) {
	state.mu.Lock()
	snap := state.LastSnapshot
	selectedIdx := state.SelectedIdx
	sortMode := state.Sort
	filterQ := state.FilterQuery
	interval := state.RefreshInterval
	lastFetched := state.LastFetched
	isFetching := state.IsFetching
	showHelp := state.ShowHelp
	expanded := state.ExpandedDetails
	state.mu.Unlock()

	var sb strings.Builder
	// Move cursor to top left
	sb.WriteString("\033[H")

	now := time.Now()
	secsUntilRefresh := int(interval.Seconds() - time.Since(lastFetched).Seconds())
	if secsUntilRefresh < 0 {
		secsUntilRefresh = 0
	}

	statusIndicator := "🟢 IDLE"
	if isFetching {
		statusIndicator = "⚡ REFRESHING..."
	}

	boxWidth := 96

	// 1. Header Box
	sb.WriteString(FormatBoxTop(boxWidth))
	if snap != nil && snap.IsLive {
		sb.WriteString(FormatBoxLine("🚀 AI MODEL USAGE MONITOR — 🟢 LIVE DATA DASHBOARD", boxWidth))
	} else {
		sb.WriteString(FormatBoxLine("🧪 AI MODEL USAGE MONITOR — 🔬 EXAMPLE / BENCHMARK SIMULATION", boxWidth))
	}
	sb.WriteString(FormatBoxDivider(boxWidth))

	timeStr := fmt.Sprintf("📅 Local Time     : %s", now.Format("2006-01-02 15:04:05 MST"))
	refreshStr := fmt.Sprintf("⏳ Next Refresh : %ds (Interval: %ds)", secsUntilRefresh, int(interval.Seconds()))
	sb.WriteString(FormatTwoColumnBoxLine(timeStr, 48, refreshStr, boxWidth))

	if snap != nil {
		cycleStr := fmt.Sprintf("💳 Billing Window : %s ➔ %s", snap.BillingCycle.Start.Format("2006-01-02"), snap.BillingCycle.End.Format("2006-01-02"))
		provStr := fmt.Sprintf("📊 Providers    : %d Active / %d Total", snap.ActiveOkCount, snap.TotalProviders)
		sb.WriteString(FormatTwoColumnBoxLine(cycleStr, 48, provStr, boxWidth))

		costStr := fmt.Sprintf("💰 Total Est Cost : $%.2f", snap.TotalCostUSD)
		statusStr := fmt.Sprintf("📡 Status       : %s", statusIndicator)
		sb.WriteString(FormatTwoColumnBoxLine(costStr, 48, statusStr, boxWidth))
	} else {
		sb.WriteString(FormatBoxLine("📡 Status         : Fetching initial usage telemetry...", boxWidth))
	}

	filterTag := "None"
	if filterQ != "" {
		filterTag = fmt.Sprintf("'%s'", filterQ)
	}
	filterStr := fmt.Sprintf("🏷️  Filter Mode   : %s", filterTag)
	sortStr := fmt.Sprintf("🔀 Sort Mode    : %s", sortMode.String())
	sb.WriteString(FormatTwoColumnBoxLine(filterStr, 48, sortStr, boxWidth))

	sb.WriteString(FormatBoxBottom(boxWidth))
	sb.WriteString("\n")

	if showHelp {
		sb.WriteString(FormatBoxTop(boxWidth))
		sb.WriteString(FormatBoxLine("⌨️  KEYBOARD SHORTCUTS & HELP", boxWidth))
		sb.WriteString(FormatBoxDivider(boxWidth))
		sb.WriteString(FormatBoxLine("• [q] or [Ctrl+C] : Quit and exit dashboard", boxWidth))
		sb.WriteString(FormatBoxLine("• [r]             : Trigger immediate live usage refresh", boxWidth))
		sb.WriteString(FormatBoxLine("• [↑] / [k]       : Move selection to previous provider row", boxWidth))
		sb.WriteString(FormatBoxLine("• [↓] / [j]       : Move selection to next provider row", boxWidth))
		sb.WriteString(FormatBoxLine("• [s]             : Cycle sort mode (Usage %, Consumed, Quota, Cost, Name)", boxWidth))
		sb.WriteString(FormatBoxLine("• [/] or [f]      : Filter providers by name / ID (type query, Enter to apply)", boxWidth))
		sb.WriteString(FormatBoxLine("• [Space]/[Enter] : Toggle detailed inspector panel for selected provider", boxWidth))
		sb.WriteString(FormatBoxLine("• [+] / [-]       : Increase / decrease auto-refresh interval (1s - 60s)", boxWidth))
		sb.WriteString(FormatBoxLine("• [h] or [?]      : Close this help overlay", boxWidth))
		sb.WriteString(FormatBoxBottom(boxWidth))
		sb.WriteString("\n")
	}

	if snap == nil {
		sb.WriteString("  ⏳ Gathering usage from active provider adapters...\n")
		sb.WriteString("\033[J") // Clear below
		_, _ = w.Write([]byte(sb.String()))
		return
	}

	providers := filterAndSortProviders(snap.Providers, filterQ, sortMode)

	if len(providers) == 0 {
		sb.WriteString(fmt.Sprintf("  ⚠️ No providers matching filter query: '%s'\n", filterQ))
		sb.WriteString("  Press '/' to update filter, or 'Esc' to clear.\n\n")
	} else {
		if selectedIdx >= len(providers) {
			selectedIdx = len(providers) - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}

		sb.WriteString(fmt.Sprintf("  %-3s %-4s %-18s %-24s %-11s %-14s %-10s %-8s %-14s %-10s %s\n",
			"SEL", "RANK", "PROVIDER", "MODEL / TIER", "CYCLE", "CONSUMED", "QUOTA", "USAGE %", "PROGRESS BAR", "EST. COST", "SOURCE / STATUS"))
		sb.WriteString(fmt.Sprintf("  %-3s %-4s %-18s %-24s %-11s %-14s %-10s %-8s %-14s %-10s %s\n",
			"---", "----", "--------", "------------", "-----", "--------", "-----", "-------", "------------", "---------", "---------------"))

		for i, p := range providers {
			cursor := "  "
			rowPrefix := ""
			rowSuffix := ""
			if i == selectedIdx {
				cursor = "► "
				rowPrefix = "\033[1;36m" // Cyan Bold
				rowSuffix = "\033[0m"
			}

			rank := fmt.Sprintf("#%d", i+1)
			cycleStr := "-"
			if !p.BillingStart.IsZero() && !p.BillingEnd.IsZero() {
				cycleStr = fmt.Sprintf("%s➔%s", p.BillingStart.Format("01/02"), p.BillingEnd.Format("01/02"))
			}

			if p.IsStale {
				sb.WriteString(fmt.Sprintf("%s%s%-3s %-18s %-24s %-11s %-14s %-10s %-8s %-14s %-10s ⚠️ STALE (%dd)%s\n",
					rowPrefix, cursor, rank, p.DisplayName, p.ModelOrTier, cycleStr, "0 "+string(p.Unit), p.FormatQuota(), "0.0%", "[············]", "$  0.00", p.DaysInactive, rowSuffix))
				continue
			}

			if p.Status != "ok" {
				errText := p.ErrorMessage
				if len(errText) > 20 {
					errText = errText[:17] + "..."
				}
				sb.WriteString(fmt.Sprintf("%s%s%-3s %-18s %-24s %-11s %-14s %-10s %-8s %-14s %-10s ❌ %s%s\n",
					rowPrefix, cursor, rank, p.DisplayName, "-", cycleStr, "-", "-", "-", "-", "-", errText, rowSuffix))
				continue
			}

			statusStr := "✔ OK"
			if p.DataSource != "" {
				if snap.IsLive {
					statusStr = fmt.Sprintf("🟢 %s", p.DataSource)
				} else {
					statusStr = fmt.Sprintf("🧪 %s", p.DataSource)
				}
			}

			bar := RenderProgressBar(p.PercentUsed, 12)
			costStr := fmt.Sprintf("$%6.2f", p.EstimatedCost)
			sb.WriteString(fmt.Sprintf("%s%s%-3s %-18s %-24s %-11s %-14s %-10s %6.1f%% %-14s %-10s %s%s\n",
				rowPrefix,
				cursor,
				rank,
				truncate(p.DisplayName, 18),
				truncate(p.ModelOrTier, 24),
				cycleStr,
				truncate(p.FormatConsumed(), 14),
				p.FormatQuota(),
				p.PercentUsed,
				bar,
				costStr,
				statusStr,
				rowSuffix,
			))
		}

		sb.WriteString("\n")

		// Inspector Card for Selected Provider
		if selectedIdx < len(providers) {
			sel := providers[selectedIdx]
			cycleStr := "-"
			if !sel.BillingStart.IsZero() && !sel.BillingEnd.IsZero() {
				cycleStr = fmt.Sprintf("%s to %s", sel.BillingStart.Format("2006-01-02"), sel.BillingEnd.Format("2006-01-02"))
			}

			if expanded {
				sb.WriteString(FormatBoxTop(boxWidth))
				sb.WriteString(FormatBoxLine("🔎 SELECTED PROVIDER INSPECTION (EXPANDED DETAILS)", boxWidth))
				sb.WriteString(FormatBoxDivider(boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Name         : %s", sel.DisplayName), 48, fmt.Sprintf("Provider ID  : %s", sel.ProviderID), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Model/Tier   : %s", truncate(sel.ModelOrTier, 32)), 48, fmt.Sprintf("Unit         : %s", string(sel.Unit)), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Consumed     : %s", sel.FormatConsumed()), 48, fmt.Sprintf("Quota Rem.   : %s", sel.FormatRemaining()), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Quota Total  : %s", sel.FormatQuota()), 48, fmt.Sprintf("Usage Ratio  : %.2f%%", sel.PercentUsed), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Est. Cost    : $%.2f", sel.EstimatedCost), 48, fmt.Sprintf("Last Polled  : %s", sel.LastUpdated.Format("15:04:05 MST")), boxWidth))
				sb.WriteString(FormatBoxLine(fmt.Sprintf("Cycle Window : %s", cycleStr), boxWidth))
				if sel.DataSource != "" {
					sb.WriteString(FormatBoxLine(fmt.Sprintf("Data Source  : %s", sel.DataSource), boxWidth))
				}
				if sel.Status != "ok" {
					sb.WriteString(FormatBoxLine(fmt.Sprintf("⚠️  Error      : %s", truncate(sel.ErrorMessage, 72)), boxWidth))
				}
				sb.WriteString(FormatBoxBottom(boxWidth))
				sb.WriteString("\n")
			} else {
				sb.WriteString(FormatBoxTop(boxWidth))
				sb.WriteString(FormatBoxLine("🔎 SELECTED PROVIDER INSPECTION (Press [Space] to expand details)", boxWidth))
				sb.WriteString(FormatBoxDivider(boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Name       : %s", sel.DisplayName), 48, fmt.Sprintf("Provider ID  : %s", sel.ProviderID), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Model/Tier : %s", truncate(sel.ModelOrTier, 32)), 48, fmt.Sprintf("Cycle Window : %s", cycleStr), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Consumed   : %s", sel.FormatConsumed()), 48, fmt.Sprintf("Quota Rem.   : %s", sel.FormatRemaining()), boxWidth))
				sb.WriteString(FormatTwoColumnBoxLine(fmt.Sprintf("Est. Cost  : $%.2f", sel.EstimatedCost), 48, fmt.Sprintf("Last Polled  : %s", sel.LastUpdated.Format("15:04:05 MST")), boxWidth))
				if sel.Status != "ok" {
					sb.WriteString(FormatBoxLine(fmt.Sprintf("⚠️  Error    : %s", truncate(sel.ErrorMessage, 72)), boxWidth))
				}
				sb.WriteString(FormatBoxBottom(boxWidth))
				sb.WriteString("\n")
			}
		}
	}

	// Interactive Footer Keybindings Bar
	sb.WriteString("\033[7m [q] Quit  [r] Refresh  [↑/↓/j/k] Select  [s] Sort  [/] Filter  [+/-] Interval  [h] Help \033[0m\n")

	sb.WriteString("\033[J") // Clear remaining lines at bottom
	_, _ = w.Write([]byte(sb.String()))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
