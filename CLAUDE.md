# SignalHound

Self-hosted intent-signal monitoring agent. Collects from HN, Reddit, RSS, GitHub and job
boards, scores signals against a user-defined ICP with an LLM, and delivers qualified leads
to Telegram/Slack. Open source, single binary, no outreach automation.

## Stack

- **Agent, API, embedded UI host:** Go 1.22+, single binary, UI served from `embed.FS`
- **UI:** React + Vite + TypeScript + Tailwind, built to static assets, embedded into the binary
- **Store:** SQLite by default, Postgres optional. Migrations via goose (embedded)
- **LLM:** provider interface — OpenAI-compatible (covers OpenAI/OpenRouter/vLLM/Ollama) and Anthropic
- **License:** AGPL-3.0

## Commands

Backend (repo root):
- dev: `go run ./cmd/signalhound serve`
- test: `go test ./...`
- lint: `golangci-lint run`
- build: `go build -o bin/signalhound ./cmd/signalhound`

UI (`ui/`):
- dev: `npm run dev`
- test: `npm test`
- lint: `npm run lint`
- build: `npm run build`

## Rules

- Non-trivial changes go through a plan first: propose, then write code.
- Tests are mandatory for business logic. Pipeline and scoring are tested against a fake
  LLM provider; collectors against recorded fixtures (golden files). Never hit a live
  external API in a test.
- **Design tokens come only from `docs/DESIGN.md` §2.** Hardcoded colors or spacing in the
  UI are a defect. The anti-patterns listed there (AI gradients, acid-green terminal,
  emoji instead of SVG icons, light theme as default) are prohibited, not discouraged.
- **No outreach automation, ever.** The agent finds and drafts; a human sends. This is a
  product principle, not a scope decision — do not add sending, even behind a flag.
- Every LLM score must carry its explanation (`reasons`, `icp_match`). A score without
  reasoning is incomplete.
- Never commit real API keys, `config.yaml` with live credentials, or `.env`.
- Do not add LinkedIn or X scraping collectors — deliberate ToS decision, see `docs/PRD.md` §2.2.1.

## References (read on demand, do not hold in context)
- Spec: docs/PRD.md
- Design system and tokens: docs/DESIGN.md
- Architecture: docs/ARCHITECTURE.md
- Cheat sheet: docs/QUICK_REF.md
- Decisions (ADR): docs/decisions/
- UI mockups: signalhound-ui/*.html, signalhound-design-concept.html

## Compaction policy
When compacting, preserve: the full list of changed files, test commands, and decisions
with their reasoning. Condense research findings aggressively.
