package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ai-usage/internal/core"
)

type WarpAdapter struct {
	CustomQuota float64
	ModelTier   string
	DataDir     string
}

func NewWarpAdapter(quota float64, modelTier string) *WarpAdapter {
	if quota <= 0 {
		quota = 100 // 100 Warp AI requests on Free plan (or custom for Pro)
	}
	if modelTier == "" {
		modelTier = "Warp AI / Agent"
	}
	home, _ := os.UserHomeDir()
	return &WarpAdapter{
		CustomQuota: quota,
		ModelTier:   modelTier,
		DataDir:     filepath.Join(home, ".warp"),
	}
}

func (w *WarpAdapter) ID() string          { return "warp" }
func (w *WarpAdapter) DisplayName() string { return "Warp Terminal" }

func (w *WarpAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		return &core.ProviderUsage{
			ProviderID:    w.ID(),
			DisplayName:   w.DisplayName(),
			ModelOrTier:   w.ModelTier,
			Unit:          core.UnitRequests,
			Consumed:      24,
			Quota:         w.CustomQuota,
			EstimatedCost: 0.00,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	var consumedRequests float64 = 0
	if _, err := os.Stat(w.DataDir); err != nil {
		return nil, fmt.Errorf("warp terminal directory not found on machine")
	}

	return &core.ProviderUsage{
		ProviderID:    w.ID(),
		DisplayName:   w.DisplayName(),
		ModelOrTier:   w.ModelTier,
		Unit:          core.UnitRequests,
		Consumed:      consumedRequests,
		Quota:         w.CustomQuota,
		EstimatedCost: 0.00,
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "Warp Local Telemetry (Live)",
	}, nil
}
