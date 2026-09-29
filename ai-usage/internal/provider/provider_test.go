package provider

import (
	"context"
	"testing"
	"time"

	"ai-usage/internal/core"
)

func TestAdapters_FetchUsage(t *testing.T) {
	window := core.CalculateBillingCycle(time.Now(), 1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	adapters := []core.ProviderAdapter{
		NewAntigravityAdapter(10_000_000),
		NewClaudeAdapter("", "", 5_000_000, "Claude 3.5 Sonnet"),
		NewCodexAdapter("", "", 4_000_000, "GPT-4o"),
		NewGeminiAdapter("", "", 4_000_000, "Gemini 1.5 Pro"),
		NewCopilotAdapter("", 3000, "Individual"),
		NewCursorAdapter(500, "Cursor Pro"),
		NewMuseAdapter(50_000_000, ""),
		NewWarpAdapter(100, "Warp AI"),
		NewCustomAdapter("deepseek", "DeepSeek API", "https://api.deepseek.com", "", 10_000_000, "DeepSeek-V3"),
	}

	for _, adp := range adapters {
		// Test Example mode
		usage, err := adp.FetchUsage(ctx, window, false)
		if err != nil {
			t.Errorf("adapter %s failed in example mode: %v", adp.ID(), err)
			continue
		}
		if usage.ProviderID != adp.ID() {
			t.Errorf("expected provider ID %s, got %s", adp.ID(), usage.ProviderID)
		}
		if usage.Consumed <= 0 {
			t.Errorf("adapter %s consumed value should be > 0 in example mode", adp.ID())
		}
		if usage.Quota <= 0 {
			t.Errorf("adapter %s quota value should be > 0", adp.ID())
		}
		if usage.IsLive {
			t.Errorf("adapter %s IsLive should be false in example mode", adp.ID())
		}
	}
}
