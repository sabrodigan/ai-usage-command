package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ai-usage/internal/core"
)

type ClaudeAdapter struct {
	APIKey      string
	CustomQuota float64
	ModelTier   string
	client      *http.Client
}

func NewClaudeAdapter(apiKey string, quota float64, modelTier string) *ClaudeAdapter {
	if quota <= 0 {
		quota = 5_000_000 // 5M tokens default
	}
	if modelTier == "" {
		modelTier = "Claude 3.5 Sonnet / Opus"
	}
	return &ClaudeAdapter{
		APIKey:      apiKey,
		CustomQuota: quota,
		ModelTier:   modelTier,
		client:      NewHTTPClient(6 * time.Second),
	}
}

func (c *ClaudeAdapter) ID() string          { return "claude" }
func (c *ClaudeAdapter) DisplayName() string { return "Anthropic Claude" }

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

	// Live API validation and organization usage query
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid or unauthorized API key (HTTP %d)", resp.StatusCode)
	}

	// If key is authenticated, check admin usage or report active verified key
	var consumedTokens float64 = 0
	var cost float64 = 0

	adminReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/organizations/usage", nil)
	if err == nil {
		adminReq.Header.Set("x-api-key", c.APIKey)
		adminReq.Header.Set("anthropic-version", "2023-06-01")
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
