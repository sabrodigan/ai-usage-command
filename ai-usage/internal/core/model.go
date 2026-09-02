package core

import (
	"fmt"
	"time"
)

// MetricUnit represents how consumption is quantified.
type MetricUnit string

const (
	UnitTokens   MetricUnit = "tokens"
	UnitRequests MetricUnit = "requests"
	UnitUSD      MetricUnit = "USD"
	UnitCredits  MetricUnit = "credits"
)

// ProviderUsage holds the normalized usage data for a single provider.
type ProviderUsage struct {
	ProviderID    string     `json:"provider_id" bson:"provider_id"`
	DisplayName   string     `json:"display_name" bson:"display_name"`
	ModelOrTier   string     `json:"model_or_tier" bson:"model_or_tier"`
	Unit          MetricUnit `json:"unit" bson:"unit"`
	Consumed      float64    `json:"consumed" bson:"consumed"`
	Quota         float64    `json:"quota" bson:"quota"`
	Remaining     float64    `json:"remaining" bson:"remaining"`
	PercentUsed   float64    `json:"percent_used" bson:"percent_used"`
	EstimatedCost float64    `json:"estimated_cost_usd" bson:"estimated_cost_usd"`
	BillingStart  time.Time  `json:"billing_start" bson:"billing_start"`
	BillingEnd    time.Time  `json:"billing_end" bson:"billing_end"`
	LastUpdated   time.Time  `json:"last_updated" bson:"last_updated"`
	Status        string     `json:"status" bson:"status"` // "ok", "degraded", "error", "stale"
	ErrorMessage  string     `json:"error_message,omitempty" bson:"error_message,omitempty"`
	IsStale       bool       `json:"is_stale" bson:"is_stale"`
	DaysInactive  int        `json:"days_inactive,omitempty" bson:"days_inactive,omitempty"`
	IsLive        bool       `json:"is_live" bson:"is_live"`
	DataSource    string     `json:"data_source,omitempty" bson:"data_source,omitempty"`
}

// FormatConsumed returns a readable string formatted for CLI display.
func (p *ProviderUsage) FormatConsumed() string {
	if p.Unit == UnitTokens {
		return FormatNumber(p.Consumed) + " " + string(p.Unit)
	}
	if p.Unit == UnitUSD {
		return fmt.Sprintf("$%.2f", p.Consumed)
	}
	return fmt.Sprintf("%.0f %s", p.Consumed, p.Unit)
}

// FormatQuota returns a readable quota string.
func (p *ProviderUsage) FormatQuota() string {
	if p.Quota <= 0 {
		return "Unlimited"
	}
	if p.Unit == UnitUSD {
		return fmt.Sprintf("$%.2f", p.Quota)
	}
	return FormatNumber(p.Quota)
}

// FormatRemaining returns a readable remaining string.
func (p *ProviderUsage) FormatRemaining() string {
	if p.Quota <= 0 {
		return "N/A"
	}
	if p.Unit == UnitUSD {
		return fmt.Sprintf("$%.2f", p.Remaining)
	}
	return FormatNumber(p.Remaining)
}

// BillingWindow describes the active billing cycle date boundaries.
type BillingWindow struct {
	Start time.Time `json:"start" bson:"start"`
	End   time.Time `json:"end" bson:"end"`
}

// UsageSnapshot encapsulates an entire query run across all configured providers.
type UsageSnapshot struct {
	Timestamp      time.Time       `json:"timestamp" bson:"timestamp"`
	BillingCycle   BillingWindow   `json:"billing_cycle" bson:"billing_cycle"`
	TotalProviders int             `json:"total_providers" bson:"total_providers"`
	ActiveOkCount  int             `json:"active_ok_count" bson:"active_ok_count"`
	TotalTokens    float64         `json:"total_tokens_consumed" bson:"total_tokens_consumed"`
	TotalCostUSD   float64         `json:"total_cost_usd" bson:"total_cost_usd"`
	StaleCount     int             `json:"stale_count" bson:"stale_count"`
	IsLive         bool            `json:"is_live" bson:"is_live"`
	SnapshotMode   string          `json:"snapshot_mode" bson:"snapshot_mode"` // "LIVE" or "EXAMPLE"
	Providers      []ProviderUsage `json:"providers" bson:"providers"`
}

// FormatNumber outputs large numbers in human-readable notation (e.g. 1.25M, 450K).
func FormatNumber(n float64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", n/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.2fM", n/1_000_000)
	}
	if n >= 10_000 {
		return fmt.Sprintf("%.1fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}
