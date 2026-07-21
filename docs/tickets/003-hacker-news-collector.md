---
id: 3
title: Hacker News collector
role: dev
depends: [1]
status: done
---

Implement the Hacker News collector against the Algolia HN Search API — the first and simplest
source, and the one that needs no credentials.

Spec: `docs/PRD.md` §2.2.1 (source 1). Design: `docs/ARCHITECTURE.md` "External integrations".


## Likely files
- `internal/collect/hn/hn.go`
- `internal/collect/hn/hn_test.go`
- `internal/collect/hn/testdata/*.json`

## Acceptance criteria
- [ ] Implements `core.Collector` against the Algolia HN Search API; requires no API key.
- [ ] Searches every keyword in `sources.hackernews.keywords` and merges the results,
      deduplicating by HN `objectID` within a single `Collect` call.
- [ ] Returns `RawSignal` with `Source` `"hackernews"`, `ExternalID` = `objectID`, a URL to the
      HN item, and `Author`, `Title`, `Content`, `PublishedAt` populated. Both stories and
      comments are supported.
- [ ] `since` is honoured through the `numericFilters` `created_at_i` bound; calling `Collect`
      with `since = now` returns zero results against the fixtures.
- [ ] Every test runs against recorded golden fixtures in `testdata/` via an injected
      `http.Client`. No test touches the network (PRD §2.2.7, CLAUDE.md).
- [ ] Non-2xx responses and malformed JSON produce a wrapped error naming the source, never a
      panic.
- [ ] Rate limiting and scheduling are **not** implemented here — they belong to the collector
      runner (#6). This package exposes `Collect` and nothing else.
- [ ] `go test ./internal/collect/hn/... -race` and `golangci-lint run` pass.
