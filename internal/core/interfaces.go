package core

import (
	"context"
	"time"
)

// Collector polls one source for signals published since a cursor.
//
// Collect must be incremental: the runner persists a cursor per source and passes it
// back on the next tick. Rate limiting and scheduling belong to the runner, not here,
// so limits stay configurable without touching collector code.
type Collector interface {
	Name() string
	Collect(ctx context.Context, since time.Time) ([]RawSignal, error)
}

// CompletionRequest is a provider-agnostic LLM call.
type CompletionRequest struct {
	System      string
	User        string
	Model       string
	MaxTokens   int
	Temperature float64
	JSONSchema  []byte // when set, the provider requests structured output
}

// CompletionResponse carries the reply and what it cost.
type CompletionResponse struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	LatencyMS    int
}

// LLMProvider is any model backend. Implementations are wrapped by decorators for
// budget capping and score caching rather than each one reimplementing them (ADR-003).
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// Scorer judges one Signal against the configured ICP.
//
// Implementations return ErrBudgetExceeded when the daily cap is reached and
// ErrInvalidScoreJSON when the model's reply cannot be parsed after one retry.
type Scorer interface {
	Score(ctx context.Context, s Signal) (Score, error)
}

// Notifier delivers a scored signal to a human. Implementations never send outreach to
// the signal's author — see ADR-006; that is a product principle, not a scope decision.
type Notifier interface {
	Name() string
	Notify(ctx context.Context, s Signal, sc Score) error
}

// Stage is one element of the processing pipeline. Stages are connected by buffered
// channels and run as goroutines under an errgroup; a stage that returns an error
// cancels the group and leaves the source cursor unadvanced so the next tick retries.
//
// A stage owns closing its out channel when it returns. See ADR-004.
type Stage interface {
	Name() string
	Run(ctx context.Context, in <-chan Signal, out chan<- Signal) error
}
