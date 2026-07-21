---
id: 8
title: Budget cap and score cache decorators
role: backend
depends: [1]
status: todo
---

Implement the daily budget cap and the score cache as decorators over `core.LLMProvider`, so
both work for every provider and are testable without a prompt or a network.

Spec: `docs/PRD.md` §2.2.3 (budget limit, scoring cache). Design: ADR-003.


## Likely files
- `internal/llmguard/{budget,cache}.go`
- `internal/llmguard/llmguard_test.go`

## Acceptance criteria
- [ ] Budget: wraps a `core.LLMProvider` and returns a `core.LLMProvider`. It reads the day's
      spend from the `llm_spend` ledger and returns `core.ErrBudgetExceeded` **without calling
      the inner provider** once `llm.daily_budget_usd` is reached; it records spend after each
      successful call (PRD §2.2.3).
- [ ] The budget window rolls over at local midnight — a test with an injected clock proves a
      new day resets the counter.
- [ ] Cache: wraps a `core.LLMProvider` and returns a `core.LLMProvider`, keyed on
      `content_hash` + `model` + `icp_fingerprint`. A hit returns the stored response without
      calling the inner provider **and without touching the ledger**.
- [ ] A changed `icp_fingerprint` is a cache miss, so editing the ICP invalidates stale
      reasoning rather than serving it (ADR-007).
- [ ] Both decorators are safe for concurrent use. A `-race` test with N concurrent calls proves
      the cap is never exceeded and the inner provider is called exactly once per distinct cache
      key.
- [ ] Tests use a stub `core.LLMProvider` and a temporary SQLite store — no prompts, no network,
      no real provider.
- [ ] Each decorator is independently usable; neither imports `internal/llm` or
      `internal/scoring` (ADR-005).
- [ ] `go test ./internal/llmguard/... -race` and `golangci-lint run` pass.

## Notes for the implementer
Composition order (cache outside budget) is decided in ADR-003 and applied by the wiring ticket
(#19). Do not encode the order here.
