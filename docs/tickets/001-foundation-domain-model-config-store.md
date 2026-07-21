---
id: 1
title: Foundation: domain model, config, store
role: dev
depends: []
status: done
---

Bootstrap the Go module and the three packages every other ticket depends on: the domain
model, config loading, and the store. Nothing else in the MVP compiles until this lands.

Design: `docs/ARCHITECTURE.md` ("Package layout", "Data model"), ADR-002, ADR-005, ADR-007.
Spec: `docs/PRD.md` §2.2.4, §2.2.5, §2.2.7.


## Likely files
- `go.mod`, `go.sum`
- `internal/core/{signal,score,lead,interfaces,errors}.go`
- `internal/config/{config,validate,load}.go`, `config.example.yaml`
- `internal/store/{store,sqlite}.go`, `internal/store/migrations/*.sql`
- `.golangci.yml`

## Acceptance criteria
- [ ] `go build ./...` and `go test ./...` pass on a clean clone; Go 1.22+.
- [ ] `internal/core` defines `Signal`, `Score`, `Lead`, `RawSignal` exactly as in
      `docs/ARCHITECTURE.md` "Data model", plus the `Collector`, `LLMProvider`, `Scorer`,
      `Notifier` and `Stage` interfaces and the sentinel errors `ErrBudgetExceeded` and
      `ErrInvalidScoreJSON`.
- [ ] `internal/core` imports no other internal package, asserted by a test that inspects its
      imports (ADR-005).
- [ ] `internal/config` loads the `config.yaml` from PRD §2.2.5 verbatim without error; each
      required field, when missing, produces an error naming that field rather than a panic.
- [ ] Secrets (LLM API keys, Reddit credentials, Telegram bot token) are read from the
      environment by reference and never parsed out of the config file (ADR-007).
- [ ] `internal/config` exposes `ICPFingerprint()` returning a hash that changes whenever any
      `icp.*` field changes and is stable otherwise.
- [ ] `internal/store` defines a `Store` interface covering signals, scores, leads, source
      cursors, `llm_spend` and `score_cache`, with one SQLite implementation using
      `modernc.org/sqlite` (pure Go, no cgo — ADR-002).
- [ ] goose migrations are embedded and run automatically on open; opening a fresh temp DB a
      second time is a no-op. A unique index exists on `signals.content_hash`.
- [ ] Each entity has a store round-trip test against a temporary file-backed DB (not
      `:memory:`).
- [ ] `golangci-lint run` passes.

## Notes for the implementer
The interfaces here are consumed by nine parallel tickets, so copy them verbatim from
`docs/ARCHITECTURE.md` rather than improving them. If one is genuinely wrong, comment on this
issue before changing it.
