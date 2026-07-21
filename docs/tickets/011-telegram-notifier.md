---
id: 11
title: Telegram notifier
role: dev
depends: [1]
status: todo
---

Deliver qualified leads to the operator's Telegram chat as a readable card with the score, the
reasoning, and the draft.

Spec: `docs/PRD.md` §3.4 (MVP DoD). Design: `docs/DESIGN.md` §3.4, §5. ADR-006.


## Likely files
- `internal/notify/{notify,telegram}.go`
- `internal/notify/telegram_test.go`
- `internal/notify/testdata/*.json`

## Acceptance criteria
- [ ] Implements `core.Notifier`. The bot token comes from the environment; `chat_id` comes from
      `notifiers.telegram.chat_id` in config (ADR-007).
- [ ] The message renders a lead card: score, source, title, link to the original post, the top
      reasons, and the draft reply — within Telegram's 4096-character limit. When truncation is
      needed, the draft is truncated and **the reasons are never dropped**.
- [ ] Sends only to the configured operator `chat_id`. No code path accepts a recipient derived
      from a signal, a lead, or an API request (ADR-006) — this is checked in review, so keep
      the signature narrow.
- [ ] The message includes the line that drafts are never sent automatically
      (`docs/DESIGN.md` §3.4).
- [ ] A send failure returns an error and is retried once; it never blocks or fails the caller's
      pipeline run.
- [ ] Tests use recorded fixtures and an injected `http.Client`, covering: card rendering,
      truncation of an over-long draft, HTTP 429, and retry-then-success. No network
      (PRD §2.2.7, CLAUDE.md).
- [ ] `go test ./internal/notify/... -race` and `golangci-lint run` pass.
