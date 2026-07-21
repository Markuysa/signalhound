---
id: 6
title: Collector runner: scheduling, rate limiting, cursors
role: backend
depends: [1]
status: todo
---

Build the component that owns *when* and *how often* collectors run: per-source cron schedules,
rate limits, backoff, and cursor persistence. Collectors stay dumb; the runner is where the
operational policy lives.

Spec: `docs/PRD.md` §2.2.1 (requirements list). Design: `docs/ARCHITECTURE.md` "Flow",
"External integrations". ADR-005.


## Likely files
- `internal/collect/runner/{runner,schedule,limit}.go`
- `internal/collect/runner/runner_test.go`

## Acceptance criteria
- [ ] Runs each registered `core.Collector` on its own goroutine with its own cron schedule from
      config (PRD §2.2.1). An invalid cron expression fails at startup with the offending source
      named.
- [ ] Per-source rate limiting via `golang.org/x/time/rate`, with limits taken from config and
      owned here rather than by any collector (`docs/ARCHITECTURE.md` "External integrations").
- [ ] Retries with exponential backoff, maximum 3 attempts, then logs the failure and skips the
      tick rather than crashing the process.
- [ ] Reads `last_seen` from `source_cursors` before calling `Collect`, and advances it **only
      after** the downstream consumer reports success. A consumer error leaves the cursor
      unchanged so the next tick reprocesses the window (ADR-004).
- [ ] Collected `RawSignal`s are handed to an injected consumer function. The runner does not
      import `internal/pipeline` or any collector implementation (ADR-005).
- [ ] Stops cleanly on context cancellation: in-flight collections are allowed to finish or are
      cancelled within a bounded timeout, and `Run` returns.
- [ ] Tests use fake collectors and an injected clock, covering: the schedule firing, the
      backoff sequence, the cursor not advancing on consumer failure, per-source rate limits
      being independent, and clean shutdown. Suite passes with `-race`.
- [ ] `golangci-lint run` passes.
