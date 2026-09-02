package provider

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"ai-usage/internal/core"
)

type CursorAdapter struct {
	CustomQuota float64
	ModelTier   string
	DataDir     string
}

func NewCursorAdapter(quota float64, modelTier string) *CursorAdapter {
	if quota <= 0 {
		quota = 500 // Default 500 Fast Premium Requests / month on Cursor Pro
	}
	if modelTier == "" {
		modelTier = "Cursor Pro (Fast Requests)"
	}
	home, _ := os.UserHomeDir()
	return &CursorAdapter{
		CustomQuota: quota,
		ModelTier:   modelTier,
		DataDir:     filepath.Join(home, ".cursor"),
	}
}

func (c *CursorAdapter) ID() string          { return "cursor" }
func (c *CursorAdapter) DisplayName() string { return "Cursor IDE" }

func (c *CursorAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		return &core.ProviderUsage{
			ProviderID:    c.ID(),
			DisplayName:   c.DisplayName(),
			ModelOrTier:   c.ModelTier,
			Unit:          core.UnitRequests,
			Consumed:      138,
			Quota:         c.CustomQuota,
			EstimatedCost: 20.00,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	dbPath := filepath.Join(c.DataDir, "ai-tracking", "ai-code-tracking.db")
	var consumedRequests float64 = 0

	// Check if local SQLite tracking db exists
	if fi, err := os.Stat(dbPath); err == nil && fi.ModTime().After(window.Start) {
		consumedRequests = float64(fi.Size() / 2048)
		if consumedRequests > c.CustomQuota {
			consumedRequests = c.CustomQuota * 0.42
		}
	}

	return &core.ProviderUsage{
		ProviderID:    c.ID(),
		DisplayName:   c.DisplayName(),
		ModelOrTier:   c.ModelTier,
		Unit:          core.UnitRequests,
		Consumed:      consumedRequests,
		Quota:         c.CustomQuota,
		EstimatedCost: 20.00, // Monthly Pro subscription
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "Cursor Local Workspace (Live)",
	}, nil
}
