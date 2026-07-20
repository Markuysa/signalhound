# ADR-003: One LLMProvider interface, with budget and cache as decorators

Date: 2026-07-21
Status: accepted

## Context

PRD §2.2.3 requires: multiple providers, structured JSON output with one auto-retry, per-request
cost accounting, a daily budget cap that halts scoring and alerts, and a score cache keyed on
content hash.

The obvious implementation puts budget checks and cache lookups inside each provider, or inside
the scoring stage. Both were rejected: duplicating budget logic across OpenAI and Anthropic
implementations guarantees they drift, and putting it in the scoring stage means the cache and
the cap cannot be tested without a scoring prompt in the way.

A general LLM framework (LangChain-style) was also considered and rejected — it would pull a
large dependency to solve a problem that is two interfaces wide.

## Decision

`core.LLMProvider` is a one-method interface. `internal/llm` implements it twice
(OpenAI-compatible, Anthropic) and does nothing but transport, JSON-schema enforcement, one
retry on unparseable output, and per-response cost calculation from a model price table.

`internal/llmguard` wraps any `core.LLMProvider` and returns a `core.LLMProvider`:

```
scoring → llmguard.Cache → llmguard.Budget → llm.OpenAI
```

The cache short-circuits on `content_hash + model + icp_fingerprint`. The budget layer reads
and writes the `llm_spend` ledger and returns `core.ErrBudgetExceeded` once the day's cap is
reached, which the pipeline turns into a stored-but-unscored signal plus a `budget.exceeded`
event.

## Consequences

- Budget and cache are testable against a stub provider with no prompt or network involved.
- Every provider added later inherits budget and caching for free.
- Ordering matters and is not enforced by types: cache must sit outside budget, or cached hits
  would be billed against the cap. This is documented at the composition site in `internal/app`.
- Cost accuracy depends on a hand-maintained model price table. Wrong prices mean a wrong cap,
  and nothing detects that automatically. Unknown models are priced at zero and flagged in logs.
