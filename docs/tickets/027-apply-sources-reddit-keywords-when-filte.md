---
id: 27
title: Apply sources.reddit.keywords when filtering signals
role: dev
depends: [4, 10]
status: todo
---

Follow-up from #4 (PR #25).

`sources.reddit.keywords` is parsed and validated in `internal/config` and documented in
PRD §2.2.1 ("настраиваемые сабреддиты + keywords"), but nothing reads it. The Reddit collector
polls whole subreddits and returns every new post; unlike the HN collector, where keywords are
the search query itself, Reddit has no equivalent server-side hook on `/new`.

The open question is *where* the filter belongs, which is why this was not guessed at in #4:
the pipeline's pre-filter stage (#10) is the natural home, but the config field lives under
the source. Decide, then implement in one place.


## Acceptance criteria
- [ ] `sources.reddit.keywords` demonstrably affects what reaches scoring.
- [ ] The decision on where the filter lives is recorded — in the pre-filter or in the collector, not both.
- [ ] An empty keyword list keeps today's behaviour: everything passes.
- [ ] Matching is case-insensitive and covers title and body.
- [ ] Tests cover a post that matches, one that does not, and the empty-list case.
