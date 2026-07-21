// Package store persists everything except settings (config.yaml is the source of truth
// for those — ADR-007). It exposes a Store interface with one SQLite implementation;
// the interface exists from day one so Postgres can be added later without touching
// callers (ADR-002).
package store

import (
	"context"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// SignalFilter narrows a signal query. A zero value matches everything.
type SignalFilter struct {
	ScoreGTE int
	Source   string
	Intent   string
	Since    time.Time
	Limit    int
	Cursor   string // opaque; the id to page after
}

// Cursor is a collector's incremental position in a source.
type Cursor struct {
	LastSeenAt     time.Time
	LastExternalID string
}

// Spend is one day's LLM ledger row.
type Spend struct {
	Day   string // YYYY-MM-DD
	USD   float64
	Calls int
}

// Store is the persistence seam. Every method takes a context so callers control
// cancellation and timeouts.
type Store interface {
	// Signals. SaveSignal is idempotent on content_hash: a duplicate is a no-op that
	// reports inserted=false, which is how dedup avoids re-scoring the same post.
	SaveSignal(ctx context.Context, s core.Signal) (inserted bool, err error)
	GetSignal(ctx context.Context, id string) (core.Signal, error)
	ListSignals(ctx context.Context, f SignalFilter) ([]core.Signal, string, error) // items, next cursor

	// Scores.
	SaveScore(ctx context.Context, sc core.Score) error
	GetScore(ctx context.Context, signalID string) (core.Score, error)

	// Leads.
	SaveLead(ctx context.Context, l core.Lead) error
	GetLead(ctx context.Context, id string) (core.Lead, error)
	ListLeads(ctx context.Context, status string) ([]core.Lead, error)

	// Source cursors.
	GetCursor(ctx context.Context, source string) (Cursor, error)
	SetCursor(ctx context.Context, source string, c Cursor) error

	// LLM spend ledger. AddSpend increments the day's row atomically and returns the
	// running total, so the budget decorator can stop at the cap without a race.
	AddSpend(ctx context.Context, day string, usd float64) (total float64, err error)
	GetSpend(ctx context.Context, day string) (Spend, error)

	// Score cache, keyed on content hash + model + ICP fingerprint.
	GetCachedScore(ctx context.Context, contentHash, model, icpFingerprint string) (core.Score, bool, error)
	PutCachedScore(ctx context.Context, contentHash, model, icpFingerprint string, sc core.Score) error

	Close() error
}
