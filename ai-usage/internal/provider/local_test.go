package provider

import (
	"context"
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"ai-usage/internal/core"
)

var testWindow = core.BillingWindow{
	Start: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	End:   time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMuseAdapter_SumsModelCallsAndPricesThem(t *testing.T) {
	dir := t.TempDir()
	inWindow := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC).UnixMicro()
	before := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC).UnixMicro()
	journal := `{"payload_type":"runtime.session","recorded_at":` + itoa(inWindow) + `,"payload":{"kind":"run","event":{"kind":"model_completed","model":"muse-spark-1.3","usage":{"cached_tokens":400000,"input_tokens":1000000,"output_tokens":100000}}}}
{"payload_type":"runtime.session","recorded_at":` + itoa(inWindow) + `,"payload":{"kind":"approval","event":{"kind":"automated_review_completed","model":{"model_id":"muse-spark-1.3"},"usage":{"cached_input_tokens":0,"input_tokens":1000,"output_tokens":10}}}}
{"payload_type":"runtime.session","recorded_at":` + itoa(inWindow) + `,"payload":{"kind":"run","event":{"kind":"goal_usage_attribution","record":{"quantity":{"input_tokens":1000000,"output_tokens":100000}}}}}
{"payload_type":"runtime.session","recorded_at":` + itoa(before) + `,"payload":{"kind":"run","event":{"kind":"model_completed","model":"muse-spark-1.3","usage":{"input_tokens":5,"output_tokens":5}}}}
{"omitted_record":true}
`
	writeFile(t, filepath.Join(dir, "sessions", "2026", "09", "29", "abc", "session.jsonl"), journal)
	writeFile(t, filepath.Join(dir, "model-catalog", "meta.json"),
		`{"rows":[{"model_id":"muse-spark-1.3","cost":{"input":"1.25","output":"4.25","cached":"0.15"}}]}`)

	m := NewMuseAdapter(0, "")
	m.DataDir, m.ConfigDir = dir, filepath.Join(dir, "cfg")
	u, err := m.FetchUsage(context.Background(), testWindow, true)
	if err != nil {
		t.Fatal(err)
	}
	if u.Consumed != 1_101_010 {
		t.Fatalf("consumed = %v, want 1101010 (goal attribution and out-of-window calls excluded)", u.Consumed)
	}
	if len(u.Models) != 1 || u.Models[0].Requests != 2 || u.ModelOrTier != "muse-spark-1.3 (2)" {
		t.Fatalf("unexpected breakdown %+v / %q", u.Models, u.ModelOrTier)
	}
	// 601000 uncached * 1.25 + 400000 cached * 0.15 + 100010 out * 4.25, per 1M
	want := (601_000*1.25 + 400_000*0.15 + 100_010*4.25) / 1_000_000
	if math.Abs(u.EstimatedCost-want) > 1e-9 {
		t.Fatalf("cost = %v, want %v", u.EstimatedCost, want)
	}
}

func TestCursorAdapter_CountsPromptsPerModel(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.vscdb")
	db, err := sql.Open("sqlite", statePath)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`,
		`INSERT INTO cursorDiskKV VALUES ('composerData:c1', '{"modelConfig":{"modelName":"grok-4.6"}}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c1:b1', '{"type":1,"createdAt":"2026-09-18T16:35:37.065Z","modelInfo":{"modelName":"grok-4.6"}}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c1:b2', '{"type":1,"createdAt":"2026-09-19T10:00:00Z"}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c1:b3', '{"type":2,"createdAt":"2026-09-19T10:00:01Z"}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c1:b4', '{"type":1,"createdAt":"2026-08-27T16:37:57Z","modelInfo":{"modelName":"grok-4.6"}}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c2:b1', '{"type":1,"createdAt":1789000000000,"modelInfo":{"modelName":"default"}}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:c2:b2', 'not json')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	c := NewCursorAdapter(500, "")
	c.DataDir, c.StateDB = filepath.Join(dir, "missing"), statePath
	u, err := c.FetchUsage(context.Background(), testWindow, true)
	if err != nil {
		t.Fatal(err)
	}
	if u.Consumed != 3 || u.ModelOrTier != "grok-4.6 (2) · auto (1)" {
		t.Fatalf("consumed=%v tier=%q models=%+v", u.Consumed, u.ModelOrTier, u.Models)
	}
}

func TestCursorAdapter_FallsBackToTrackingDB(t *testing.T) {
	dir := t.TempDir()
	trackPath := filepath.Join(dir, "ai-tracking", "ai-code-tracking.db")
	_ = os.MkdirAll(filepath.Dir(trackPath), 0o755)
	db, err := sql.Open("sqlite", trackPath)
	if err != nil {
		t.Fatal(err)
	}
	ms := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC).UnixMilli()
	for _, s := range []string{
		`CREATE TABLE ai_code_hashes (hash TEXT PRIMARY KEY, source TEXT, requestId TEXT, model TEXT, createdAt INTEGER NOT NULL)`,
		`INSERT INTO ai_code_hashes VALUES ('h1','composer','r1','grok-4.6',` + itoa(ms) + `)`,
		`INSERT INTO ai_code_hashes VALUES ('h2','composer','r1','grok-4.6',` + itoa(ms) + `)`,
		`INSERT INTO ai_code_hashes VALUES ('h3','composer','r2','grok-4.6',` + itoa(ms) + `)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	c := NewCursorAdapter(500, "")
	c.DataDir, c.StateDB = dir, filepath.Join(dir, "missing.vscdb")
	u, err := c.FetchUsage(context.Background(), testWindow, true)
	if err != nil {
		t.Fatal(err)
	}
	if u.Consumed != 2 || u.ModelOrTier != "grok-4.6 (2)" {
		t.Fatalf("consumed=%v tier=%q", u.Consumed, u.ModelOrTier)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
