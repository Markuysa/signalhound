---
id: 2
title: UI scaffold: design tokens, app shell, signal meter, API client
role: dev
depends: []
status: done
---

Stand up the React/Vite/TypeScript/Tailwind app, encode the design tokens once, and build the
shared shell and primitives the three screen tickets will consume. No backend dependency — it
codes against the frozen API contract in `docs/ARCHITECTURE.md` ("API contract").

Design: `docs/DESIGN.md` §2 (tokens), §3.1 (signal meter), §6 (quality checklist), §8.
Spec: `docs/PRD.md` §3.1, §3.3. ADR-001.


## Likely files
- `ui/package.json`, `ui/vite.config.ts`, `ui/tsconfig.json`, `ui/tailwind.config.ts`
- `ui/src/main.tsx`, `ui/src/App.tsx`, `ui/src/styles/tokens.css`
- `ui/src/lib/api.ts`, `ui/src/lib/sse.ts`
- `ui/src/components/{Sidebar,SignalMeter,Chip,Button,EmptyState}.tsx`
- `ui/src/fonts/*.woff2`

## Acceptance criteria
- [ ] Vite + React + TypeScript + Tailwind; `npm run build`, `npm run lint` and `npm test` all
      pass.
- [ ] Every token in `docs/DESIGN.md` §2 is defined once as a CSS variable and surfaced through
      `tailwind.config`. No hardcoded hex colour or raw spacing value anywhere in `ui/src/` —
      enforced by a lint rule or a test that greps for hex literals (CLAUDE.md: hardcoded
      colours are a defect, not a nit).
- [ ] Space Grotesk, Inter and JetBrains Mono are self-hosted as woff2; the built app makes no
      external font or CDN request at runtime (`docs/DESIGN.md` §8).
- [ ] `SignalMeter` renders the five-stroke scale with the exact fill and colour bands of
      `docs/DESIGN.md` §3.1, in `standard` (22px) and `small` (14px) sizes, with the score as a
      mono numeral. Snapshot-tested at scores 12, 45, 70 and 92.
- [ ] App shell: fixed 216px sidebar with navigation and the LLM budget bar in its footer, dark
      theme only (`color-scheme: dark`), routes for `/dashboard`, `/feed`, `/settings`.
- [ ] A typed API client covers every endpoint in the `docs/ARCHITECTURE.md` contract table,
      plus an SSE hook for `/api/events`. Auth is the session cookie from `POST /api/session`.
- [ ] The Vite dev server proxies `/api` to the Go process so UI and backend can be developed
      independently.
- [ ] Icons are `lucide-react` SVG — no emoji. Visible focus states (2px amber outline, 2px
      offset) and `prefers-reduced-motion: reduce` disabling all motion
      (`docs/DESIGN.md` §2.4, §6).
- [ ] Responsive behaviour at the §2.3 breakpoints: detail panel hidden below 1180px, sidebar
      collapses below 820px.

## Notes for the implementer
The static mockups in `signalhound-ui/*.html` are the visual reference for the shell and the
meter. Do not build the Dashboard, Feed or Settings screens here — they are separate tickets
that will import from this one.
