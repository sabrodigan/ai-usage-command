package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-usage/internal/config"
	"ai-usage/internal/core"
	"ai-usage/internal/credential"
)

// CredentialStatus represents whether a detected credential is newly discovered, updated, or already configured.
type CredentialStatus string

const (
	StatusNew        CredentialStatus = "new"        // Provider not in user's config
	StatusUpdatedKey CredentialStatus = "updated"    // Provider in config, but detected key/source is different
	StatusConfigured CredentialStatus = "configured" // Provider already in config with same credentials
)

// ScannedCredential holds discovered credential metadata, comparisons, and recommended defaults.
type ScannedCredential struct {
	ProviderID        string           `json:"provider_id"`
	DisplayName       string           `json:"display_name"`
	SourcePath        string           `json:"source_path"`
	SourceType        string           `json:"source_type"`     // "file", "env", "service"
	CredentialType    string           `json:"credential_type"` // "api_key" | "oauth" | "session"
	KeySnippet        string           `json:"key_snippet"`
	RawKey            string           `json:"-"`
	ModelTier         string           `json:"model_tier"`
	Endpoint          string           `json:"endpoint,omitempty"`
	SuggestedCycleDay int              `json:"suggested_cycle_day"`
	DefaultQuota      float64          `json:"default_quota"`
	Unit              core.MetricUnit  `json:"unit"`
	Status            CredentialStatus `json:"status"` // "new", "updated", "configured"
	ExistingQuota     float64          `json:"existing_quota,omitempty"`
	ExistingCycleDay  int              `json:"existing_cycle_day,omitempty"`
}

// DetectedCredential maintains backward compatibility with older callers.
type DetectedCredential = ScannedCredential

// ScanMachine inspects the machine and returns raw detected credentials.
func ScanMachine() []ScannedCredential {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	return ScanMachineWithDiff(cfg)
}

// ScanMachineWithDiff scans local files, running services, and env vars, then diffs against cfg.
func ScanMachineWithDiff(cfg *config.Config) []ScannedCredential {
	var rawFound []ScannedCredential
	home, _ := os.UserHomeDir()

	globalCycleDay := 1
	if cfg != nil && cfg.AnchorBillingDay > 0 {
		globalCycleDay = cfg.AnchorBillingDay
	}

	addOrUpdate := func(c ScannedCredential) {
		if c.CredentialType == "" {
			c.CredentialType = string(credential.Detect(c.RawKey))
		}
		for i, existing := range rawFound {
			if existing.ProviderID == c.ProviderID {
				// Prefer file / active session over env var if both present
				if existing.SourceType == "env" && c.SourceType != "env" {
					rawFound[i] = c
				}
				return
			}
		}
		rawFound = append(rawFound, c)
	}

	// 1. Antigravity Agent (Gemini)
	antigravityDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if info, err := os.Stat(antigravityDir); err == nil && info.IsDir() {
		addOrUpdate(ScannedCredential{
			ProviderID:        "antigravity",
			DisplayName:       "Antigravity Agent",
			SourcePath:        antigravityDir,
			SourceType:        "file (session telemetry)",
			KeySnippet:        "Local Session Active ✔",
			RawKey:            "",
			ModelTier:         "Agent Pro Runtime",
			SuggestedCycleDay: globalCycleDay,
			DefaultQuota:      10_000_000,
			Unit:              core.UnitTokens,
		})
	}

	// 2. Cursor IDE (~/.cursor or ~/.config/Cursor)
	cursorDir := filepath.Join(home, ".cursor")
	if info, err := os.Stat(cursorDir); err == nil && info.IsDir() {
		addOrUpdate(ScannedCredential{
			ProviderID:        "cursor",
			DisplayName:       "Cursor IDE",
			SourcePath:        cursorDir,
			SourceType:        "file (~/.cursor/ai-tracking)",
			KeySnippet:        "Active IDE Tracking ✔",
			RawKey:            "",
			ModelTier:         "Cursor Pro (Fast Requests)",
			SuggestedCycleDay: globalCycleDay,
			DefaultQuota:      500,
			Unit:              core.UnitRequests,
		})
	}

	// 2b. Meta Muse coding agent (~/.config/muse, sessions in ~/.local/share/muse).
	// Usage is read from local session journals, so the Meta token is never imported.
	museConfig := filepath.Join(home, ".config", "muse")
	museData := filepath.Join(home, ".local", "share", "muse")
	if _, err := os.Stat(filepath.Join(museConfig, "auth.json")); err == nil || dirExists(filepath.Join(museData, "sessions")) {
		tier := "Muse Spark"
		var settings struct {
			Model string `json:"model"`
		}
		if data, err := os.ReadFile(filepath.Join(museConfig, "settings.json")); err == nil {
			if json.Unmarshal(data, &settings) == nil && settings.Model != "" {
				tier = settings.Model
			}
		}
		addOrUpdate(ScannedCredential{
			ProviderID:        "muse",
			DisplayName:       "Meta Muse",
			SourcePath:        museData,
			SourceType:        "file (~/.local/share/muse sessions)",
			CredentialType:    string(credential.Session),
			KeySnippet:        "Local Session Active ✔",
			RawKey:            "",
			ModelTier:         tier,
			SuggestedCycleDay: globalCycleDay,
			DefaultQuota:      50_000_000,
			Unit:              core.UnitTokens,
		})
	}

	// 3. Warp Terminal (~/.warp)
	warpDir := filepath.Join(home, ".warp")
	if info, err := os.Stat(warpDir); err == nil && info.IsDir() {
		addOrUpdate(ScannedCredential{
			ProviderID:        "warp",
			DisplayName:       "Warp Terminal",
			SourcePath:        warpDir,
			SourceType:        "file (~/.warp)",
			KeySnippet:        "Terminal Installed ✔",
			RawKey:            "",
			ModelTier:         "Warp AI / Agent",
			SuggestedCycleDay: globalCycleDay,
			DefaultQuota:      100,
			Unit:              core.UnitRequests,
		})
	}

	// 4. OpenAI / Codex (~/.codex/auth.json or ~/.openai/api_key)
	codexAuth := filepath.Join(home, ".codex", "auth.json")
	if data, err := os.ReadFile(codexAuth); err == nil {
		var auth struct {
			OpenAIKey string `json:"OPENAI_API_KEY"`
			Tokens    struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		}
		_ = json.Unmarshal(data, &auth)
		key := auth.OpenAIKey
		credType := string(credential.APIKey)
		if key == "" {
			key = auth.Tokens.AccessToken
			credType = string(credential.OAuth)
		}
		if key != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        "codex",
				DisplayName:       "OpenAI Codex",
				SourcePath:        codexAuth,
				SourceType:        "file (~/.codex/auth.json)",
				CredentialType:    credType,
				KeySnippet:        maskKey(key),
				RawKey:            key,
				ModelTier:         "GPT-4o / Codex",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      4_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}
	openaiKeyFile := filepath.Join(home, ".openai", "api_key")
	if data, err := os.ReadFile(openaiKeyFile); err == nil {
		k := strings.TrimSpace(string(data))
		if k != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        "codex",
				DisplayName:       "OpenAI Codex",
				SourcePath:        openaiKeyFile,
				SourceType:        "file (~/.openai/api_key)",
				KeySnippet:        maskKey(k),
				RawKey:            k,
				ModelTier:         "GPT-4o / Codex",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      4_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}

	// 5. Anthropic Claude (~/.claude/.credentials.json or ~/.claude.json)
	claudeCreds := filepath.Join(home, ".claude", ".credentials.json")
	if data, err := os.ReadFile(claudeCreds); err == nil {
		var creds struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
			} `json:"claudeAiOauth"`
			ApiKey string `json:"apiKey"`
		}
		_ = json.Unmarshal(data, &creds)
		key := creds.ApiKey
		credType := string(credential.APIKey)
		if key == "" {
			key = creds.ClaudeAiOauth.AccessToken
			credType = string(credential.OAuth)
		}
		if key != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        "claude",
				DisplayName:       "Anthropic Claude",
				SourcePath:        claudeCreds,
				SourceType:        "file (~/.claude/.credentials.json)",
				CredentialType:    credType,
				KeySnippet:        maskKey(key),
				RawKey:            key,
				ModelTier:         "Claude 3.5 Sonnet / Opus",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      5_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}
	claudeAltCreds := filepath.Join(home, ".claude.json")
	if data, err := os.ReadFile(claudeAltCreds); err == nil {
		var creds struct {
			ApiKey string `json:"apiKey"`
		}
		_ = json.Unmarshal(data, &creds)
		if creds.ApiKey != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        "claude",
				DisplayName:       "Anthropic Claude",
				SourcePath:        claudeAltCreds,
				SourceType:        "file (~/.claude.json)",
				KeySnippet:        maskKey(creds.ApiKey),
				RawKey:            creds.ApiKey,
				ModelTier:         "Claude 3.5 Sonnet / Opus",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      5_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}

	// 6. Google Gemini (~/.gemini/oauth_creds.json)
	geminiCreds := filepath.Join(home, ".gemini", "oauth_creds.json")
	if data, err := os.ReadFile(geminiCreds); err == nil {
		var oauth struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		_ = json.Unmarshal(data, &oauth)
		key := oauth.AccessToken
		if key != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        "gemini",
				DisplayName:       "Google Gemini",
				SourcePath:        geminiCreds,
				SourceType:        "file (~/.gemini/oauth_creds.json)",
				CredentialType:    string(credential.OAuth),
				KeySnippet:        maskKey(key),
				RawKey:            key,
				ModelTier:         "Gemini 1.5 Pro / Flash",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      4_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}

	// 7. GitHub Copilot (~/.config/gh/hosts.yml)
	ghHosts := filepath.Join(home, ".config", "gh", "hosts.yml")
	if file, err := os.Open(ghHosts); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "oauth_token:") {
				token := strings.TrimSpace(strings.TrimPrefix(line, "oauth_token:"))
				if token != "" {
					addOrUpdate(ScannedCredential{
						ProviderID:        "copilot",
						DisplayName:       "GitHub Copilot",
						SourcePath:        ghHosts,
						SourceType:        "file (~/.config/gh/hosts.yml)",
						KeySnippet:        maskKey(token),
						RawKey:            token,
						ModelTier:         "Individual Plan",
						SuggestedCycleDay: globalCycleDay,
						DefaultQuota:      3000,
						Unit:              core.UnitRequests,
					})
					break
				}
			}
		}
	}

	// 8. Continue.dev config (~/.continue/config.json)
	continueConfig := filepath.Join(home, ".continue", "config.json")
	if data, err := os.ReadFile(continueConfig); err == nil {
		var cont struct {
			Models []struct {
				Title    string `json:"title"`
				Provider string `json:"provider"`
				APIKey   string `json:"apiKey"`
			} `json:"models"`
		}
		if err := json.Unmarshal(data, &cont); err == nil {
			for _, m := range cont.Models {
				if m.APIKey != "" {
					pID := strings.ToLower(m.Provider)
					if pID != "" {
						addOrUpdate(ScannedCredential{
							ProviderID:        pID,
							DisplayName:       strings.Title(pID),
							SourcePath:        continueConfig,
							SourceType:        "file (~/.continue/config.json)",
							KeySnippet:        maskKey(m.APIKey),
							RawKey:            m.APIKey,
							ModelTier:         m.Title,
							SuggestedCycleDay: globalCycleDay,
							DefaultQuota:      5_000_000,
							Unit:              core.UnitTokens,
						})
					}
				}
			}
		}
	}

	// 9. Ollama Local Endpoint Check (http://localhost:11434)
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://127.0.0.1:11434"
	}
	if !strings.HasPrefix(ollamaHost, "http://") && !strings.HasPrefix(ollamaHost, "https://") {
		ollamaHost = "http://" + ollamaHost
	}
	client := &http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(ollamaHost + "/api/tags")
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			addOrUpdate(ScannedCredential{
				ProviderID:        "ollama",
				DisplayName:       "Ollama Local",
				SourcePath:        ollamaHost,
				SourceType:        "local daemon (active)",
				KeySnippet:        "Local Server Active ✔",
				RawKey:            "ollama-local",
				Endpoint:          ollamaHost,
				ModelTier:         "Llama 3.3 / Qwen / Mistral",
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      50_000_000,
				Unit:              core.UnitTokens,
			})
		}
	}

	// 10. Environment Variables
	checkEnv := func(envVar, providerID, displayName, modelTier, endpoint string, quota float64, unit core.MetricUnit) {
		val := strings.TrimSpace(os.Getenv(envVar))
		if val != "" {
			addOrUpdate(ScannedCredential{
				ProviderID:        providerID,
				DisplayName:       displayName,
				SourcePath:        "$" + envVar,
				SourceType:        "env",
				KeySnippet:        maskKey(val),
				RawKey:            val,
				ModelTier:         modelTier,
				Endpoint:          endpoint,
				SuggestedCycleDay: globalCycleDay,
				DefaultQuota:      quota,
				Unit:              unit,
			})
		}
	}

	checkEnv("ANTHROPIC_API_KEY", "claude", "Anthropic Claude", "Claude 3.5 Sonnet", "https://api.anthropic.com", 5_000_000, core.UnitTokens)
	checkEnv("OPENAI_API_KEY", "codex", "OpenAI Codex", "GPT-4o", "https://api.openai.com/v1", 4_000_000, core.UnitTokens)
	checkEnv("GEMINI_API_KEY", "gemini", "Google Gemini", "Gemini 1.5 Pro", "https://generativelanguage.googleapis.com", 4_000_000, core.UnitTokens)
	checkEnv("GITHUB_TOKEN", "copilot", "GitHub Copilot", "Individual Plan", "https://api.github.com", 3000, core.UnitRequests)
	checkEnv("DEEPSEEK_API_KEY", "deepseek", "DeepSeek AI", "DeepSeek-V3", "https://api.deepseek.com", 10_000_000, core.UnitTokens)
	checkEnv("GROQ_API_KEY", "groq", "Groq Cloud", "Llama-3.3-70b-versatile", "https://api.groq.com/openai/v1", 10_000_000, core.UnitTokens)
	checkEnv("OPENROUTER_API_KEY", "openrouter", "OpenRouter", "Unified Multi-Model Gateway", "https://openrouter.ai/api/v1", 10_000_000, core.UnitTokens)
	checkEnv("MISTRAL_API_KEY", "mistral", "Mistral AI", "Mistral Large / Codestral", "https://api.mistral.ai/v1", 10_000_000, core.UnitTokens)
	checkEnv("PERPLEXITY_API_KEY", "perplexity", "Perplexity AI", "Sonar Online Pro", "https://api.perplexity.ai", 5_000_000, core.UnitTokens)
	checkEnv("XAI_API_KEY", "xai", "xAI (Grok)", "Grok 2 / Grok Vision", "https://api.x.ai/v1", 10_000_000, core.UnitTokens)
	checkEnv("GROK_API_KEY", "xai", "xAI (Grok)", "Grok 2 / Grok Vision", "https://api.x.ai/v1", 10_000_000, core.UnitTokens)
	checkEnv("TOGETHER_API_KEY", "together", "Together AI", "Llama / Mixtral Cloud", "https://api.together.xyz/v1", 10_000_000, core.UnitTokens)
	checkEnv("COHERE_API_KEY", "cohere", "Cohere", "Command R+", "https://api.cohere.com/v1", 5_000_000, core.UnitTokens)

	// Perform smart diff comparison against known configured providers in cfg
	var results []ScannedCredential
	for _, cred := range rawFound {
		if cfg == nil || cfg.Providers == nil {
			cred.Status = StatusNew
			results = append(results, cred)
			continue
		}

		existing, exists := cfg.Providers[cred.ProviderID]
		if !exists {
			cred.Status = StatusNew
			cred.SuggestedCycleDay = globalCycleDay
			results = append(results, cred)
			continue
		}

		cred.ExistingQuota = existing.CustomQuota
		cred.ExistingCycleDay = existing.AnchorBillingDay
		if cred.ExistingCycleDay <= 0 {
			cred.ExistingCycleDay = globalCycleDay
		}

		// For session-based providers (like antigravity, cursor, warp)
		if cred.RawKey == "" {
			if existing.Enabled {
				cred.Status = StatusConfigured
			} else {
				cred.Status = StatusNew
			}
			results = append(results, cred)
			continue
		}

		// For API key based providers, compare raw key with stored decrypted key
		decryptedKey := existing.GetDecryptedKey()
		if decryptedKey == cred.RawKey {
			cred.Status = StatusConfigured
		} else if decryptedKey == "" && cred.RawKey != "" {
			cred.Status = StatusUpdatedKey
		} else if decryptedKey != cred.RawKey {
			cred.Status = StatusUpdatedKey
		} else {
			cred.Status = StatusConfigured
		}

		results = append(results, cred)
	}

	return results
}

// FilterNewCredentials returns only credentials that are either completely new or have updated keys.
func FilterNewCredentials(scanned []ScannedCredential) []ScannedCredential {
	var news []ScannedCredential
	for _, c := range scanned {
		if c.Status == StatusNew || c.Status == StatusUpdatedKey {
			news = append(news, c)
		}
	}
	return news
}

// ImportSingleCredential adds or updates a single scanned provider credential into the config.
func ImportSingleCredential(cfg *config.Config, d ScannedCredential, quota float64, cycleDay int) error {
	if quota <= 0 {
		quota = d.DefaultQuota
	}
	if cycleDay <= 0 || cycleDay > 28 {
		cycleDay = d.SuggestedCycleDay
	}
	if cycleDay <= 0 || cycleDay > 28 {
		cycleDay = 1
	}

	keyToStore := d.RawKey
	credType := d.CredentialType
	if keyToStore == "" {
		if existing, ok := cfg.Providers[d.ProviderID]; ok {
			keyToStore = existing.APIKey
			if credType == "" {
				credType = existing.ResolvedCredentialType()
			}
		}
	}
	if credType == "" {
		credType = string(credential.Detect(keyToStore))
	}

	return cfg.SetProvider(config.ProviderConfig{
		ID:               d.ProviderID,
		DisplayName:      d.DisplayName,
		APIKey:           keyToStore,
		CredentialType:   credType,
		CustomQuota:      quota,
		ModelTier:        d.ModelTier,
		Endpoint:         d.Endpoint,
		AnchorBillingDay: cycleDay,
		Enabled:          true,
	})
}

// ImportCredentials encrypts and saves detected credentials into the app config.
func ImportCredentials(cfg *config.Config, detected []ScannedCredential) (int, error) {
	importedCount := 0
	for _, d := range detected {
		quota := d.DefaultQuota
		if existing, exists := cfg.Providers[d.ProviderID]; exists && existing.CustomQuota > 0 {
			quota = existing.CustomQuota
		}

		cycleDay := d.SuggestedCycleDay
		if existing, exists := cfg.Providers[d.ProviderID]; exists && existing.AnchorBillingDay > 0 {
			cycleDay = existing.AnchorBillingDay
		}

		if err := ImportSingleCredential(cfg, d, quota, cycleDay); err == nil {
			importedCount++
		}
	}

	if err := config.Save(cfg); err != nil {
		return 0, fmt.Errorf("failed saving config: %w", err)
	}
	return importedCount, nil
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "***"
	}
	if len(k) > 20 {
		return k[:6] + "..." + k[len(k)-4:]
	}
	return k[:4] + "..." + k[len(k)-2:]
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
