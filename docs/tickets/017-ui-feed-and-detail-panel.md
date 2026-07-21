---
id: 17
title: UI: Feed and detail panel
role: frontend
depends: [2]
status: todo
---

Build the Feed — the main screen. Signal cards, filters, and the detail panel where the score
explanation and the draft live. This is where the product's explainability promise becomes
visible.

Spec: `docs/PRD.md` §3.2.2, §3.3. Design: `docs/DESIGN.md` §3.1, §3.4, §4, §5.
Reference mockup: `signalhound-ui/feed.html`. ADR-006.


## Likely files
- `ui/src/pages/Feed.tsx`
- `ui/src/components/{SignalCard,DetailPanel,FilterChips,IntentTag}.tsx`

## Acceptance criteria
- [ ] Signal cards use the `64px 1fr auto` grid of `docs/DESIGN.md` §4: meter on the left, then a
      meta row (source + intent tag + language), the title at 600 weight, and a two-line snippet
      with an amber `<mark>` on the key phrase.
- [ ] Score-band styling is exact: hot cards get `border-left: 2px solid --hot`; the selected
      card gets an amber border at 45%; cards scoring below 40 render at `opacity .65`.
- [ ] Intent tags follow §4: mono 10px uppercase, 12% background — `pain`→hot, `hiring`→live,
      `buying`/`research`→amber.
- [ ] Filter chips for score, source and intent, with mono counts at `.75` opacity; the active
      chip uses the amber-soft background.
- [ ] Detail panel (380px): 34px mono score with meter, title, the evidence quote with a 2px
      amber left border, "Why this score" built from `reasons` and `icp_match`, the draft in a
      dashed box, then the actions.
- [ ] The draft has a **Copy** button and the line "Drafts are never sent automatically. You
      review, you send." There is no Send control anywhere in this screen (ADR-006,
      `docs/DESIGN.md` §3.4).
- [ ] Actions — add to leads, ignore signal, ignore author — apply optimistically and reconcile
      on the response, rolling back on failure (PRD §3.3, ≤200ms interaction budget).
- [ ] The detail panel is hidden below the 1180px breakpoint (`docs/DESIGN.md` §2.3).
- [ ] Empty state for a filter combination with no results: icon, one line, one action.
- [ ] No hardcoded colours or spacing — tokens only (CLAUDE.md).
- [ ] Tests cover: card rendering in each score band, filter interaction, opening the panel from
      a card, Copy writing to the clipboard, and an optimistic update rolling back on an API
      error. `npm test` and `npm run lint` pass.
