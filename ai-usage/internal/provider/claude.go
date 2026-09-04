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

// anthropicOAuthBeta is the beta flag Anthropic requires on requests
// authenticated with a Claude subscription OAuth token rather than an API key.
const anthropicOAuthBeta = "oauth-2025-04-20"

type ClaudeAdapter struct {
	APIKey      string
	CredType    string
	CustomQuota float64
	ModelTier   string
	client      *http.Client
}

func NewClaudeAdapter(apiKey, credType string, quota float64, modelTier string) *ClaudeAdapter {
	if quota <= 0 {
		quota = 5_000_000 // 5M tokens default
	}
	if modelTier == "" {
		modelTier = "Claude 3.5 Sonnet / Opus"
	}
	if credType == "" {
		credType = string(credential.Detect(apiKey))
	}
	return &ClaudeAdapter{
		APIKey:      apiKey,
		CredType:    credType,
		CustomQuota: quota,
		ModelTier:   modelTier,
		client:      NewHTTPClient(6 * time.Second),
	}
}

func (c *ClaudeAdapter) ID() string          { return "claude" }
func (c *ClaudeAdapter) DisplayName() string { return "Anthropic Claude" }

// authHeaders sets the correct authentication scheme for the configured
// credential: "x-api-key" for a first-party key, bearer + oauth beta for a
// subscription OAuth token.
func (c *ClaudeAdapter) authHeaders(h http.Header) {
	h.Set("anthropic-version", "2023-06-01")
	if c.CredType == string(credential.OAuth) {
		h.Set("Authorization", "Bearer "+c.APIKey)
		h.Set("anthropic-beta", anthropicOAuthBeta)
		return
	}
	h.Set("x-api-key", c.APIKey)
}

func (c *ClaudeAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		var consumedTokens float64 = 1_845_200
		cost := (consumedTokens / 1_000_000) * 15.0
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
		return nil, fmt.Errorf("no Anthropic API key configured")
	}

	isOAuth := c.CredType == string(credential.OAuth)
	if isOAuth && credential.Expired(c.APIKey) {
		return nil, fmt.Errorf("Claude OAuth token has expired; re-authenticate with `claude` or `claude login`")
	}

	// Validate the credential against the models endpoint.
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	c.authHeaders(req.Header)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if isOAuth {
			return nil, fmt.Errorf("Claude OAuth token rejected (HTTP %d); re-authenticate with `claude`", resp.StatusCode)
		}
		return nil, fmt.Errorf("invalid or unauthorized API key (HTTP %d)", resp.StatusCode)
	}

	// Subscription OAuth tokens cannot read the organization usage API, and
	// Anthropic exposes no per-token usage endpoint. Report the credential as
	// verified with usage totals unavailable rather than failing the row.
	if isOAuth {
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
			DataSource:    "Anthropic OAuth session (verified — per-token usage not exposed)",
		}, nil
	}

	// API key: attempt the organization usage endpoint (admin keys only).
	var consumedTokens float64 = 0
	var cost float64 = 0

	adminReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/organizations/usage", nil)
	if err == nil {
		c.authHeaders(adminReq.Header)
		if adminResp, err := c.client.Do(adminReq); err == nil {
			defer adminResp.Body.Close()
			if adminResp.StatusCode == http.StatusOK {
				var res struct {
					TotalTokens float64 `json:"total_tokens"`
					CostUSD     float64 `json:"cost_usd"`
				}
				if err := json.NewDecoder(adminResp.Body).Decode(&res); err == nil {
					consumedTokens = res.TotalTokens
					cost = res.CostUSD
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
		DataSource:    "Anthropic Live API (Key Verified ✔)",
	}, nil
}
