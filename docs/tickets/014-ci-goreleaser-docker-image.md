---
id: 14
title: CI, goreleaser, Docker image
role: dev
depends: [1]
status: todo
---

Set up the build and release pipeline: lint, test, cross-platform binaries and a multi-arch
Docker image. The UI build has to run before the Go build or the binary embeds stale assets.

Spec: `docs/PRD.md` §2.2.7, §3.4. Design: ADR-001 (the Node-before-Go ordering constraint).


## Likely files
- `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- `.goreleaser.yaml`, `Dockerfile`, `Makefile`

## Acceptance criteria
- [ ] The CI workflow runs `golangci-lint`, `go test ./... -race`, and — in `ui/` —
      `npm ci && npm run lint && npm test`. The UI steps skip cleanly (not fail) while `ui/`
      does not yet exist.
- [ ] `Makefile` provides `make ui` (npm build → `ui/dist`), `make build` (ui, then
      `go build -o bin/signalhound ./cmd/signalhound`), `make test`, `make lint`, matching the
      commands in `CLAUDE.md`.
- [ ] `make build` fails with a readable message rather than an obscure `embed` error when
      `ui/dist` is missing (ADR-001 consequence).
- [ ] `goreleaser check` passes and `goreleaser build --snapshot --clean` succeeds locally,
      producing linux/darwin/windows archives for amd64 and arm64.
- [ ] The release workflow builds a multi-arch Docker image and is triggered by a tag, not by a
      push to main.
- [ ] `Dockerfile` is multi-stage (node build → go build → minimal runtime), runs as a non-root
      user, and the final image contains the binary and nothing else.
- [ ] No secret is echoed in any workflow log; the release workflow's token permissions are
      scoped, not `write-all`.
- [ ] CI is green on the branch that adds it.

## Notes for the implementer
Do not add a workflow step that merges, tags or publishes automatically on push to main — releases
are cut by a human.
