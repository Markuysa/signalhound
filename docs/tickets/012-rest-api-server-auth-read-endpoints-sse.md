---
id: 12
title: REST API: server, auth, read endpoints, SSE
role: dev
depends: [1]
status: todo
---

Build the HTTP surface the UI consumes: router, auth, the read endpoints, lead mutations, and the
SSE event stream. Config and test-scoring endpoints are a follow-up ticket (#15) because they
touch the same router file.

Spec: `docs/PRD.md` §2.2.6. Design: `docs/ARCHITECTURE.md` "API contract" — that table is the
contract, implement it exactly.


## Likely files
- `internal/api/{server,auth,signals,leads,stats,events}.go`
- `internal/api/server_test.go`

## Acceptance criteria
- [ ] A `chi` router mounted at `/api`, matching the contract table in `docs/ARCHITECTURE.md`
      exactly — paths, methods and response shapes.
- [ ] `GET /api/signals` supports `score_gte`, `source`, `intent` and `period` filters plus
      cursor pagination (`cursor`, `limit`), returning `{items, next_cursor}`.
- [ ] `GET /api/signals/{id}` returns the signal together with its score, including `reasons`
      and `icp_match` (PRD §2.2.4 — explainability is not optional).
- [ ] `GET /api/leads` returns leads grouped by status. `PATCH /api/leads/{id}` accepts
      `{status?, notes?}` and rejects an unknown status with 422.
- [ ] `GET /api/stats` returns the signals-per-day series, LLM spend against the cap, the score
      histogram, and the per-source funnel (PRD §2.2.6).
- [ ] Auth middleware: `Authorization: Bearer <token>` validated against
      `SIGNALHOUND_API_TOKEN`; `POST /api/session` exchanges a valid token for a session cookie
      for the embedded UI. Missing or invalid credentials return 401, and no handler reads or
      leaks data before the check runs.
- [ ] `GET /api/events` is an SSE stream emitting `signal.scored`, `budget.exceeded` and
      `collector.error`, with periodic heartbeats. A disconnecting client is cleaned up with no
      goroutine leak — proven by a `-race` test with N subscribers connecting and dropping.
- [ ] No endpoint accepts a recipient, address or handle, and no endpoint sends anything
      outward (ADR-006).
- [ ] Handler tests use `httptest` and a temporary store — no live server, no network.
- [ ] `go test ./internal/api/... -race` and `golangci-lint run` pass.

## Notes for the implementer
Leave the router extensible but do not add config or score-test routes — #15 adds those and will
conflict if you pre-empt it.
