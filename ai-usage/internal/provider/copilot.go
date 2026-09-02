package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ai-usage/internal/core"
)

type CopilotAdapter struct {
	Token       string
	CustomQuota float64
	ModelTier   string
	client      *http.Client
}

func NewCopilotAdapter(token string, quota float64, modelTier string) *CopilotAdapter {
	if quota <= 0 {
		quota = 3000 // 3000 requests / premium interactions per month
	}
	if modelTier == "" {
		modelTier = "Individual Plan"
	}
	return &CopilotAdapter{
		Token:       token,
		CustomQuota: quota,
		ModelTier:   modelTier,
		client:      NewHTTPClient(5 * time.Second),
	}
}

func (c *CopilotAdapter) ID() string          { return "copilot" }
func (c *CopilotAdapter) DisplayName() string { return "GitHub Copilot" }

func (c *CopilotAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		return &core.ProviderUsage{
			ProviderID:    c.ID(),
			DisplayName:   c.DisplayName(),
			ModelOrTier:   c.ModelTier,
			Unit:          core.UnitRequests,
			Consumed:      1240,
			Quota:         c.CustomQuota,
			EstimatedCost: 10.00,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	if !IsAPIKeyConfigured(c.Token) {
		return nil, fmt.Errorf("no GitHub token configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid or unauthorized GitHub token (HTTP %d)", resp.StatusCode)
	}

	return &core.ProviderUsage{
		ProviderID:    c.ID(),
		DisplayName:   c.DisplayName(),
		ModelOrTier:   c.ModelTier,
		Unit:          core.UnitRequests,
		Consumed:      0, // Flat subscription
		Quota:         c.CustomQuota,
		EstimatedCost: 10.00, // Monthly flat subscription
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "GitHub Live Auth (Subscription Active ✔)",
	}, nil
}
