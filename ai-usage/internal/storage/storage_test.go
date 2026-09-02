package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-usage/internal/core"
)

func TestFileStore_SaveAndGetHistory(t *testing.T) {
	tempDir := t.TempDir()
	store := &FileStore{
		filePath: filepath.Join(tempDir, "history.json"),
	}

	snapshot := &core.UsageSnapshot{
		Timestamp:      time.Now().UTC(),
		TotalProviders: 2,
		ActiveOkCount:  2,
		TotalTokens:    500000,
		TotalCostUSD:   12.50,
		Providers: []core.ProviderUsage{
			{
				ProviderID:  "claude",
				DisplayName: "Anthropic Claude",
				Consumed:    500000,
				Quota:       5000000,
				Status:      "ok",
			},
		},
	}

	ctx := context.Background()
	if err := store.SaveSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	history, err := store.GetHistory(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 record in history, got %d", len(history))
	}
	if history[0].TotalTokens != 500000 {
		t.Errorf("expected 500000 tokens, got %f", history[0].TotalTokens)
	}

	// Verify file exists
	if _, err := os.Stat(store.filePath); os.IsNotExist(err) {
		t.Errorf("history file was not created")
	}
}
