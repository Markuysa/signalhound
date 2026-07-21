---
id: 4
title: Reddit collector
role: dev
depends: [1]
status: todo
---

Implement the Reddit collector using the official API with app-only OAuth.

Spec: `docs/PRD.md` §2.2.1 (source 2). Design: `docs/ARCHITECTURE.md` "External integrations".
ADR-006 constrains the OAuth scope.


## Likely files
- `internal/collect/reddit/reddit.go`, `internal/collect/reddit/auth.go`
- `internal/collect/reddit/reddit_test.go`
- `internal/collect/reddit/testdata/*.json`

## Acceptance criteria
- [ ] Implements `core.Collector` using Reddit OAuth2 client-credentials (app-only). **Read
      scope only — requesting any write scope is forbidden (ADR-006).**
- [ ] Client ID and secret are read from the environment, never from `config.yaml` (ADR-007).
- [ ] Polls each subreddit listed in `sources.reddit.subreddits`; `Source` is
      `"reddit:r/<name>"`.
- [ ] The access token is cached and refreshed on a 401. A refresh failure returns a typed auth
      error whose message tells the user what to do, in the style of `docs/DESIGN.md` §5
      ("Reddit collector: auth expired → reconnect in Sources").
- [ ] `since` is honoured via `created_utc`; pagination stops as soon as entries are older than
      the cursor.
- [ ] Sets a descriptive `User-Agent` as required by the Reddit API rules.
- [ ] Every test uses recorded fixtures and an injected `http.Client`; no network access, no
      live credentials (PRD §2.2.7, CLAUDE.md). Fixtures cover: successful listing, token
      refresh after 401, refresh failure, and rate-limit 429.
- [ ] Scheduling and rate limiting are handled by the runner (#6), not here.
- [ ] `go test ./internal/collect/reddit/... -race` and `golangci-lint run` pass.
