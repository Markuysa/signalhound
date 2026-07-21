---
id: 26
title: Reddit collector: isolate per-subreddit failures
role: dev
depends: [4]
status: todo
---

Follow-up from #4 (PR #25), surfaced during validation and left out of that ticket on purpose.

`Collect` iterates subreddits sequentially and returns on the first error, discarding the
signals already gathered from the subreddits that succeeded. One subreddit that 429s or goes
private therefore costs the whole tick: the runner's cursor does not advance for any source,
and the next tick refetches everything.

Dedup means this is wasted work rather than lost data, but #6 will inherit the all-or-nothing
behaviour per tick. `docs/ARCHITECTURE.md` already specifies exactly this isolation for RSS
("one bad feed never fails the tick"); Reddit deserves the same shape.


## Acceptance criteria
- [ ] A failure on one subreddit does not discard signals already collected from others.
- [ ] The failure is still surfaced, not swallowed — the runner has to be able to emit a `collector.error`.
- [ ] An auth failure remains fatal for the whole tick: it is not per-subreddit, and retrying the rest just burns a rejected credential.
- [ ] A test covers one subreddit failing while another succeeds.
