---
name: unblock
description: Recomputes which blocked issues are now dispatchable and relabels them ready. Run after a pull request merges, since that is when a dependency's code actually reaches the default branch.
allowed-tools: Bash(gh issue *), Bash(gh pr *)
---

Move issues from `blocked` to `ready` once everything they depend on has closed.

## Why merge, and not pull-request-open

A new worktree and a cloud session both branch from the **default branch**. A ticket
unblocked while its dependency is still an open pull request would start from a tree that
does not contain that dependency, and fail for a reason that has nothing to do with its own
work. An issue closes when its pull request merges — that is the correct signal, so key off
closed issues and nothing else.

## Steps

1. List open issues labeled `blocked`:
   `gh issue list --state open --label blocked --json number,title`

2. For each one, read the body and take **only** the `## Depends on` section. Collect the
   `#N` references from that section alone — issue numbers elsewhere in the body are
   context, not dependencies. A section reading `none` means no dependencies.

3. Check the state of every referenced issue with `gh issue view <n> --json state`.

4. If **all** of them are `CLOSED`, the issue is dispatchable:
   `gh issue edit <n> --add-label ready --remove-label blocked`

   If even one is still open, leave it alone and say which one is holding it.

## Report

List what you unblocked and what is still waiting, with the blocking issue named for each.
If nothing changed, say so in one line — that is a normal outcome, not a failure.

Never relabel an issue that is already `in-progress` or `in-review`, and never close an
issue yourself. You only move `blocked` to `ready`.
