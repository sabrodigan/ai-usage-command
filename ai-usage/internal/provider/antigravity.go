package provider

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"ai-usage/internal/core"
)

type AntigravityAdapter struct {
	CustomQuota float64
	DataDir     string
}

func NewAntigravityAdapter(quota float64) *AntigravityAdapter {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if quota <= 0 {
		quota = 10_000_000 // 10M tokens default tier
	}
	return &AntigravityAdapter{
		CustomQuota: quota,
		DataDir:     dataDir,
	}
}

func (a *AntigravityAdapter) ID() string          { return "antigravity" }
func (a *AntigravityAdapter) DisplayName() string { return "Antigravity Agent" }

func (a *AntigravityAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	if !liveMode {
		return &core.ProviderUsage{
			ProviderID:    a.ID(),
			DisplayName:   a.DisplayName(),
			ModelOrTier:   "Agent Pro Runtime",
			Unit:          core.UnitTokens,
			Consumed:      1_024_500,
			Quota:         a.CustomQuota,
			EstimatedCost: 0.00,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			IsLive:        false,
			DataSource:    "Simulated Example",
		}, nil
	}

	// In live mode: parse actual local Antigravity transcript logs strictly within current billing window
	var estimatedTokens float64 = 0
	brainDir := filepath.Join(a.DataDir, "brain")

	if info, err := os.Stat(brainDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(brainDir, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err != nil {
				return nil
			}
			if !d.IsDir() && filepath.Ext(path) == ".jsonl" {
				if fi, err := d.Info(); err == nil {
					// Only count logs created or modified within the billing cycle
					if fi.ModTime().After(window.Start) && fi.ModTime().Before(window.End) {
						estimatedTokens += float64(fi.Size()) * 0.30
					}
				}
			}
			return nil
		})
	}

	return &core.ProviderUsage{
		ProviderID:    a.ID(),
		DisplayName:   a.DisplayName(),
		ModelOrTier:   "Agent Pro Runtime",
		Unit:          core.UnitTokens,
		Consumed:      estimatedTokens,
		Quota:         a.CustomQuota,
		EstimatedCost: 0.00, // Included in subscription tier
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "Local Session Logs (Live)",
	}, nil
}
