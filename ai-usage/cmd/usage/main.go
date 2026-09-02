package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ai-usage/internal/analytics"
	"ai-usage/internal/config"
	"ai-usage/internal/core"
	"ai-usage/internal/provider"
	"ai-usage/internal/scanner"
	"ai-usage/internal/storage"
	"ai-usage/internal/ui"
)

const Version = "2.1.0"

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

func main() {
	if len(os.Args) < 2 {
		runNow(nil)
		return
	}

	command := os.Args[1]

	switch command {
	case "live":
		runLive(os.Args[2:])
	case "example", "demo", "mock":
		runExample(os.Args[2:])
	case "now":
		runNow(os.Args[2:])
	case "watch", "tui", "top", "dashboard":
		runWatch(os.Args[2:])
	case "scan":
		runScan(os.Args[2:])
	case "scan-new":
		runScanNew(os.Args[2:])
	case "recommend", "recommendations", "insights":
		runRecommend(os.Args[2:])
	case "stats", "trends":
		runStats(os.Args[2:])
	case "keys", "key", "show-keys", "list-keys":
		runKeys(os.Args[2:])
	case "providers":
		runProviders(os.Args[2:])
	case "add-provider":
		runAddProvider(os.Args[2:])
	case "update-key":
		runUpdateKey(os.Args[2:])
	case "prune-stale":
		runPruneStale(os.Args[2:])
	case "remove-provider":
		runRemoveProvider(os.Args[2:])
	case "set-cycle":
		runSetCycle(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	case "history":
		runHistory(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("ai-usage CLI v%s (SQLite Analytics + Live Dashboard + AI Intelligence)\n", Version)
	case "help", "--help", "-h":
		printUsageHelp()
	default:
		if strings.HasPrefix(command, "-") {
			runNow(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: '%s'\nRun 'usage help' for available commands.\n", command)
		os.Exit(1)
	}
}

func getActiveTasks(cfg *config.Config, now time.Time) []core.ProviderTask {
	var tasks []core.ProviderTask

	getWin := func(p config.ProviderConfig) core.BillingWindow {
		day := p.AnchorBillingDay
		if day <= 0 {
			day = cfg.AnchorBillingDay
		}
		if day <= 0 {
			day = 1
		}
		return core.CalculateBillingCycle(now, day)
	}

	// Antigravity adapter
	if p, exists := cfg.Providers["antigravity"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewAntigravityAdapter(p.CustomQuota),
			Window:  getWin(p),
		})
	} else if !exists {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewAntigravityAdapter(10_000_000),
			Window:  core.CalculateBillingCycle(now, 1),
		})
	}

	// Cursor IDE adapter
	if p, exists := cfg.Providers["cursor"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewCursorAdapter(p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// Warp Terminal adapter
	if p, exists := cfg.Providers["warp"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewWarpAdapter(p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// Claude adapter
	if p, exists := cfg.Providers["claude"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewClaudeAdapter(p.GetDecryptedKey(), p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// OpenAI / Codex adapter
	if p, exists := cfg.Providers["codex"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewCodexAdapter(p.GetDecryptedKey(), p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// Gemini adapter
	if p, exists := cfg.Providers["gemini"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewGeminiAdapter(p.GetDecryptedKey(), p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// GitHub Copilot adapter
	if p, exists := cfg.Providers["copilot"]; exists && p.Enabled {
		tasks = append(tasks, core.ProviderTask{
			Adapter: provider.NewCopilotAdapter(p.GetDecryptedKey(), p.CustomQuota, p.ModelTier),
			Window:  getWin(p),
		})
	}

	// Custom registered providers
	for id, p := range cfg.Providers {
		if id == "antigravity" || id == "cursor" || id == "warp" || id == "claude" || id == "codex" || id == "gemini" || id == "copilot" {
			continue
		}
		if p.Enabled {
			tasks = append(tasks, core.ProviderTask{
				Adapter: provider.NewCustomAdapter(
					p.ID,
					p.DisplayName,
					p.Endpoint,
					p.GetDecryptedKey(),
					p.CustomQuota,
					p.ModelTier,
				),
				Window: getWin(p),
			})
		}
	}

	return tasks
}

func runUsageSnapshot(args []string, liveMode bool, cmdName string) {
	fs := flag.NewFlagSet(cmdName, flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output usage as JSON")
	csvFlag := fs.Bool("csv", false, "Output usage as CSV")
	timeoutSec := fs.Int("timeout", 6, "Query timeout in seconds")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	if *timeoutSec > 0 {
		cfg.TimeoutSeconds = *timeoutSec
	}

	now := time.Now()
	defaultWindow := core.CalculateBillingCycle(now, cfg.AnchorBillingDay)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	tasks := getActiveTasks(cfg, now)
	snapshot := core.FetchAndAggregate(ctx, tasks, defaultWindow, liveMode)

	// Check for stale providers (>= 30 days inactive)
	for i, p := range snapshot.Providers {
		if cfgP, exists := cfg.Providers[p.ProviderID]; exists {
			if stale, days := cfgP.IsStale(); stale {
				snapshot.Providers[i].IsStale = true
				snapshot.Providers[i].DaysInactive = days
				snapshot.StaleCount++
			}
		}
	}

	if liveMode {
		// Persist live snapshots to embedded SQLite database (~/.config/ai-usage/usage.db)
		if sqliteStore, err := storage.NewSQLiteStore(); err == nil {
			_ = sqliteStore.SaveSnapshot(context.Background(), snapshot)
			_ = sqliteStore.Close()
		}

		// Also backup to legacy JSON history
		if store, err := storage.NewFileStore(); err == nil {
			_ = store.SaveSnapshot(context.Background(), snapshot)
		}
	}

	if *jsonFlag {
		_ = ui.RenderJSON(os.Stdout, snapshot)
	} else if *csvFlag {
		_ = ui.RenderCSV(os.Stdout, snapshot)
	} else {
		ui.RenderTable(os.Stdout, snapshot)
	}
}

func runLive(args []string) {
	runUsageSnapshot(args, true, "live")
}

func runExample(args []string) {
	runUsageSnapshot(args, false, "example")
}

func runNow(args []string) {
	runUsageSnapshot(args, true, "now")
}

func runRecommend(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	defaultWindow := core.CalculateBillingCycle(now, cfg.AnchorBillingDay)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	tasks := getActiveTasks(cfg, now)
	snapshot := core.FetchAndAggregate(ctx, tasks, defaultWindow, true)

	recs := analytics.GenerateInsights(snapshot)
	ui.RenderRecommendations(os.Stdout, recs)
}

func runStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	days := fs.Int("days", 30, "Lookback days for historical trends")
	_ = fs.Parse(args)

	sqliteStore, err := storage.NewSQLiteStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open SQLite store: %v\n", err)
		os.Exit(1)
	}
	defer sqliteStore.Close()

	summaries, err := sqliteStore.GetDailyTrends(context.Background(), *days)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying trends: %v\n", err)
		return
	}

	ui.RenderDailyTrends(os.Stdout, summaries)
}

func runKeys(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	ui.RunKeysManager(os.Stdin, os.Stdout, cfg)
}

func runWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	intervalSec := fs.Int("interval", 3, "Live auto-refresh interval in seconds")
	filterFlag := fs.String("filter", "", "Initial provider name/ID filter")
	timeoutSec := fs.Int("timeout", 6, "Timeout for adapter queries")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if *timeoutSec > 0 {
		cfg.TimeoutSeconds = *timeoutSec
	}

	// 1. Auto-discover any new/updated providers installed on the machine prior to starting dashboard
	scanned := scanner.ScanMachineWithDiff(cfg)
	newCreds := scanner.FilterNewCredentials(scanned)
	if len(newCreds) > 0 {
		fmt.Printf("🔍 Auto-discovery: Found %d new/updated AI provider(s) on machine. Auto-syncing...\n", len(newCreds))
		importedCount, err := scanner.ImportCredentials(cfg, newCreds)
		if err == nil && importedCount > 0 {
			fmt.Printf("✔ Auto-synced %d newly discovered provider(s) into live monitoring!\n", importedCount)
			time.Sleep(500 * time.Millisecond)
		}
	}

	fetcher := func(ctx context.Context) *core.UsageSnapshot {
		// Reload config dynamically in case providers or cycles changed
		liveCfg, err := config.Load()
		if err != nil {
			liveCfg = cfg
		}

		now := time.Now()
		defaultWindow := core.CalculateBillingCycle(now, liveCfg.AnchorBillingDay)
		tasks := getActiveTasks(liveCfg, now)
		snapshot := core.FetchAndAggregate(ctx, tasks, defaultWindow, true)

		for i, p := range snapshot.Providers {
			if cfgP, exists := liveCfg.Providers[p.ProviderID]; exists {
				if stale, days := cfgP.IsStale(); stale {
					snapshot.Providers[i].IsStale = true
					snapshot.Providers[i].DaysInactive = days
					snapshot.StaleCount++
				}
			}
		}

		// Save snapshot to SQLite in background
		if sqliteStore, err := storage.NewSQLiteStore(); err == nil {
			_ = sqliteStore.SaveSnapshot(context.Background(), snapshot)
			_ = sqliteStore.Close()
		}

		return snapshot
	}

	err = ui.RunDashboard(fetcher, time.Duration(*intervalSec)*time.Second, *filterFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Dashboard error: %v\n", err)
	}
}

func runScan(args []string) {
	if len(args) > 0 && (args[0] == "new" || args[0] == "--new") {
		runScanNew(args[1:])
		return
	}

	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	importFlag := fs.Bool("import", false, "Automatically import and encrypt all found new/updated credentials")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	scanned := scanner.ScanMachineWithDiff(cfg)
	ui.RenderScanAll(os.Stdout, scanned, cfg.AnchorBillingDay)

	newCreds := scanner.FilterNewCredentials(scanned)

	if *importFlag {
		if len(newCreds) == 0 {
			fmt.Println("✔ All detected providers are already configured.")
			return
		}
		count, err := scanner.ImportCredentials(cfg, newCreds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving imported credentials: %v\n", err)
			return
		}
		fmt.Printf("✔ Successfully imported and AES-256 encrypted %d new provider configuration(s)!\n", count)
		fmt.Println("Run 'usage' to view your updated live report.")
	} else if len(newCreds) > 0 {
		fmt.Printf("💡 Found %d new or updated provider credential(s) on this machine.\n", len(newCreds))
		fmt.Println("   • To interactively review & add them (with cycle date & quota): usage scan new")
		fmt.Println("   • To automatically import all new providers with defaults      : usage scan --import")
		fmt.Println()
	}
}

func runScanNew(args []string) {
	fs := flag.NewFlagSet("scan new", flag.ExitOnError)
	yesFlag := fs.Bool("yes", false, "Automatically accept and import all newly discovered providers")
	fs.BoolVar(yesFlag, "y", false, "Alias for --yes")
	fs.BoolVar(yesFlag, "import", false, "Alias for --yes")
	jsonFlag := fs.Bool("json", false, "Output newly detected credentials as JSON")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		return
	}

	now := time.Now()
	scanned := scanner.ScanMachineWithDiff(cfg)
	newCreds := scanner.FilterNewCredentials(scanned)

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(newCreds)
		return
	}

	ui.RenderScanNewCards(os.Stdout, newCreds, cfg.AnchorBillingDay, now)

	if len(newCreds) == 0 {
		return
	}

	if *yesFlag {
		count, err := scanner.ImportCredentials(cfg, newCreds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving imported credentials: %v\n", err)
			return
		}
		fmt.Printf("✔ Successfully imported and AES-256 encrypted %d provider configuration(s)!\n", count)
		fmt.Println("Run 'usage' to view your updated live report.")
		return
	}

	// Interactive mode
	reader := bufio.NewReader(os.Stdin)
	importedCount := 0
	addAllRemaining := false

	for _, c := range newCreds {
		if addAllRemaining {
			if err := scanner.ImportSingleCredential(cfg, c, c.DefaultQuota, c.SuggestedCycleDay); err == nil {
				importedCount++
			}
			continue
		}

		quotaStr := core.FormatNumber(c.DefaultQuota)
		if c.Unit == core.UnitRequests {
			quotaStr = fmt.Sprintf("%.0f requests", c.DefaultQuota)
		} else if c.Unit == core.UnitTokens {
			quotaStr = quotaStr + " tokens"
		}

		fmt.Printf("👉 Would you like to add provider '%s' (%s, Quota: %s, Cycle: %s)?\n",
			c.DisplayName, c.ModelTier, quotaStr, ordinal(c.SuggestedCycleDay))
		fmt.Print("   [Y]es / [n]o / [e]dit / [a]ll / [s]kip (Default: Y): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			// Non-interactive or EOF
			break
		}
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" || input == "y" || input == "yes" {
			if err := scanner.ImportSingleCredential(cfg, c, c.DefaultQuota, c.SuggestedCycleDay); err == nil {
				importedCount++
				fmt.Printf("   ✔ Added and encrypted '%s' (Cycle resets on %s).\n\n", c.DisplayName, ordinal(c.SuggestedCycleDay))
			}
		} else if input == "a" || input == "all" {
			addAllRemaining = true
			if err := scanner.ImportSingleCredential(cfg, c, c.DefaultQuota, c.SuggestedCycleDay); err == nil {
				importedCount++
				fmt.Printf("   ✔ Added and encrypted '%s' (Cycle resets on %s).\n\n", c.DisplayName, ordinal(c.SuggestedCycleDay))
			}
		} else if input == "e" || input == "edit" {
			quota := c.DefaultQuota
			cycleDay := c.SuggestedCycleDay

			fmt.Printf("   Enter monthly quota [%v]: ", c.DefaultQuota)
			qInput, _ := reader.ReadString('\n')
			qInput = strings.TrimSpace(qInput)
			if qInput != "" {
				if qVal, err := strconv.ParseFloat(qInput, 64); err == nil && qVal > 0 {
					quota = qVal
				}
			}

			fmt.Printf("   Enter monthly billing cycle reset day (1-28) [%d]: ", cycleDay)
			cInput, _ := reader.ReadString('\n')
			cInput = strings.TrimSpace(cInput)
			if cInput != "" {
				if cVal, err := strconv.Atoi(cInput); err == nil && cVal >= 1 && cVal <= 28 {
					cycleDay = cVal
				}
			}

			if err := scanner.ImportSingleCredential(cfg, c, quota, cycleDay); err == nil {
				importedCount++
				fmt.Printf("   ✔ Added '%s' with custom quota %s and cycle reset on %s.\n\n",
					c.DisplayName, core.FormatNumber(quota), ordinal(cycleDay))
			}
		} else {
			fmt.Printf("   ⏩ Skipped '%s'.\n\n", c.DisplayName)
		}
	}

	if importedCount > 0 {
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			return
		}
		fmt.Printf("🎉 Finished! Successfully added %d provider(s) to your usage tracker.\n", importedCount)
		fmt.Println("Run 'usage' to view current consumption or 'usage watch' for the live dashboard.")
	}
}

func runProviders(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Configured AI Model Providers:")
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("%-15s %-20s %-10s %-12s %-12s %-12s %s\n", "ID", "NAME", "STATUS", "QUOTA", "RESET DAY", "ENCRYPTED", "MODEL / TIER")
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────")

	for id, p := range cfg.Providers {
		status := "Active"
		if !p.Enabled {
			status = "Disabled"
		} else if stale, days := p.IsStale(); stale {
			status = fmt.Sprintf("Stale(%dd)", days)
		}

		quotaStr := core.FormatNumber(p.CustomQuota)
		if p.CustomQuota <= 0 {
			quotaStr = "Default"
		}

		day := p.AnchorBillingDay
		if day <= 0 {
			day = cfg.AnchorBillingDay
		}
		resetStr := ordinal(day)

		isEncrypted := "No Key"
		if p.APIKey != "" {
			if strings.HasPrefix(p.APIKey, "enc:v1:") {
				isEncrypted = "AES-256 ✔"
			} else {
				isEncrypted = "Plain"
			}
		}

		fmt.Printf("%-15s %-20s %-10s %-12s %-12s %-12s %s\n", id, p.DisplayName, status, quotaStr, resetStr, isEncrypted, p.ModelTier)
	}
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("Default global billing cycle anchor: %s of each month.\n\n", ordinal(cfg.AnchorBillingDay))
}

func runAddProvider(args []string) {
	fs := flag.NewFlagSet("add-provider", flag.ExitOnError)
	id := fs.String("id", "", "Provider identifier (e.g., claude, codex, cursor, warp, deepseek)")
	name := fs.String("name", "", "Human readable display name")
	key := fs.String("key", "", "API Key or Token (stored encrypted with AES-256)")
	quota := fs.Float64("quota", 5_000_000, "Monthly quota (tokens/requests)")
	tier := fs.String("tier", "", "Model or plan tier description")
	endpoint := fs.String("endpoint", "", "Custom API endpoint URL")
	cycleDay := fs.Int("cycle", 1, "Billing cycle reset day of the month (1-28)")
	_ = fs.Parse(args)

	if *id == "" {
		nonFlags := fs.Args()
		if len(nonFlags) >= 2 {
			*id = nonFlags[0]
			*key = nonFlags[1]
			if len(nonFlags) >= 3 {
				if q, err := strconv.ParseFloat(nonFlags[2], 64); err == nil {
					*quota = q
				}
			}
		} else {
			fmt.Println("Error: Provider ID is required.")
			fmt.Println("Usage: usage add-provider --id <id> --key <api_key> [--quota <tokens>] [--name <name>] [--cycle <1-28>]")
			fmt.Println("Example: usage add-provider --id deepseek --name 'DeepSeek AI' --key sk-xxx --quota 10000000 --cycle 14")
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	displayName := *name
	if displayName == "" {
		if existing, ok := cfg.Providers[*id]; ok && existing.DisplayName != "" {
			displayName = existing.DisplayName
		} else {
			displayName = strings.ToUpper((*id)[:1]) + (*id)[1:]
		}
	}

	modelTier := *tier
	if modelTier == "" {
		if existing, ok := cfg.Providers[*id]; ok && existing.ModelTier != "" {
			modelTier = existing.ModelTier
		} else {
			modelTier = "Standard Tier"
		}
	}

	err = cfg.SetProvider(config.ProviderConfig{
		ID:               *id,
		DisplayName:      displayName,
		APIKey:           *key,
		CustomQuota:      *quota,
		ModelTier:        modelTier,
		Endpoint:         *endpoint,
		AnchorBillingDay: *cycleDay,
		Enabled:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure provider: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save provider config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✔ Provider '%s' (%s) configured & encrypted (Cycle resets on %s).\n", *id, displayName, ordinal(*cycleDay))
}

func runUpdateKey(args []string) {
	fs := flag.NewFlagSet("update-key", flag.ExitOnError)
	id := fs.String("id", "", "Provider identifier (e.g., claude, codex)")
	key := fs.String("key", "", "New API Key or Token")
	_ = fs.Parse(args)

	if *id == "" || *key == "" {
		nonFlags := fs.Args()
		if len(nonFlags) >= 2 {
			*id = nonFlags[0]
			*key = nonFlags[1]
		} else {
			fmt.Println("Usage: usage update-key <provider_id> <new_api_key>")
			fmt.Println("   or: usage update-key --id <provider_id> --key <new_api_key>")
			fmt.Println("Example: usage update-key claude sk-ant-api03-new-key-xxx")
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.UpdateKey(*id, *key); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating key: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save updated config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✔ API Key for provider '%s' replaced and encrypted with AES-256 (Activity reset).\n", *id)
}

func runSetCycle(args []string) {
	fs := flag.NewFlagSet("set-cycle", flag.ExitOnError)
	idFlag := fs.String("id", "", "Specific provider ID (e.g. claude, codex, cursor, warp)")
	dayFlag := fs.Int("day", 0, "Day of the month the cycle resets (1-28)")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	nonFlags := fs.Args()

	if len(nonFlags) >= 2 {
		id := nonFlags[0]
		day, err := strconv.Atoi(nonFlags[1])
		if err != nil || day < 1 || day > 28 {
			fmt.Println("Error: Cycle day must be a number between 1 and 28.")
			os.Exit(1)
		}
		if err := cfg.SetProviderCycle(id, day); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		_ = config.Save(cfg)
		fmt.Printf("✔ Billing cycle for '%s' set to reset on the %s of each month.\n", id, ordinal(day))
		return
	}

	if *idFlag != "" && *dayFlag > 0 {
		if *dayFlag < 1 || *dayFlag > 28 {
			fmt.Println("Error: Cycle day must be between 1 and 28.")
			os.Exit(1)
		}
		if err := cfg.SetProviderCycle(*idFlag, *dayFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		_ = config.Save(cfg)
		fmt.Printf("✔ Billing cycle for '%s' set to reset on the %s of each month.\n", *idFlag, ordinal(*dayFlag))
		return
	}

	if len(nonFlags) == 1 {
		day, err := strconv.Atoi(nonFlags[0])
		if err != nil || day < 1 || day > 28 {
			fmt.Println("Error: Global cycle day must be between 1 and 28.")
			os.Exit(1)
		}
		cfg.AnchorBillingDay = day
		_ = config.Save(cfg)
		fmt.Printf("✔ Default global billing cycle set to reset on the %s of each month.\n", ordinal(day))
		return
	}

	if *dayFlag > 0 {
		if *dayFlag < 1 || *dayFlag > 28 {
			fmt.Println("Error: Global cycle day must be between 1 and 28.")
			os.Exit(1)
		}
		cfg.AnchorBillingDay = *dayFlag
		_ = config.Save(cfg)
		fmt.Printf("✔ Default global billing cycle set to reset on the %s of each month.\n", ordinal(*dayFlag))
		return
	}

	fmt.Println("Usage:")
	fmt.Println("  usage set-cycle <provider_id> <day_1_to_28>    # Set cycle for specific provider")
	fmt.Println("  usage set-cycle <day_1_to_28>                  # Set global default cycle")
}

func runPruneStale(args []string) {
	fs := flag.NewFlagSet("prune-stale", flag.ExitOnError)
	days := fs.Int("days", 30, "Inactivity threshold in days")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	pruned := cfg.PruneStaleProviders(*days)
	if len(pruned) == 0 {
		fmt.Printf("✔ No stale providers found (all active within the last %d days).\n", *days)
		return
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✔ Pruned %d inactive provider(s): %s\n", len(pruned), strings.Join(pruned, ", "))
}

func runRemoveProvider(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: usage remove-provider <id>")
		return
	}
	id := args[0]
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.RemoveProvider(id) {
		_ = config.Save(cfg)
		fmt.Printf("✔ Provider '%s' removed.\n", id)
	} else {
		fmt.Printf("Provider '%s' was not found.\n", id)
	}
}

func runExport(args []string) {
	runNow(args)
}

func runHistory(args []string) {
	store, err := storage.NewFileStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open history store: %v\n", err)
		os.Exit(1)
	}

	history, err := store.GetHistory(context.Background(), 5)
	if err != nil || len(history) == 0 {
		fmt.Println("No historical snapshots recorded yet. Run 'usage now' first.")
		return
	}

	fmt.Println()
	fmt.Println("Recent Usage Snapshots:")
	fmt.Println("──────────────────────────────────────────────────────────────────────")
	for i, h := range history {
		fmt.Printf("[%d] %s | Active Providers: %d | Total Cost: $%.2f\n",
			i+1,
			h.Timestamp.Local().Format("2006-01-02 15:04:05"),
			h.ActiveOkCount,
			h.TotalCostUSD,
		)
	}
	fmt.Println("──────────────────────────────────────────────────────────────────────")
	fmt.Println()
}

func printUsageHelp() {
	fmt.Print(`
AI Usage CLI Tracker (SQLite Analytics + Live Dashboard + AI Intelligence)
Monitor, aggregate, and rank developer AI model consumption.

Usage:
  usage [command] [flags]

Core Commands:
  live                        Query and display real-time live data from APIs & logs (default)
  example                     Show representative simulation / benchmark demo data
  now                         Alias for live usage report
  watch [--interval <sec>]    Open interactive live auto-refreshing dashboard / TUI (auto-syncs machine)
  keys                        Interactive manager to view, reveal/hide, and inspect private API keys
  scan                        Scan machine environment & display configured vs new credentials
  scan new [--yes] [--json]   Scan and interactively review & add new AI providers with cycle/quota
  recommend                   Show AI cost optimization recommendations & insights
  stats [--days <n>]          Show historical consumption trends from local SQLite DB
  providers                   List all configured provider endpoints, reset days & encryption

Live Dashboard Shortcuts (usage watch / usage tui):
  [q] Quit   [r] Refresh   [↑/↓/j/k] Select   [s] Sort   [/] Filter   [+/-] Interval   [h] Help

Management Commands:
  add-provider                Add or configure a provider API key, quota & cycle
  update-key <id> <key>       Replace/update an existing provider's API key
  set-cycle <id> <1-28>       Set billing reset day for a specific provider
  set-cycle <1-28>            Set default global billing reset day
  prune-stale [--days]        Clean up providers with no usage for 30+ days
  remove-provider <id>        Remove a provider configuration
  export --json | --csv       Export raw usage snapshot
  history                     Show recent usage snapshots
  version                     Display version information
  help                        Show this help message

Local Database:
  Data is stored locally in ~/.config/ai-usage/usage.db (pure Go embedded SQLite).
`)
}
