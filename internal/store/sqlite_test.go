package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// openTemp opens a store backed by a real file (not :memory:), as the acceptance
// criterion requires: an in-memory DB would hide migration and file-locking issues.
func openTemp(t *testing.T) *SQLite {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenIsIdempotent(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s1, err := OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = s1.Close()
	// Opening the same file again must run migrations as a no-op, not fail.
	s2, err := OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("second open must be a no-op, got: %v", err)
	}
	_ = s2.Close()
}

func TestSignalRoundTripAndDedup(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	sig := core.Signal{
		ID: "01H", Source: "hackernews", URL: "https://x", Author: "a", Title: "t",
		Content: "c", Lang: "en", PublishedAt: time.Now().UTC().Truncate(time.Second),
		CollectedAt: time.Now().UTC().Truncate(time.Second), ContentHash: "hash1",
	}
	inserted, err := s.SaveSignal(ctx, sig)
	if err != nil || !inserted {
		t.Fatalf("first save: inserted=%v err=%v", inserted, err)
	}
	// Same content_hash, different id: dedup must make this a no-op.
	dup := sig
	dup.ID = "01H-other"
	inserted, err = s.SaveSignal(ctx, dup)
	if err != nil {
		t.Fatalf("dup save: %v", err)
	}
	if inserted {
		t.Error("a duplicate content_hash must not insert a second row")
	}

	got, err := s.GetSignal(ctx, "01H")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "t" || got.Source != "hackernews" || !got.PublishedAt.Equal(sig.PublishedAt) {
		t.Errorf("round trip mismatch: %+v", got)
	}

	if _, err := s.GetSignal(ctx, "missing"); err != ErrNotFound {
		t.Errorf("missing signal should be ErrNotFound, got %v", err)
	}
}

func TestScoreRoundTrip(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	if _, err := s.SaveSignal(ctx, core.Signal{ID: "sig1", Source: "hn", ContentHash: "h"}); err != nil {
		t.Fatal(err)
	}
	sc := core.Score{
		SignalID: "sig1", Value: 82, IntentType: core.IntentHiring,
		Reasons: []string{"hiring an LLM engineer", "EU based"}, ICPMatch: []string{"region", "role"},
		DraftReply: "draft", Model: "qwen2.5:14b", CostUSD: 0.002, LatencyMS: 450,
		ScoredAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := s.SaveScore(ctx, sc); err != nil {
		t.Fatalf("save score: %v", err)
	}
	got, err := s.GetScore(ctx, "sig1")
	if err != nil {
		t.Fatalf("get score: %v", err)
	}
	if got.Value != 82 || len(got.Reasons) != 2 || got.Reasons[0] != "hiring an LLM engineer" {
		t.Errorf("score round trip mismatch: %+v", got)
	}
	if len(got.ICPMatch) != 2 {
		t.Errorf("icp_match not preserved: %+v", got.ICPMatch)
	}
}

func TestScoreFilters(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	for _, tc := range []struct {
		id, source string
		score      int
		intent     string
	}{
		{"a", "hackernews", 90, core.IntentHiring},
		{"b", "reddit", 40, core.IntentResearch},
		{"c", "hackernews", 75, core.IntentPain},
	} {
		if _, err := s.SaveSignal(ctx, core.Signal{ID: tc.id, Source: tc.source, ContentHash: tc.id}); err != nil {
			t.Fatal(err)
		}
		if err := s.SaveScore(ctx, core.Score{SignalID: tc.id, Value: tc.score, IntentType: tc.intent, Reasons: []string{"r"}}); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := s.ListSignals(ctx, SignalFilter{ScoreGTE: 70})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("score_gte=70 should return 2 signals, got %d", len(got))
	}
	got, _, err = s.ListSignals(ctx, SignalFilter{Source: "reddit"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "b" {
		t.Errorf("source filter wrong: %+v", got)
	}
}

func TestListSignalsPagination(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	for _, id := range []string{"01", "02", "03", "04", "05"} {
		if _, err := s.SaveSignal(ctx, core.Signal{ID: id, Source: "hn", ContentHash: id}); err != nil {
			t.Fatal(err)
		}
	}
	page1, next, err := s.ListSignals(ctx, SignalFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || next == "" {
		t.Fatalf("page1=%d next=%q, want 2 and a cursor", len(page1), next)
	}
	page2, _, err := s.ListSignals(ctx, SignalFilter{Limit: 2, Cursor: next})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || page2[0].ID >= page1[len(page1)-1].ID {
		t.Errorf("page2 should continue after page1: %+v then %+v", page1, page2)
	}
}

func TestLeadRoundTripWithSignals(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	for _, id := range []string{"s1", "s2"} {
		if _, err := s.SaveSignal(ctx, core.Signal{ID: id, Source: "hn", ContentHash: id}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Truncate(time.Second)
	l := core.Lead{
		ID: "l1", Name: "Acme", Company: "Acme Inc", Signals: []string{"s1", "s2"},
		BestScore: 88, Status: core.LeadNew, Notes: "n", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.SaveLead(ctx, l); err != nil {
		t.Fatalf("save lead: %v", err)
	}
	got, err := s.GetLead(ctx, "l1")
	if err != nil {
		t.Fatalf("get lead: %v", err)
	}
	if got.Name != "Acme" || len(got.Signals) != 2 {
		t.Errorf("lead round trip mismatch: %+v", got)
	}

	// Re-saving with one fewer signal must replace, not accumulate.
	l.Signals = []string{"s1"}
	if err := s.SaveLead(ctx, l); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetLead(ctx, "l1")
	if len(got.Signals) != 1 {
		t.Errorf("resave should replace links, got %d signals", len(got.Signals))
	}

	byStatus, err := s.ListLeads(ctx, core.LeadNew)
	if err != nil || len(byStatus) != 1 {
		t.Errorf("ListLeads(new) = %d, %v", len(byStatus), err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	// No cursor yet means the zero value, not an error.
	c, err := s.GetCursor(ctx, "hackernews")
	if err != nil || !c.LastSeenAt.IsZero() {
		t.Fatalf("empty cursor = %+v, %v", c, err)
	}
	want := Cursor{LastSeenAt: time.Now().UTC().Truncate(time.Second), LastExternalID: "42"}
	if err := s.SetCursor(ctx, "hackernews", want); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCursor(ctx, "hackernews")
	if err != nil || got.LastExternalID != "42" || !got.LastSeenAt.Equal(want.LastSeenAt) {
		t.Errorf("cursor round trip: %+v, %v", got, err)
	}
}

func TestSpendLedgerAccumulates(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	day := "2026-07-21"
	t1, err := s.AddSpend(ctx, day, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := s.AddSpend(ctx, day, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	if t1 != 0.10 || t2 < 0.149 || t2 > 0.151 {
		t.Errorf("running totals wrong: %v then %v", t1, t2)
	}
	sp, err := s.GetSpend(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	if sp.Calls != 2 {
		t.Errorf("calls = %d, want 2", sp.Calls)
	}
}

func TestScoreCacheRoundTrip(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	sc := core.Score{SignalID: "x", Value: 70, IntentType: core.IntentPain, Reasons: []string{"r"}}
	if _, ok, _ := s.GetCachedScore(ctx, "h", "m", "fp"); ok {
		t.Fatal("cache should miss before a put")
	}
	if err := s.PutCachedScore(ctx, "h", "m", "fp", sc); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.GetCachedScore(ctx, "h", "m", "fp")
	if err != nil || !ok {
		t.Fatalf("cache should hit: ok=%v err=%v", ok, err)
	}
	if got.Value != 70 || len(got.Reasons) != 1 {
		t.Errorf("cached score mismatch: %+v", got)
	}
	// A different ICP fingerprint must miss — that is what makes editing the ICP
	// invalidate stale reasoning.
	if _, ok, _ := s.GetCachedScore(ctx, "h", "m", "different-fp"); ok {
		t.Error("a different ICP fingerprint must not hit the cache")
	}
}
