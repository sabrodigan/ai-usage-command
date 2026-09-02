package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockAdapter struct {
	id          string
	displayName string
	usage       *ProviderUsage
	err         error
}

func (m *mockAdapter) ID() string          { return m.id }
func (m *mockAdapter) DisplayName() string { return m.displayName }
func (m *mockAdapter) FetchUsage(ctx context.Context, window BillingWindow, liveMode bool) (*ProviderUsage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.usage, nil
}

func TestFetchAndAggregate_RankingAndCalculations(t *testing.T) {
	window := CalculateBillingCycle(time.Now(), 1)

	tasks := []ProviderTask{
		{
			Adapter: &mockAdapter{
				id:          "p1",
				displayName: "Provider 1 (20%)",
				usage: &ProviderUsage{
					ProviderID:    "p1",
					DisplayName:   "Provider 1",
					Unit:          UnitTokens,
					Consumed:      200_000,
					Quota:         1_000_000,
					EstimatedCost: 5.0,
				},
			},
			Window: window,
		},
		{
			Adapter: &mockAdapter{
				id:          "p2",
				displayName: "Provider 2 (80%)",
				usage: &ProviderUsage{
					ProviderID:    "p2",
					DisplayName:   "Provider 2",
					Unit:          UnitTokens,
					Consumed:      800_000,
					Quota:         1_000_000,
					EstimatedCost: 20.0,
				},
			},
			Window: window,
		},
		{
			Adapter: &mockAdapter{
				id:          "p3",
				displayName: "Provider 3 (Failing)",
				err:         errors.New("unauthorized 401"),
			},
			Window: window,
		},
	}

	snapshot := FetchAndAggregate(context.Background(), tasks, window, true)

	if snapshot.TotalProviders != 3 {
		t.Fatalf("expected 3 total providers, got %d", snapshot.TotalProviders)
	}
	if snapshot.ActiveOkCount != 2 {
		t.Fatalf("expected 2 active providers, got %d", snapshot.ActiveOkCount)
	}
	if snapshot.TotalTokens != 1_000_000 {
		t.Errorf("expected 1,000,000 total tokens, got %f", snapshot.TotalTokens)
	}
	if snapshot.TotalCostUSD != 25.0 {
		t.Errorf("expected 25.0 total cost, got %f", snapshot.TotalCostUSD)
	}

	// First ranked must be Provider 2 (80% used)
	if snapshot.Providers[0].ProviderID != "p2" {
		t.Errorf("expected rank 1 to be p2, got %s", snapshot.Providers[0].ProviderID)
	}
	if snapshot.Providers[0].PercentUsed != 80.0 {
		t.Errorf("expected p2 percent 80, got %f", snapshot.Providers[0].PercentUsed)
	}

	// Second ranked must be Provider 1 (20% used)
	if snapshot.Providers[1].ProviderID != "p1" {
		t.Errorf("expected rank 2 to be p1, got %s", snapshot.Providers[1].ProviderID)
	}

	// Third ranked must be failing Provider 3
	if snapshot.Providers[2].ProviderID != "p3" {
		t.Errorf("expected rank 3 to be p3, got %s", snapshot.Providers[2].ProviderID)
	}
}
