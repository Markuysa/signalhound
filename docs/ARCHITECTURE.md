# Architecture

Scope: MVP v0.1, as defined by `docs/PRD.md` §3.4.

**State of the repo at the time of writing:** no Go code, no `go.mod`, no `ui/` directory.
The only existing assets are the design system (`docs/DESIGN.md`), the static HTML mockups
(`signalhound-ui/*.html`, `signalhound-design-concept.html`), and the spec. Nothing below is
a reading of existing code — it is all being introduced.

---

## Package layout

```
cmd/signalhound/          CLI entrypoint: serve, demo, version
internal/
  core/                   domain types + every cross-package interface. Imports nothing internal.
  config/                 config.yaml load, validate, hot reload
  store/                  Store interface + SQLite implementation + goose migrations
  collect/
    hn/                   Hacker News collector (Algolia HN Search API)
    reddit/               Reddit collector (OAuth app-only)
    rss/                  RSS/Atom collector
    runner/               per-source scheduling, rate limiting, cursor persistence
  llm/                    OpenAI-compatible and Anthropic providers, cost accounting
  llmguard/               decorators over core.LLMProvider: daily budget cap, score cache
  scoring/                ICP prompt assembly, strict-JSON parsing, core.Scorer implementation
  pipeline/               normalize → dedup → prefilter → score → route, channel-connected
  notify/                 Notifier interface implementations (Telegram for MVP)
  demo/                   deterministic seed data for `--demo`
  api/                    REST + SSE over chi; single-token auth
  webui/                  embed.FS wrapper around ui/dist
  app/                    composition root: builds everything from config, owns lifecycle
ui/                       React + Vite + TypeScript + Tailwind, built to ui/dist
```

**The dependency rule:** every feature package depends on `internal/core` and, where it needs
persistence, on `internal/store`. Feature packages never import each other. Only
`internal/app` imports concrete implementations. This is what makes the tickets parallelizable
— see [ADR-005](decisions/ADR-005-composition-root.md).

---

## Data model

`internal/core` owns the types. Fields beyond PRD §2.2.4 are marked **(+)**.

```go
type RawSignal struct {                 // what a collector emits, pre-normalization
    Source     string
    ExternalID string                   // (+) source-native id, for cursor and dedup
    URL, Author, Title, Content string
    PublishedAt time.Time
    Meta       map[string]string        // (+) source-specific extras (subreddit, points, …)
}

type Signal struct {
    ID          string    // ULID
    Source      string    // "hackernews", "reddit:r/golang", "rss:<feed-name>"
    URL, Author, Title, Content string
    Lang        string    // (+) detected, used by pre-filter
    PublishedAt, CollectedAt time.Time
    ContentHash string    // sha256 of normalized title+content
}

type Score struct {
    SignalID   string
    Value      int       // 0-100
    IntentType string    // buying | pain | hiring | research | none
    Reasons    []string  // mandatory, never empty
    ICPMatch   []string
    DraftReply string    // never sent; see ADR-006
    Model      string
    CostUSD    float64
    LatencyMS  int       // (+) surfaced by test-scoring UI
    ScoredAt   time.Time // (+)
}

type Lead struct {
    ID, Name, Company string
    Signals   []string
    BestScore int
    Status    string    // new | reviewed | contacted | ignored | customer
    Notes     string
    CreatedAt, UpdatedAt time.Time // (+)
}
```

### Interfaces (all in `internal/core`)

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context, since time.Time) ([]RawSignal, error)
}

type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type Scorer interface {                 // implemented by internal/scoring
    Score(ctx context.Context, s Signal) (Score, error)
}

type Notifier interface {
    Name() string
    Notify(ctx context.Context, s Signal, sc Score) error
}

type Stage interface {                  // pipeline element
    Name() string
    Run(ctx context.Context, in <-chan Signal, out chan<- Signal) error
}
```

`ErrBudgetExceeded` and `ErrInvalidScoreJSON` are sentinel errors in `core`, so `llmguard`
and `scoring` can be handled by the pipeline without importing either.

### Tables (goose migrations, SQLite dialect first)

| Table | Purpose |
|---|---|
| `signals` | one row per deduplicated signal; unique index on `content_hash` |
| `scores` | one row per scored signal; `reasons` and `icp_match` stored as JSON arrays |
| `leads`, `lead_signals` | mini-CRM; leads are v0.2 in the UI but the tables land now |
| `source_cursors` | `source` → `last_seen_at`, `last_external_id`; incremental collection |
| `llm_spend` | `day`, `usd`, `calls` — the daily budget ledger |
| `score_cache` | `content_hash`+`model`+`icp_fingerprint` → score JSON |

`score_cache` is keyed on an ICP fingerprint (hash of the ICP block of the config) so that
editing the ICP invalidates cached scores rather than serving stale reasoning.

---

## Flow

```
runner (cron per source)
   └─▶ Collector.Collect(since cursor) ──▶ []RawSignal
          └─▶ pipeline: normalize ─▶ dedup ─▶ prefilter ─▶ score ─▶ route
                                      │         │           │        ├─▶ store
                                      │         │           │        └─▶ Notifier (if ≥ threshold)
                                      │         │           └─ scoring → llmguard → llm provider
                                      │         └─ keywords / lang / min length (no LLM cost)
                                      └─ content_hash lookup + normalized-title similarity
```

Stages are connected by buffered channels and run as goroutines under an `errgroup`. Back
pressure is the channel; a stage that errors cancels the group and the run ends with the
cursor unadvanced, so the next tick retries. See
[ADR-004](decisions/ADR-004-pipeline-stages.md).

---

## API contract

All routes are under `/api`. Auth: `Authorization: Bearer <token>` where the token comes from
`SIGNALHOUND_API_TOKEN`; the embedded UI exchanges it once for a session cookie via
`POST /api/session`. Multi-user is out of scope (PRD §2.2.6).

| Method | Path | Notes |
|---|---|---|
| GET | `/api/signals` | filters `score_gte`, `source`, `intent`, `period`; cursor pagination (`cursor`, `limit`), returns `{items, next_cursor}` |
| GET | `/api/signals/{id}` | signal + its score (reasons, icp_match, draft) |
| GET | `/api/leads` | grouped by status |
| PATCH | `/api/leads/{id}` | `{status?, notes?}` |
| GET | `/api/stats` | signals/day series, spend, score histogram, per-source funnel |
| GET | `/api/config` | current config as YAML text + parsed struct |
| PUT | `/api/config` | validate → write file → hot reload; 422 with field errors on invalid |
| POST | `/api/score/test` | `{text}` → score + reasons + cost + latency; does not persist |
| GET | `/api/events` | SSE stream of `signal.scored`, `budget.exceeded`, `collector.error` |
| GET | `/metrics` | Prometheus, outside `/api`, no auth by default |

This table is the contract the UI tickets code against; it is frozen before any UI work
starts so UI and API can be built in parallel.

---

## External integrations

| Integration | Auth | Failure policy |
|---|---|---|
| Algolia HN Search API | none | retry with exponential backoff, 3 attempts, then log + skip tick |
| Reddit API | OAuth2 client credentials (app-only) | token refresh on 401; auth failure surfaces as a `collector.error` SSE event and a UI banner |
| RSS/Atom feeds | none | per-feed isolation — one bad feed never fails the tick |
| OpenAI-compatible / Anthropic | API key from env | one automatic retry on invalid JSON, then the signal is stored unscored |
| Telegram Bot API | bot token | notification failures are logged and retried once; never block the pipeline |

Rate limits are per-source `golang.org/x/time/rate` limiters owned by the runner, not by the
collectors, so limits are configurable without touching collector code.

---

## Adopted patterns

- [ADR-001 — Single binary with embedded React UI](decisions/ADR-001-single-binary-embedded-ui.md)
- [ADR-002 — SQLite by default behind a Store interface](decisions/ADR-002-sqlite-default.md)
- [ADR-003 — LLM provider interface with decorator chain](decisions/ADR-003-llm-provider-decorators.md)
- [ADR-004 — Pipeline as channel-connected stages](decisions/ADR-004-pipeline-stages.md)
- [ADR-005 — Composition root in internal/app](decisions/ADR-005-composition-root.md)
- [ADR-006 — No outreach automation](decisions/ADR-006-no-outreach-automation.md)
- [ADR-007 — config.yaml is the source of truth](decisions/ADR-007-config-as-source-of-truth.md)

---

## Deliberately not built yet

| Not building | Why | When |
|---|---|---|
| GitHub and job-board collectors | PRD ships 3 collectors in v0.1; the 4th and 5th add no new architecture, only volume | v0.2 |
| Enrichment plugins | Needs an `Enricher` interface we cannot design well without a real enrichment source | v0.2 |
| Postgres store | The `Store` interface exists from day one so this is additive; SQLite covers the $5 VPS target | v0.2 |
| Leads and Stats **UI pages** | Not in the v0.1 DoD. Tables and API endpoints land now so the pages are pure frontend later | v0.2 |
| Slack / Discord / webhook / email notifiers | Telegram alone proves the `Notifier` seam | v0.2 |
| Multi-ICP profiles | Config schema would need a breaking change; doing it once, later, with real usage | v0.3 |
| Fuzzy cross-source dedup beyond normalized-title similarity | Simhash/embedding dedup is a research task, not a ticket | v0.3 |
| LinkedIn / X collectors | Deliberate ToS decision, PRD §2.2.1 | never |
| Any form of sending | Product principle, ADR-006 | never |
