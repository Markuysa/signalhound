# ADR-004: Pipeline as channel-connected stages

Date: 2026-07-21
Status: accepted

## Context

PRD §2.2.2 describes processing as ordered stages. Implementation options: one function that
does all steps per signal; stages connected by channels and run as goroutines; or an external
queue (NATS/Redis) between stages.

An external queue was rejected outright — it adds a second process to a single-binary product.
The real choice is a straight-line loop versus channel-connected stages.

## Decision

Each stage implements `core.Stage` and runs as its own goroutine, connected by buffered
channels, supervised by an `errgroup` tied to the run's context.

Stage order: normalize → dedup → prefilter → score → route.

The expensive stage (score) is the only one that may run with concurrency > 1, configurable.
A stage returning an error cancels the group; the source cursor is not advanced, so the next
scheduled tick reprocesses the window. Dedup makes that retry safe.

## Consequences

- The LLM stage can be parallelized without touching the cheap stages, which is the whole point:
  pre-filter is supposed to discard ~80% of input before anything costs money (PRD §2.2.2).
- Each stage is unit-testable by feeding a channel — no store and no LLM needed for the cheap
  stages.
- Cost of the choice: goroutine plumbing and shutdown ordering are easy to get subtly wrong.
  Every stage must exit on `ctx.Done()` and must close its output channel exactly once. This is
  the part of the codebase most likely to produce a flaky test, and tests must run with `-race`.
- "Reprocess the whole window on any error" is deliberately dumb. Per-signal error isolation is
  a v0.2 refinement, not an MVP requirement.
