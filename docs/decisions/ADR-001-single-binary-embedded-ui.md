# ADR-001: Single binary with an embedded React UI

Date: 2026-07-21
Status: accepted

## Context

The product promise is "`docker run` → first lead in 10 minutes" (PRD §1.4.1, §3.3). That rules
out an architecture where the user deploys a backend and a frontend separately. Three options
were on the table:

1. **Go binary + React SPA built to static assets, embedded via `embed.FS`.**
2. **templ + htmx**, a pure Go stack with no Node toolchain at all.
3. **Separate frontend deployment** (Vercel/nginx) talking to the Go API.

Option 3 breaks the single-artifact promise immediately. The real trade is 1 vs 2: htmx would
remove the entire Node build from the release pipeline and shrink the repo, at the cost of the
interactive detail panel, live SSE feed, and waterfall heatmap that `docs/DESIGN.md` specifies —
and screenshot quality is a stated product requirement (PRD §3.3), not polish.

## Decision

React + Vite + TypeScript + Tailwind, built to `ui/dist`, embedded into the binary through
`internal/webui` using `embed.FS`. The Go binary serves the SPA at `/` and the API at `/api`.
Fonts are self-hosted woff2 — a self-hosted product must not call a CDN at runtime
(`docs/DESIGN.md` §8).

## Consequences

- The release pipeline needs Node: `npm run build` must run before `go build`, and CI plus
  goreleaser both have to encode that order. A stale `ui/dist` silently ships an old UI.
- `ui/dist` is gitignored, so a fresh clone cannot `go build` until the UI is built once. The
  build target must handle this or fail with a clear message rather than an `embed` error.
- Frontend and backend can be developed in parallel against the frozen API contract in
  `docs/ARCHITECTURE.md`; in dev the Vite server proxies `/api` to the Go process.
- Reversing this later means rewriting every screen. It is the most expensive decision here,
  which is why it is written down first.
