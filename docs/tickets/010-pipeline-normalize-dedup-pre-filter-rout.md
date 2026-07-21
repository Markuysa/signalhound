---
id: 10
title: Pipeline: normalize, dedup, pre-filter, route
role: backend
depends: [1]
status: todo
---

Wire the processing stages: normalize → dedup → pre-filter → score → route, as channel-connected
goroutines. The pre-filter is what keeps the LLM bill down, so it must provably run before any
scoring.

Spec: `docs/PRD.md` §2.2.2. Design: `docs/ARCHITECTURE.md` "Flow". ADR-004, ADR-005.


## Likely files
- `internal/pipeline/{pipeline,normalize,dedup,prefilter,route}.go`
- `internal/pipeline/pipeline_test.go`

## Acceptance criteria
- [ ] Each stage implements `core.Stage`. Stages are connected by buffered channels, run as
      goroutines under an `errgroup` tied to the run context, in the order normalize → dedup →
      prefilter → score → route (ADR-004).
- [ ] Normalize: `RawSignal` → `Signal` with a ULID, a `ContentHash` computed over normalized
      title+content, a detected `Lang`, and `CollectedAt`.
- [ ] Dedup: drops a signal whose `content_hash` already exists in the store, and drops
      near-identical titles already seen from another source within the same run.
- [ ] Pre-filter: applies keyword include/exclude, the language allowlist and a minimum length
      **before any LLM call**. A table test proves a filtered signal never reaches the scorer.
- [ ] Route: a score at or above `scoring.notify_threshold` goes to the notifier; **every**
      signal is persisted regardless of score (PRD §2.2.2 step 6).
- [ ] `core.ErrBudgetExceeded` from the scorer stores the signal unscored and halts the scoring
      stage for the remainder of the run, instead of failing the whole pipeline.
- [ ] The scoring stage supports configurable concurrency > 1; the cheap stages stay
      single-goroutine.
- [ ] Every stage exits on `ctx.Done()` and closes its output channel exactly once. The full
      suite passes with `-race`, run at least 5 times without flaking.
- [ ] Tests use a fake `core.Scorer` and a fake `core.Notifier` and a temp store — no LLM, no
      network (CLAUDE.md). Cases include: dedup across sources, prefilter rejection, threshold
      routing, budget exceeded mid-run, context cancellation.
- [ ] This package imports no collector, no provider implementation and no notifier
      implementation (ADR-005).
- [ ] `golangci-lint run` passes.
