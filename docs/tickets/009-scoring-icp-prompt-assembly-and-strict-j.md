---
id: 9
title: Scoring: ICP prompt assembly and strict-JSON score
role: dev
depends: [1]
status: todo
---

Build the core value of the product: turn an ICP definition plus a signal into an explained
score. This is the ticket where "a score without reasoning is incomplete" is enforced.

Spec: `docs/PRD.md` §2.2.2 (stage 4), §2.2.3, §2.2.4, §2.2.5. Design: `docs/DESIGN.md` §5
(explanation format). ADR-003, ADR-006.


## Likely files
- `internal/scoring/{scorer,prompt,parse}.go`
- `internal/scoring/scoring_test.go`
- `internal/scoring/testdata/*.txt` (golden prompts and fake responses)

## Acceptance criteria
- [ ] Implements `core.Scorer`: a `core.Signal` plus the ICP config produces a `core.Score`.
- [ ] The prompt is assembled from `icp.description`, `icp.positive_signals` and
      `icp.negative_signals` (PRD §2.2.5). A golden-file test pins the rendered prompt so any
      change to it shows up in review.
- [ ] The JSON schema sent to the provider enforces: `value` 0–100, `intent_type` in
      `{buying, pain, hiring, research, none}`, a non-empty `reasons` array, and `icp_match`.
- [ ] A response with an empty `reasons` array is rejected as invalid **even when the JSON
      parses** — a score without reasoning is incomplete (CLAUDE.md rule, PRD §2.2.4).
- [ ] Where the model supplies them, reasons are evidence quotes from the signal text, in the
      format of `docs/DESIGN.md` §5 (`✓ Explicit intent: "considering bringing in outside
      help"`).
- [ ] `DraftReply` is populated, and the package exposes no send path of any kind — no
      recipient, address or handle appears in any signature (ADR-006).
- [ ] `core.ErrBudgetExceeded` returned by the provider is passed through unwrapped so the
      pipeline can detect it with `errors.Is`.
- [ ] `Score.Model`, `Score.CostUSD`, `Score.LatencyMS` and `Score.ScoredAt` are populated from
      the provider response.
- [ ] Every test uses a fake `core.LLMProvider` — never a live API (CLAUDE.md, PRD §2.2.7).
      Cases: valid score, empty reasons, out-of-range value, unknown intent type, budget
      exceeded, invalid JSON.
- [ ] `go test ./internal/scoring/... -race` and `golangci-lint run` pass.
