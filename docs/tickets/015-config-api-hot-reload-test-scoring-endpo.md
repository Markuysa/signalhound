---
id: 15
title: Config API, hot reload, test-scoring endpoint
role: dev
depends: [9, 12]
status: todo
---

Add the two endpoints that make the Settings screen work: config read/write with hot reload, and
the test-scoring endpoint. Split from #12 because both register routes in the same `server.go`,
so they must not run in parallel with it.

Spec: `docs/PRD.md` §2.2.6, §3.2.4 (test scoring is called out as the killer feature).
Design: ADR-007, `docs/DESIGN.md` §3.3.


## Likely files
- `internal/api/{config,scoretest}.go` and the route registration in `internal/api/server.go`
- `internal/config/reload.go`

## Acceptance criteria
- [ ] `GET /api/config` returns the raw YAML text plus the parsed struct. Secrets are redacted
      and never returned in either form (ADR-007).
- [ ] `PUT /api/config` validates before writing. Invalid input returns 422 with per-field
      errors and leaves the file on disk untouched.
- [ ] A valid write is atomic — temp file plus rename, never a truncate-in-place (ADR-007).
- [ ] A successful write triggers hot reload: collector schedules and the scorer pick up the new
      values without a process restart. A test asserts that a live component observes the new
      value, not a startup snapshot.
- [ ] `POST /api/score/test` runs a pasted text through the real scoring path and returns the
      score, `reasons`, `icp_match`, cost and latency — and **persists nothing** (PRD §2.2.6).
- [ ] Test scoring respects the daily budget cap and returns a typed, renderable error when the
      cap is hit rather than a generic 500 (`docs/DESIGN.md` §5).
- [ ] Both endpoints sit behind the existing auth middleware from #12.
- [ ] Tests use `httptest`, a temporary config file, and a fake LLM provider — no live API, no
      network (CLAUDE.md).
- [ ] `go test ./internal/api/... ./internal/config/... -race` and `golangci-lint run` pass.
