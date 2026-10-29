package core

import (
	"context"
	"sort"
	"sync"
	"time"
)

// ProviderAdapter defines the interface that all service providers must implement.
type ProviderAdapter interface {
	ID() string
	DisplayName() string
	FetchUsage(ctx context.Context, window BillingWindow, liveMode bool) (*ProviderUsage, error)
}

// ProviderTask pairs an adapter with its specific billing cycle window.
type ProviderTask struct {
	Adapter ProviderAdapter
	Window  BillingWindow
}

// FetchAndAggregate queries all adapters concurrently using their specific billing windows.
func FetchAndAggregate(ctx context.Context, tasks []ProviderTask, defaultWindow BillingWindow, liveMode bool) *UsageSnapshot {
	modeStr := "EXAMPLE"
	if liveMode {
		modeStr = "LIVE"
	}
	snapshot := &UsageSnapshot{
		Timestamp:      time.Now().UTC(),
		BillingCycle:   defaultWindow,
		TotalProviders: len(tasks),
		IsLive:         liveMode,
		SnapshotMode:   modeStr,
		Providers:      make([]ProviderUsage, len(tasks)),
	}

	if len(tasks) == 0 {
		return snapshot
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t ProviderTask) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					snapshot.Providers[idx] = ProviderUsage{
						ProviderID:   t.Adapter.ID(),
						DisplayName:  t.Adapter.DisplayName(),
						Status:       "error",
						ErrorMessage: "adapter panicked during execution",
						LastUpdated:  time.Now().UTC(),
						IsLive:       liveMode,
					}
					mu.Unlock()
				}
			}()

			win := t.Window
			if win.Start.IsZero() || win.End.IsZero() {
				win = defaultWindow
			}

			res, err := t.Adapter.FetchUsage(ctx, win, liveMode)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				snapshot.Providers[idx] = ProviderUsage{
					ProviderID:   t.Adapter.ID(),
					DisplayName:  t.Adapter.DisplayName(),
					Status:       "error",
					ErrorMessage: err.Error(),
					BillingStart: win.Start,
					BillingEnd:   win.End,
					LastUpdated:  time.Now().UTC(),
				}
				return
			}

			// Ensure billing window is recorded on the result
			res.BillingStart = win.Start
			res.BillingEnd = win.End

			// Normalize computed metrics
			if res.Quota > 0 {
				res.Remaining = res.Quota - res.Consumed
				if res.Remaining < 0 {
					res.Remaining = 0
				}
				res.PercentUsed = (res.Consumed / res.Quota) * 100.0
			} else {
				res.Remaining = 0
				res.PercentUsed = 0
			}

			if res.Status == "" {
				res.Status = "ok"
			}
			if res.LastUpdated.IsZero() {
				res.LastUpdated = time.Now().UTC()
			}

			snapshot.Providers[idx] = *res
		}(i, task)
	}

	wg.Wait()

	// Compute totals across successful providers
	var totalTokens float64
	var totalCost float64
	var activeCount int

	for _, p := range snapshot.Providers {
		if p.Status == "ok" {
			activeCount++
			if p.Unit == UnitTokens {
				totalTokens += p.Consumed
			}
			totalCost += p.EstimatedCost
		}
	}

	snapshot.ActiveOkCount = activeCount
	snapshot.TotalTokens = totalTokens
	snapshot.TotalCostUSD = totalCost

	// Sort and rank
	sort.SliceStable(snapshot.Providers, func(i, j int) bool {
		pi := snapshot.Providers[i]
		pj := snapshot.Providers[j]

		if (pi.Status == "ok") != (pj.Status == "ok") {
			return pi.Status == "ok"
		}
		if pi.PercentUsed != pj.PercentUsed {
			return pi.PercentUsed > pj.PercentUsed
		}
		return pi.Consumed > pj.Consumed
	})

	return snapshot
}
