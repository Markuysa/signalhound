---
id: 7
title: LLM providers: OpenAI-compatible and Anthropic
role: backend
depends: [1]
status: todo
---

Implement the two LLM providers behind `core.LLMProvider`: transport, structured output with one
retry, and per-request cost accounting. Nothing else — the budget cap and the score cache are
decorators in a separate ticket (#8).

Spec: `docs/PRD.md` §2.2.3. Design: ADR-003.


## Likely files
- `internal/llm/{provider,openai,anthropic,pricing}.go`
- `internal/llm/provider_test.go`
- `internal/llm/testdata/*.json`

## Acceptance criteria
- [ ] Two implementations of `core.LLMProvider`: OpenAI-compatible (with a configurable
      `base_url`, so OpenAI, OpenRouter, vLLM and Ollama are all covered) and Anthropic.
- [ ] Structured output: the caller supplies a JSON schema. The provider requests native
      structured output where the API supports it, and validates the response against the schema
      in every case.
- [ ] Exactly one automatic retry when a response fails schema validation. A second failure
      returns `core.ErrInvalidScoreJSON` with the raw response body attached (PRD §2.2.3).
- [ ] `CompletionResponse` carries prompt tokens, completion tokens, `CostUSD` computed from a
      model price table, and latency.
- [ ] An unknown model is priced at 0 and logs a warning naming the model, rather than failing
      the request (ADR-003).
- [ ] API keys are read from the environment only, never from `config.yaml` (ADR-007).
- [ ] Tests use recorded fixtures and an injected `http.Client`, covering: happy path, one
      invalid JSON then valid, two invalid JSON, HTTP 429, HTTP 500, and cost calculation for
      both providers. No test touches a live API (CLAUDE.md).
- [ ] This package does not read or write the store and does not know what a budget is.
- [ ] `go test ./internal/llm/... -race` and `golangci-lint run` pass.
