package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ai-usage/internal/config"
	"ai-usage/internal/core"
)

// Store defines persistence methods for usage historical records.
type Store interface {
	SaveSnapshot(ctx context.Context, snapshot *core.UsageSnapshot) error
	GetHistory(ctx context.Context, limit int) ([]core.UsageSnapshot, error)
	Close() error
}

// FileStore provides lightweight local JSON persistence without external database requirements.
type FileStore struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileStore creates a file-backed historical storage.
func NewFileStore() (*FileStore, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{
		filePath: filepath.Join(dir, "history.json"),
	}, nil
}

func (f *FileStore) SaveSnapshot(ctx context.Context, snapshot *core.UsageSnapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var history []core.UsageSnapshot
	data, err := os.ReadFile(f.filePath)
	if err == nil {
		_ = json.Unmarshal(data, &history)
	}

	// Keep up to 100 recent snapshots
	history = append([]core.UsageSnapshot{*snapshot}, history...)
	if len(history) > 100 {
		history = history[:100]
	}

	bytes, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(f.filePath, bytes, 0600)
}

func (f *FileStore) GetHistory(ctx context.Context, limit int) ([]core.UsageSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return []core.UsageSnapshot{}, nil
	} else if err != nil {
		return nil, err
	}

	var history []core.UsageSnapshot
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}
	return history, nil
}

func (f *FileStore) Close() error {
	return nil
}

// SnapshotDoc is the schema used for MongoDB / Remote Document stores.
type SnapshotDoc struct {
	ID        string              `json:"id" bson:"_id,omitempty"`
	UserID    string              `json:"user_id" bson:"user_id"`
	Timestamp time.Time           `json:"timestamp" bson:"timestamp"`
	Cycle     core.BillingWindow  `json:"billing_cycle" bson:"billing_cycle"`
	Providers []core.ProviderUsage `json:"providers" bson:"providers"`
	TotalCost float64             `json:"total_cost_usd" bson:"total_cost_usd"`
}
