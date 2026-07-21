---
id: 13
title: Demo mode seed data
role: backend
depends: [1]
status: todo
---

Produce the seed dataset behind `--demo`, so anyone can see the UI without a config file, an API
key, or a network connection. This is the ticket that makes the README screenshots and the demo
gif possible.

Spec: `docs/PRD.md` §3.4 (MVP DoD), §3.3 (screenshot quality is a stated requirement).
Design: `docs/DESIGN.md` §3.1 (score bands), §4 (intent tags).


## Likely files
- `internal/demo/seed.go`
- `internal/demo/seed_test.go`
- `internal/demo/data/*.json`

## Acceptance criteria
- [ ] `Seed(ctx, store) error` populates a realistic dataset: at least 120 signals across
      `hackernews`, `reddit:*` and `rss:*`; scores spanning all four `docs/DESIGN.md` §3.1 bands
      (0–40, 41–60, 61–84, 85+); all five intent types; at least 15 leads covering every status;
      and 14 days of LLM spend history.
- [ ] Every seeded score carries a non-empty `reasons` and `icp_match` — the demo exists to show
      explainability, and blank panels defeat it (PRD §2.2.4).
- [ ] Deterministic: the same seed value produces identical IDs and timestamps relative to a
      fixed "now", so screenshots are reproducible across runs.
- [ ] Seeding twice is idempotent, not duplicative.
- [ ] The data is plausible but entirely fabricated — no real usernames, no real company names,
      no URLs pointing at live posts.
- [ ] The package exposes only `Seed`. The `--demo` flag and the ephemeral store belong to the
      wiring ticket (#19).
- [ ] `go test ./internal/demo/... -race` and `golangci-lint run` pass.
