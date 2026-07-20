# ADR-005: Single composition root in internal/app; feature packages depend only on core

Date: 2026-07-21
Status: accepted

## Context

The MVP is being built by parallel agents working in separate git worktrees. Two agents editing
the same file is the failure mode that setup exists to prevent. In a greenfield Go service the
natural magnet for that conflict is wiring code — `main.go`, where every new component wants to
register itself.

Options: let each feature ticket wire itself into `main.go` (guaranteed conflicts); use a DI
container like wire or fx (generated or reflective wiring, still one file, plus a build step);
or concentrate all wiring in one package owned by exactly one ticket.

## Decision

`internal/core` holds every cross-package interface and imports nothing internal. Feature
packages (`collect/*`, `llm`, `llmguard`, `scoring`, `pipeline`, `notify`, `api`, `store`,
`demo`) depend on `internal/core`, and on `internal/store` where they persist. **They never
import each other.**

`internal/app` is the only package that imports concrete implementations. It reads config,
constructs everything, owns startup and graceful shutdown. `cmd/signalhound` is a thin flag
parser over it.

No DI framework. Wiring is a hundred lines of explicit constructor calls, which is cheaper to
read than generated code.

## Consequences

- Ticket boundaries fall out of package boundaries: each feature ticket creates one directory
  and touches nothing else, so tickets genuinely run in parallel.
- The cost is one serialization point — the wiring ticket depends on every feature ticket and
  cannot start early. This is accepted deliberately: one late ticket beats N merge conflicts.
- An interface that turns out wrong is expensive, because it is discovered at wiring time when
  every implementation already exists. The mitigation is that `core` interfaces are tiny and
  copied verbatim from the PRD.
- `internal/core` must stay dependency-free. A feature type leaking into it (an HTTP client, a
  config struct) collapses the whole scheme, and nothing enforces this but review.
