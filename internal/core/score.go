package core

import "time"

// Intent types a Scorer may assign.
const (
	IntentBuying   = "buying"
	IntentPain     = "pain"
	IntentHiring   = "hiring"
	IntentResearch = "research"
	IntentNone     = "none"
)

// Score is the LLM's judgement of one Signal against the configured ICP.
//
// Reasons is mandatory and never empty: an unexplained score is useless for tuning the
// ICP prompt, which is the main thing a user does with this product. See PRD §2.2.4.
type Score struct {
	SignalID   string
	Value      int      // 0-100
	IntentType string   // one of the Intent* constants
	Reasons    []string // mandatory, never empty
	ICPMatch   []string
	DraftReply string // prepared for a human to send; never sent automatically (ADR-006)
	Model      string
	CostUSD    float64
	LatencyMS  int
	ScoredAt   time.Time
}

// ValidIntentType reports whether s is one of the intent types the model may return.
func ValidIntentType(s string) bool {
	switch s {
	case IntentBuying, IntentPain, IntentHiring, IntentResearch, IntentNone:
		return true
	}
	return false
}
