package provider

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"ai-usage/internal/core"
)

type CursorAdapter struct {
	CustomQuota float64
	ModelTier   string
	DataDir     string // ~/.cursor
	StateDB     string // <UserConfigDir>/Cursor/User/globalStorage/state.vscdb
}

func NewCursorAdapter(quota float64, modelTier string) *CursorAdapter {
	if quota <= 0 {
		quota = 500 // Default 500 Fast Premium Requests / month on Cursor Pro
	}
	if modelTier == "" {
		modelTier = "Cursor Pro (Fast Requests)"
	}
	home, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	return &CursorAdapter{
		CustomQuota: quota,
		ModelTier:   modelTier,
		DataDir:     filepath.Join(home, ".cursor"),
		StateDB:     filepath.Join(configDir, "Cursor", "User", "globalStorage", "state.vscdb"),
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

	// Cursor keeps token counts server-side, so the best local signal is the
	// number of prompts sent, attributed to the model that answered them.
	counts, last, source, err := c.requestsFromChats(ctx, window)
	if err != nil || len(counts) == 0 {
		if tracked, trackedLast, trackErr := c.requestsFromTracking(ctx, window); trackErr == nil && len(tracked) > 0 {
			counts, last, source = tracked, trackedLast, "Cursor AI code tracking (Live)"
		}
	}
	if source == "" {
		source = "Cursor local chat history (Live)"
	}

	var consumed float64
	models := make([]core.ModelUsage, 0, len(counts))
	for name, n := range counts {
		consumed += float64(n)
		models = append(models, core.ModelUsage{Model: name, Unit: core.UnitRequests, Consumed: float64(n), Requests: n})
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Requests != models[j].Requests {
			return models[i].Requests > models[j].Requests
		}
		return models[i].Model < models[j].Model
	})
	tier := c.ModelTier
	if len(models) > 0 {
		tier = core.SummarizeModels(models)
	}

	return &core.ProviderUsage{
		ProviderID:    c.ID(),
		DisplayName:   c.DisplayName(),
		ModelOrTier:   tier,
		Unit:          core.UnitRequests,
		Consumed:      consumed,
		Quota:         c.CustomQuota,
		EstimatedCost: 20.00, // Monthly Pro subscription
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    source,
		Models:        models,
		LastActivity:  lastActivity(last),
	}, nil
}

// requestsFromChats counts user prompts (bubble type 1) in the IDE's chat store,
// keyed by the model each prompt was sent to.
func (c *CursorAdapter) requestsFromChats(ctx context.Context, window core.BillingWindow) (map[string]int, time.Time, string, error) {
	var last time.Time
	db, err := openReadOnly(c.StateDB)
	if err != nil {
		return nil, last, "", err
	}
	defer db.Close()

	// Composer-level model is the fallback for older prompts without modelInfo.
	composerModel := map[string]string{}
	if rows, err := db.QueryContext(ctx, `
		SELECT substr(key, 14), json_extract(CAST(value AS TEXT), '$.modelConfig.modelName')
		FROM cursorDiskKV
		WHERE key LIKE 'composerData:%' AND json_valid(CAST(value AS TEXT))`); err == nil {
		for rows.Next() {
			var id string
			var model sql.NullString
			if rows.Scan(&id, &model) == nil && model.Valid {
				composerModel[id] = model.String
			}
		}
		rows.Close()
	}

	rows, err := db.QueryContext(ctx, `
		SELECT key,
		       json_extract(CAST(value AS TEXT), '$.createdAt'),
		       json_extract(CAST(value AS TEXT), '$.modelInfo.modelName')
		FROM cursorDiskKV
		WHERE key LIKE 'bubbleId:%' AND json_valid(CAST(value AS TEXT))
		  AND json_extract(CAST(value AS TEXT), '$.type') = 1`)
	if err != nil {
		return nil, last, "", err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var key string
		var created any
		var model sql.NullString
		if rows.Scan(&key, &created, &model) != nil {
			continue
		}
		at, ok := parseCursorTime(created)
		if !ok || at.Before(window.Start) || !at.Before(window.End) {
			continue
		}
		if at.After(last) {
			last = at
		}
		name := model.String
		if name == "" {
			// key is bubbleId:<composerId>:<bubbleId>
			if parts := strings.SplitN(key, ":", 3); len(parts) == 3 {
				name = composerModel[parts[1]]
			}
		}
		counts[cursorModelName(name)]++
	}
	return counts, last, "Cursor local chat history (Live)", rows.Err()
}

// requestsFromTracking falls back to ~/.cursor/ai-tracking, which records the
// model behind each AI-authored code change.
func (c *CursorAdapter) requestsFromTracking(ctx context.Context, window core.BillingWindow) (map[string]int, time.Time, error) {
	var last time.Time
	db, err := openReadOnly(filepath.Join(c.DataDir, "ai-tracking", "ai-code-tracking.db"))
	if err != nil {
		return nil, last, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(model, ''), COUNT(DISTINCT COALESCE(requestId, hash)), MAX(createdAt)
		FROM ai_code_hashes
		WHERE createdAt >= ? AND createdAt < ?
		GROUP BY 1`, window.Start.UnixMilli(), window.End.UnixMilli())
	if err != nil {
		return nil, last, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var model string
		var n int
		var latest int64
		if rows.Scan(&model, &n, &latest) == nil {
			counts[cursorModelName(model)] += n
			if at := time.UnixMilli(latest); at.After(last) {
				last = at
			}
		}
	}
	return counts, last, rows.Err()
}

func openReadOnly(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed // Windows drive paths: file:///C:/...
	}
	dsn := (&url.URL{Scheme: "file", Path: slashed, RawQuery: "mode=ro&_pragma=busy_timeout(2000)"}).String()
	return sql.Open("sqlite", dsn)
}

// lastActivity returns nil for the zero time so the field is omitted from JSON.
func lastActivity(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}

// cursorModelName maps Cursor's internal "default" (the Auto router) to its UI label.
func cursorModelName(name string) string {
	switch name {
	case "", "default":
		return "auto"
	}
	return name
}

// parseCursorTime accepts ISO-8601 strings (current builds) and epoch millis (older builds).
func parseCursorTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case string:
		at, err := time.Parse(time.RFC3339Nano, t)
		return at, err == nil
	case int64:
		return time.UnixMilli(t), true
	case float64:
		return time.UnixMilli(int64(t)), true
	}
	return time.Time{}, false
}
