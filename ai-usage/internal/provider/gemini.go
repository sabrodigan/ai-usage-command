package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ai-usage/internal/core"
)

type GeminiAdapter struct {
	APIKey      string
	CustomQuota float64
	ModelTier   string
	client      *http.Client
}

func NewGeminiAdapter(apiKey string, quota float64, modelTier string) *GeminiAdapter {
	if quota <= 0 {
		quota = 4_000_000 // 4M tokens default
	}
	if modelTier == "" {
		modelTier = "Gemini 1.5 Pro / Flash"
	}
	return &GeminiAdapter{
		APIKey:      apiKey,
		CustomQuota: quota,
		ModelTier:   modelTier,
		client:      NewHTTPClient(6 * time.Second),
	}
}

func (g *GeminiAdapter) ID() string          { return "gemini" }
func (g *GeminiAdapter) DisplayName() string { return "Google Gemini" }

func (g *GeminiAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		var consumedTokens float64 = 920_000
		cost := (consumedTokens / 1_000_000) * 3.50
		return &core.ProviderUsage{
			ProviderID:    g.ID(),
			DisplayName:   g.DisplayName(),
			ModelOrTier:   g.ModelTier,
			Unit:          core.UnitTokens,
			Consumed:      consumedTokens,
			Quota:         g.CustomQuota,
			EstimatedCost: cost,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	if !IsAPIKeyConfigured(g.APIKey) {
		return nil, fmt.Errorf("no Google Gemini API key configured")
	}

	reqURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", g.APIKey)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("invalid or unauthorized Google AI key (HTTP %d)", resp.StatusCode)
	}

	var consumedTokens float64 = 0
	var cost float64 = 0

	return &core.ProviderUsage{
		ProviderID:    g.ID(),
		DisplayName:   g.DisplayName(),
		ModelOrTier:   g.ModelTier,
		Unit:          core.UnitTokens,
		Consumed:      consumedTokens,
		Quota:         g.CustomQuota,
		EstimatedCost: cost,
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "Google Gemini Live API (Key Verified ✔)",
	}, nil
}
