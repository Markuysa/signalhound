---
id: 5
title: RSS/Atom collector
role: backend
depends: [1]
status: todo
---

Implement the RSS/Atom collector — the universal source that covers blogs, release feeds,
Google Alerts and job boards.

Spec: `docs/PRD.md` §2.2.1 (source 3).


## Likely files
- `internal/collect/rss/rss.go`
- `internal/collect/rss/rss_test.go`
- `internal/collect/rss/testdata/*.xml`

## Acceptance criteria
- [ ] Implements `core.Collector` for both RSS 2.0 and Atom; one entry becomes one
      `RawSignal` with `Source` `"rss:<configured name or feed host>"`.
- [ ] Per-feed isolation: a feed returning 500 or malformed XML does not prevent items from the
      other configured feeds being returned. The failure is reported in the returned error
      alongside the successful results.
- [ ] Entries older than `since` are filtered out. Feeds with no usable date fall back to
      first-seen tracking via `ExternalID` (guid, else link) so entries are not re-emitted on
      the next tick.
- [ ] HTML in descriptions and content is stripped to plain text before it reaches
      `RawSignal.Content`.
- [ ] Fixtures cover: valid RSS 2.0, valid Atom, entries with missing dates, malformed XML, and
      a feed that returns 500. No test touches the network (PRD §2.2.7, CLAUDE.md).
- [ ] Scheduling and rate limiting are handled by the runner (#6), not here.
- [ ] `go test ./internal/collect/rss/... -race` and `golangci-lint run` pass.
