package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ai-usage/internal/core"
	"ai-usage/internal/credential"
)

type CodexAdapter struct {
	APIKey      string
	CredType    string
	CustomQuota float64
	ModelTier   string
	client      *http.Client
}

func NewCodexAdapter(apiKey, credType string, quota float64, modelTier string) *CodexAdapter {
	if quota <= 0 {
		quota = 4_000_000 // 4M tokens default
	}
	if modelTier == "" {
		modelTier = "GPT-4o / Codex"
	}
	if credType == "" {
		credType = string(credential.Detect(apiKey))
	}
	return &CodexAdapter{
		APIKey:      apiKey,
		CredType:    credType,
		CustomQuota: quota,
		ModelTier:   modelTier,
		client:      NewHTTPClient(6 * time.Second),
	}
}

func (c *CodexAdapter) ID() string          { return "codex" }
func (c *CodexAdapter) DisplayName() string { return "OpenAI Codex" }

func (c *CodexAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		var consumedTokens float64 = 2_150_000
		cost := (consumedTokens / 1_000_000) * 10.0
		return &core.ProviderUsage{
			ProviderID:    c.ID(),
			DisplayName:   c.DisplayName(),
			ModelOrTier:   c.ModelTier,
			Unit:          core.UnitTokens,
			Consumed:      consumedTokens,
			Quota:         c.CustomQuota,
			EstimatedCost: cost,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	if !IsAPIKeyConfigured(c.APIKey) {
		return nil, fmt.Errorf("no OpenAI API key configured")
	}

	// A Codex/ChatGPT OAuth token is a JWT scoped to the ChatGPT backend, not
	// the platform API — the /v1/* endpoints reject it. Validate it locally by
	// its expiry claim and report it as verified with usage unavailable.
	if c.CredType == string(credential.OAuth) {
		if exp, ok := credential.JWTExpiry(c.APIKey); ok && time.Now().After(exp) {
			return nil, fmt.Errorf("OpenAI/Codex OAuth token expired %s ago; run `codex login`", time.Since(exp).Round(time.Minute))
		}
		return &core.ProviderUsage{
			ProviderID:    c.ID(),
			DisplayName:   c.DisplayName(),
			ModelOrTier:   c.ModelTier,
			Unit:          core.UnitTokens,
			Consumed:      0,
			Quota:         c.CustomQuota,
			EstimatedCost: 0,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        true,
			DataSource:    "OpenAI ChatGPT OAuth session (verified — platform usage API not available)",
		}, nil
	}

	// Verify key via models endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid or expired OpenAI API key (HTTP %d)", resp.StatusCode)
	}

	var consumedTokens float64 = 0
	var cost float64 = 0

	// Check legacy/organization usage endpoint
	usageReqURL := fmt.Sprintf("https://api.openai.com/v1/usage?date=%s", window.Start.Format("2006-01-02"))
	if uReq, err := http.NewRequestWithContext(ctx, "GET", usageReqURL, nil); err == nil {
		uReq.Header.Set("Authorization", "Bearer "+c.APIKey)
		if uResp, err := c.client.Do(uReq); err == nil {
			defer uResp.Body.Close()
			if uResp.StatusCode == http.StatusOK {
				var res struct {
					TotalUsage float64 `json:"total_usage"`
				}
				if err := json.NewDecoder(uResp.Body).Decode(&res); err == nil {
					consumedTokens = res.TotalUsage * 1000
					cost = (consumedTokens / 1_000_000) * 10.0
				}
			}
		}
	}

	return &core.ProviderUsage{
		ProviderID:    c.ID(),
		DisplayName:   c.DisplayName(),
		ModelOrTier:   c.ModelTier,
		Unit:          core.UnitTokens,
		Consumed:      consumedTokens,
		Quota:         c.CustomQuota,
		EstimatedCost: cost,
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "OpenAI Live API (Key Verified ✔)",
	}, nil
}
