---
id: 19
title: Wiring: composition root, serve and demo commands, UI embed
role: dev
depends: [1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16, 17, 18]
status: todo
---

The composition root. Every other ticket builds a package that depends only on `internal/core`;
this one is the single place that knows about concrete implementations and turns them into a
running program. It is deliberately the only serialization point in the plan (ADR-005).

Spec: `docs/PRD.md` §2.2.7, §3.4. Design: `docs/ARCHITECTURE.md` "Package layout", ADR-001,
ADR-003, ADR-005.


## Likely files
- `internal/app/{app,lifecycle}.go`
- `internal/webui/embed.go`
- `cmd/signalhound/{main,serve,demo,version}.go`

## Acceptance criteria
- [ ] `internal/app` is the only package importing concrete implementations. It builds the
      store, collectors, runner, LLM provider, llmguard chain, scorer, pipeline, notifiers and
      API from config (ADR-005).
- [ ] The llmguard chain is composed **cache outside budget**, with a comment at the composition
      site explaining that cached hits must not bill against the cap (ADR-003).
- [ ] `signalhound serve` starts everything, serves the embedded UI at `/` and the API at
      `/api`, and shuts down gracefully on SIGINT/SIGTERM: collectors stop, the in-flight
      pipeline drains, the store closes — all within a bounded timeout, with no goroutine leak.
- [ ] `signalhound --demo` (or the `demo` subcommand) seeds an ephemeral store via
      `internal/demo` and serves the UI with **no config file, no API key and no network
      access** (PRD §3.4).
- [ ] `internal/webui` serves `ui/dist` from an `embed.FS`, with SPA fallback to `index.html`
      for unknown non-`/api` routes and correct content types.
- [ ] Structured logging with `slog` throughout, and `/metrics` exposing Prometheus counters for
      signals collected, signals scored, LLM cost and collector errors (PRD §2.2.7).
- [ ] An end-to-end test starts the app with fake collectors and a fake LLM provider and asserts
      a signal travels collect → score → store → API response → SSE event.
- [ ] `go build -o bin/signalhound ./cmd/signalhound` produces a single binary that runs with
      only `config.yaml` present.
- [ ] `go test ./... -race` and `golangci-lint run` pass across the whole repo.

## Notes for the implementer
If an interface from #1 turns out to be wrong, changing it here breaks merged packages. Comment
on this issue with what is wrong before reshaping `internal/core`.
