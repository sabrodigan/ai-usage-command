package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"ai-usage/internal/core"
)

// MuseAdapter reads Meta's Muse coding-agent session journals
// (~/.local/share/muse/sessions/YYYY/MM/DD/<id>/session.jsonl) and prices them
// with the model catalog Muse caches alongside (model-catalog/*.json).
type MuseAdapter struct {
	CustomQuota float64
	ModelTier   string
	DataDir     string
	ConfigDir   string
}

func NewMuseAdapter(quota float64, modelTier string) *MuseAdapter {
	if quota <= 0 {
		quota = 50_000_000
	}
	home, _ := os.UserHomeDir()
	return &MuseAdapter{
		CustomQuota: quota,
		ModelTier:   modelTier,
		DataDir:     filepath.Join(home, ".local", "share", "muse"),
		ConfigDir:   filepath.Join(home, ".config", "muse"),
	}
}

func (m *MuseAdapter) ID() string          { return "muse" }
func (m *MuseAdapter) DisplayName() string { return "Meta Muse" }

// museTokens is the union of the two usage shapes Muse journals: main-loop
// model calls report cached_tokens, automated approval reviews report
// cached_input_tokens. input_tokens includes the cached share in both.
type museTokens struct {
	Input       float64 `json:"input_tokens"`
	Cached      float64 `json:"cached_tokens"`
	CachedInput float64 `json:"cached_input_tokens"`
	Output      float64 `json:"output_tokens"`
}

type museRecord struct {
	PayloadType string `json:"payload_type"`
	RecordedAt  int64  `json:"recorded_at"` // microseconds since epoch
	Payload     struct {
		Event struct {
			Kind  string          `json:"kind"`
			Model json.RawMessage `json:"model"` // string, or {"model_id": ...} on reviews
			Usage *museTokens     `json:"usage"`
		} `json:"event"`
	} `json:"payload"`
}

type musePrice struct{ input, cached, output float64 } // USD per 1M tokens

func (m *MuseAdapter) FetchUsage(ctx context.Context, window core.BillingWindow, liveMode bool) (*core.ProviderUsage, error) {
	tier := m.ModelTier
	if tier == "" {
		tier = m.configuredModel()
	}
	if !liveMode {
		return &core.ProviderUsage{
			ProviderID:    m.ID(),
			DisplayName:   m.DisplayName(),
			ModelOrTier:   "muse-spark-1.3",
			Unit:          core.UnitTokens,
			Consumed:      2_450_000,
			Quota:         m.CustomQuota,
			EstimatedCost: 1.20,
			BillingStart:  window.Start,
			BillingEnd:    window.End,
			LastUpdated:   time.Now().UTC(),
			Status:        "ok",
			DataSource:    "Simulated Example",
		}, nil
	}

	models, last, err := m.scanSessions(ctx, window)
	if err != nil {
		return nil, err
	}
	prices := m.loadPrices()

	var consumed, cost float64
	breakdown := make([]core.ModelUsage, 0, len(models))
	for _, u := range models {
		if p, ok := prices[u.Model]; ok {
			u.EstimatedCost = ((u.InputTokens-u.CachedTokens)*p.input + u.CachedTokens*p.cached + u.OutputTokens*p.output) / 1_000_000
		}
		consumed += u.Consumed
		cost += u.EstimatedCost
		breakdown = append(breakdown, *u)
	}
	sort.Slice(breakdown, func(i, j int) bool { return breakdown[i].Consumed > breakdown[j].Consumed })
	if len(breakdown) > 0 {
		tier = core.SummarizeModels(breakdown)
	}

	return &core.ProviderUsage{
		ProviderID:    m.ID(),
		DisplayName:   m.DisplayName(),
		ModelOrTier:   tier,
		Unit:          core.UnitTokens,
		Consumed:      consumed,
		Quota:         m.CustomQuota,
		EstimatedCost: cost,
		BillingStart:  window.Start,
		BillingEnd:    window.End,
		LastUpdated:   time.Now().UTC(),
		Status:        "ok",
		IsLive:        true,
		DataSource:    "Muse session journals (Live)",
		Models:        breakdown,
		LastActivity:  lastActivity(last),
	}, nil
}

// scanSessions sums per-model token usage for model calls recorded inside window.
func (m *MuseAdapter) scanSessions(ctx context.Context, window core.BillingWindow) (map[string]*core.ModelUsage, time.Time, error) {
	models := map[string]*core.ModelUsage{}
	var last time.Time
	root := filepath.Join(m.DataDir, "sessions")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil || d.IsDir() || d.Name() != "session.jsonl" {
			return nil
		}
		if fi, err := d.Info(); err != nil || fi.ModTime().Before(window.Start) {
			return nil
		}
		return m.scanFile(path, window, models, &last)
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, last, err
	}
	return models, last, nil
}

func (m *MuseAdapter) scanFile(path string, window core.BillingWindow, models map[string]*core.ModelUsage, last *time.Time) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		line := sc.Bytes()
		// Cheaply skip the bulk of journal lines before JSON decoding.
		if !bytes.Contains(line, []byte(`"output_tokens"`)) {
			continue
		}
		var rec museRecord
		if json.Unmarshal(line, &rec) != nil || rec.PayloadType != "runtime.session" {
			continue
		}
		ev := rec.Payload.Event
		if ev.Usage == nil || (ev.Kind != "model_completed" && ev.Kind != "automated_review_completed") {
			continue
		}
		at := time.UnixMicro(rec.RecordedAt)
		if at.Before(window.Start) || !at.Before(window.End) {
			continue
		}
		if at.After(*last) {
			*last = at
		}
		name := museModelName(ev.Model)
		u, ok := models[name]
		if !ok {
			u = &core.ModelUsage{Model: name, Unit: core.UnitTokens}
			models[name] = u
		}
		cached := ev.Usage.Cached + ev.Usage.CachedInput
		u.Requests++
		u.InputTokens += ev.Usage.Input
		u.CachedTokens += cached
		u.OutputTokens += ev.Usage.Output
		u.Consumed += ev.Usage.Input + ev.Usage.Output
	}
	return sc.Err()
}

func museModelName(raw json.RawMessage) string {
	var name string
	if json.Unmarshal(raw, &name) == nil && name != "" {
		return name
	}
	var obj struct {
		ModelID string `json:"model_id"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.ModelID != "" {
		return obj.ModelID
	}
	return "unknown"
}

// loadPrices reads per-model USD/1M-token prices from Muse's cached provider catalog.
func (m *MuseAdapter) loadPrices() map[string]musePrice {
	prices := map[string]musePrice{}
	files, _ := filepath.Glob(filepath.Join(m.DataDir, "model-catalog", "*.json"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var catalog struct {
			Rows []struct {
				ModelID string `json:"model_id"`
				Cost    struct {
					Input  string `json:"input"`
					Output string `json:"output"`
					Cached string `json:"cached"`
				} `json:"cost"`
			} `json:"rows"`
		}
		if json.Unmarshal(data, &catalog) != nil {
			continue
		}
		for _, r := range catalog.Rows {
			in, _ := strconv.ParseFloat(r.Cost.Input, 64)
			out, _ := strconv.ParseFloat(r.Cost.Output, 64)
			cached, err := strconv.ParseFloat(r.Cost.Cached, 64)
			if err != nil {
				cached = in
			}
			prices[r.ModelID] = musePrice{input: in, cached: cached, output: out}
		}
	}
	return prices
}

// configuredModel returns the default model from ~/.config/muse/settings.json.
func (m *MuseAdapter) configuredModel() string {
	var settings struct {
		Model string `json:"model"`
	}
	if data, err := os.ReadFile(filepath.Join(m.ConfigDir, "settings.json")); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings.Model == "" {
		return "Muse Spark"
	}
	return settings.Model
}
