---
id: 24
title: HN collector: page past Algolia's first 20 hits
role: backend
depends: [3]
status: todo
---

Follow-up from #3 (PR #23), noticed during validation and deliberately left out of that ticket.

`internal/collect/hn` reads only page 0 of the Algolia response — 20 hits per keyword per
tick. Because `search_by_date` sorts newest-first, anything past the first page is silently
dropped, and what gets dropped is the *older* end of the catch-up window: exactly the
signals a poll after downtime exists to recover.

The ceiling is marked in `hn.go` with a `ponytail:` comment.


## Acceptance criteria
- [ ] `Collect` follows Algolia's `page`/`nbPages` until the pages are exhausted or a cap is hit.
- [ ] A page cap exists so a very broad keyword cannot spin forever on one tick.
- [ ] Deduplication by `objectID` still holds across pages.
- [ ] A fixture covering a multi-page response proves the later pages are fetched and merged.
- [ ] An error on page 2 is wrapped naming the source, same as page 0 today.
