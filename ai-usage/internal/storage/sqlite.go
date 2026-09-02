package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"ai-usage/internal/config"
	"ai-usage/internal/core"
)

type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or initializes the embedded SQLite database at ~/.config/ai-usage/usage.db
func NewSQLiteStore() (*SQLiteStore, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "usage.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		_ = db.Close()
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrating schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS usage_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		total_providers INTEGER NOT NULL,
		active_ok_count INTEGER NOT NULL,
		total_tokens REAL NOT NULL,
		total_cost_usd REAL NOT NULL
	);

	CREATE TABLE IF NOT EXISTS provider_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		snapshot_id INTEGER NOT NULL,
		timestamp TEXT NOT NULL,
		provider_id TEXT NOT NULL,
		display_name TEXT NOT NULL,
		model_tier TEXT NOT NULL,
		unit TEXT NOT NULL,
		consumed REAL NOT NULL,
		quota REAL NOT NULL,
		remaining REAL NOT NULL,
		percent_used REAL NOT NULL,
		cost_usd REAL NOT NULL,
		cycle_start TEXT NOT NULL,
		cycle_end TEXT NOT NULL,
		status TEXT NOT NULL,
		FOREIGN KEY(snapshot_id) REFERENCES usage_snapshots(id)
	);

	CREATE INDEX IF NOT EXISTS idx_provider_records_provider ON provider_records(provider_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_snapshots_timestamp ON usage_snapshots(timestamp);

	CREATE TABLE IF NOT EXISTS ai_recommendations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		provider_id TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		potential_savings_usd REAL NOT NULL,
		urgency TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveSnapshot(ctx context.Context, snapshot *core.UsageSnapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tsFormatted := snapshot.Timestamp.UTC().Format(time.RFC3339)

	res, err := tx.ExecContext(ctx, `
		INSERT INTO usage_snapshots (timestamp, total_providers, active_ok_count, total_tokens, total_cost_usd)
		VALUES (?, ?, ?, ?, ?)
	`, tsFormatted, snapshot.TotalProviders, snapshot.ActiveOkCount, snapshot.TotalTokens, snapshot.TotalCostUSD)
	if err != nil {
		return err
	}

	snapshotID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO provider_records (
			snapshot_id, timestamp, provider_id, display_name, model_tier, unit,
			consumed, quota, remaining, percent_used, cost_usd, cycle_start, cycle_end, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range snapshot.Providers {
		_, err = stmt.ExecContext(ctx,
			snapshotID,
			tsFormatted,
			p.ProviderID,
			p.DisplayName,
			p.ModelOrTier,
			string(p.Unit),
			p.Consumed,
			p.Quota,
			p.Remaining,
			p.PercentUsed,
			p.EstimatedCost,
			p.BillingStart.UTC().Format(time.RFC3339),
			p.BillingEnd.UTC().Format(time.RFC3339),
			p.Status,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

type ProviderDailySummary struct {
	Date        string
	ProviderID  string
	DisplayName string
	Consumed    float64
	CostUSD     float64
	PercentUsed float64
}

func (s *SQLiteStore) GetDailyTrends(ctx context.Context, days int) ([]ProviderDailySummary, error) {
	if days <= 0 {
		days = 30
	}

	query := `
		SELECT 
			substr(timestamp, 1, 10) as day,
			provider_id,
			display_name,
			MAX(consumed) as max_consumed,
			MAX(cost_usd) as max_cost,
			MAX(percent_used) as max_pct
		FROM provider_records
		WHERE status = 'ok'
		GROUP BY day, provider_id
		ORDER BY day DESC, max_pct DESC
		LIMIT 50
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProviderDailySummary
	for rows.Next() {
		var s ProviderDailySummary
		if err := rows.Scan(&s.Date, &s.ProviderID, &s.DisplayName, &s.Consumed, &s.CostUSD, &s.PercentUsed); err == nil {
			results = append(results, s)
		}
	}
	return results, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
