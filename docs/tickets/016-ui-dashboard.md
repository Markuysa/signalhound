---
id: 16
title: UI: Dashboard
role: dev
depends: [2]
status: todo
---

Build the Dashboard screen: the four stat cards, the signals chart, the waterfall heatmap, and
the live SSE feed. This is the screenshot that sells the project, so the DESIGN.md spec is
literal, not indicative.

Spec: `docs/PRD.md` §3.2.1, §3.3. Design: `docs/DESIGN.md` §3.2, §4, §5, §7.1.
Reference mockup: `signalhound-ui/dashboard.html`.


## Likely files
- `ui/src/pages/Dashboard.tsx`
- `ui/src/components/{StatCard,SignalsChart,Waterfall,LiveFeed}.tsx`

## Acceptance criteria
- [ ] Four stat cards — Signals 24h, Hot leads, LLM spend, Top source — with every number in
      JetBrains Mono and the contextual number format of `docs/DESIGN.md` §5
      (`$0.68 / $2.00 cap`, `147 signals · 24h`), never a bare figure.
- [ ] A 14-day signals area chart in amber, implemented as a hand-rolled SVG component. Do not
      add Recharts or any charting dependency (`docs/DESIGN.md` §8).
- [ ] A 7×24 waterfall heatmap using the exact five-step intensity ramp of `docs/DESIGN.md` §3.2,
      with 2px gaps and 2px radius cells.
- [ ] A live list of recent hot signals driven by the SSE hook, prepending new rows without a
      full refetch (PRD §3.2.1).
- [ ] Empty state when there is no data: icon, one line of guidance, one action button — no
      illustration (`docs/DESIGN.md` §4, PRD §3.3).
- [ ] Layout and visual hierarchy match `signalhound-ui/dashboard.html`. Responsive at 375, 768,
      1024 and 1440 (`docs/DESIGN.md` §6).
- [ ] No hardcoded colours or spacing — all values come from the tokens defined in #2
      (CLAUDE.md).
- [ ] Component tests cover: rendering from fixture data, the empty state, and an SSE event
      prepending a row. `npm test` and `npm run lint` pass.
