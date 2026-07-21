---
id: 18
title: UI: ICP & Sources settings with test scoring
role: dev
depends: [2, 15]
status: todo
---

Build the ICP & Sources screen with the sticky Test-scoring panel — the loop that lets a user
tune their ICP prompt and immediately see what it does. PRD calls this the killer feature.

Spec: `docs/PRD.md` §3.2.4, §2.2.5. Design: `docs/DESIGN.md` §3.3, §4, §5.
Reference mockup: `signalhound-ui/icp.html`. ADR-007.


## Likely files
- `ui/src/pages/Settings.tsx`
- `ui/src/components/{ConfigForm,SourceRow,TestScoringPanel,Toggle}.tsx`

## Acceptance criteria
- [ ] Left column: ICP description textarea; positive and negative signal chips with add and
      remove; the source list with per-source toggle, schedule and last-run status; and LLM
      settings including the daily budget cap (PRD §2.2.5).
- [ ] Saving issues a `PUT /api/config` with the whole config. A 422 renders its field errors
      inline on the offending fields, not as a toast.
- [ ] Editing any ICP field warns, before saving, that cached scores will be invalidated and
      re-scoring may cost money (ADR-007 consequence).
- [ ] Right column: a sticky Test-scoring panel — mono textarea → "Run test" → result showing the
      score, the meter, the reasons, and `cost: $0.0004 · 1.2s` (`docs/DESIGN.md` §3.3, §4).
- [ ] A budget-exceeded response renders as an actionable message naming the cap, not a generic
      failure (`docs/DESIGN.md` §5).
- [ ] Inputs, toggles (32×18px, amber when on) and focus states follow `docs/DESIGN.md` §4;
      labels above fields use the mono-uppercase sub style.
- [ ] Buttons name their action — "Run test", "Save & reload" — never "Submit" or "OK"
      (`docs/DESIGN.md` §5).
- [ ] No hardcoded colours or spacing — tokens only (CLAUDE.md).
- [ ] Tests cover: load → edit → save round trip, inline validation errors from a 422, a
      successful test score, and a budget-exceeded response. `npm test` and `npm run lint` pass.
