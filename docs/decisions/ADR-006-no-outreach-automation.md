# ADR-006: No outreach automation — the agent drafts, a human sends

Date: 2026-07-21
Status: accepted

## Context

Every scored signal carries a `DraftReply`. The mechanical distance between "we have a draft and
the author's handle" and "we posted it" is one HTTP call, and there will be feature requests for
it. The options were: ship sending; ship sending behind an off-by-default flag; or refuse it
architecturally.

The off-by-default flag is the tempting middle. It was rejected because a flag is documentation,
not a boundary — once the send code exists, the product is a spam tool with a default, and the
differentiator from AI SDR platforms (PRD §1.3) evaporates.

## Decision

No sending, ever, in any form, behind any flag. Concretely, this constrains the code:

- No API endpoint accepts a destination address, handle, or recipient.
- No collector credential is granted write scope. Reddit OAuth is app-only read; the Telegram
  bot token sends only to the operator's own chat, never to a lead.
- The UI offers **Copy** on every draft, never **Send** (`docs/DESIGN.md` §3.4), and states
  under each draft: *"Drafts are never sent automatically. You review, you send."*

## Consequences

- Rules out a whole class of features and some users. That is the intent — it is the ethical
  position the project is marketed on (PRD §1.3), so it must be visible in the code, not only
  in the README.
- Sidesteps platform ToS risk, deliverability engineering, and abuse handling entirely.
- Notifications to the operator (Telegram, Slack) are not outreach and are unaffected. The line
  is the recipient: the operator, always; never the lead.
- This is the one decision in this set that should not be revisited on convenience grounds. A
  PR adding a send path is closed, not reviewed.
