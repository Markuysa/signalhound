---
name: next-ticket
description: Picks the next dispatchable issue and implements it, without anyone naming which one. Serializes on the in-progress label so concurrent runs cannot claim the same ticket.
argument-hint: "[max tickets in flight, default 1]"
allowed-tools: Bash(gh issue *), Read, Write, Edit, Grep, Glob, Bash, Agent
---

Max in flight: $ARGUMENTS (default **1** — strictly one ticket at a time).

## 1. Check whether there is room

```
gh issue list --state open --label in-progress --json number,title
```

If the count is at or above the limit, **stop**. Report which tickets are in flight and
exit. This check is what keeps two runs from grabbing the same issue: without it, both read
the same "next ready" before either can claim it.

Raising the limit above 1 is a deliberate choice. Parallel work is bounded by your review
throughput and by plan usage limits, not by how many issues are open.

## 2. Pick the next ticket

```
gh issue list --state open --label ready --json number,title,labels
```

Discard anything also labeled `in-progress`, `in-review`, or `needs-attention` —
`needs-attention` means a human has to look before it is retried.

Take the **lowest issue number** that survives. Issue numbers follow the order the planner
created them, which is dependency order, so lowest-first is the intended sequence.

## 3. If nothing is ready

Stop, and say which of these it is — the distinction tells the human what to do next:

- **Blocked work exists.** Name the blocked issues and the open issues holding them. If some
  dependency is closed and the label just was not refreshed, say the `unblock` skill should run.
- **Everything is in review.** The queue is waiting on merges, which is the human's move.
- **Nothing left.** All issues are closed. Say so plainly.

Do not invent work, and do not pick a `blocked` issue because it looks ready to you. The
labels are the queue.

## 4. Run it

Invoke the `execute-ticket` skill for the chosen issue number. It claims the issue, does the
work, and opens a pull request. Everything about how the ticket is implemented lives there —
do not restate or second-guess it here.

## Stop conditions

Handle exactly one ticket per run. Do not loop back to step 1 for another. The chain is
advanced by the routine firing again after a merge, so that a human review sits between
every ticket — which is the whole point of the gate.
