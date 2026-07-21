package core

import "errors"

// Sentinel errors live here rather than in the packages that return them, so the
// pipeline can react to a budget stop or a malformed model response without importing
// internal/llmguard or internal/scoring. See ADR-005.
var (
	// ErrBudgetExceeded is returned once the daily LLM spend cap is reached. The
	// pipeline stops scoring and stores signals unscored; it is not a failure.
	ErrBudgetExceeded = errors.New("llm daily budget exceeded")

	// ErrInvalidScoreJSON is returned when the model's reply is not valid score JSON
	// after the one permitted retry.
	ErrInvalidScoreJSON = errors.New("invalid score json")
)
