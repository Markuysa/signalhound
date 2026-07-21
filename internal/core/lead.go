package core

import "time"

// Lead statuses, in the order they move through the mini-CRM.
const (
	LeadNew       = "new"
	LeadReviewed  = "reviewed"
	LeadContacted = "contacted"
	LeadIgnored   = "ignored"
	LeadCustomer  = "customer"
)

// Lead groups the signals of one author or company.
type Lead struct {
	ID        string
	Name      string
	Company   string
	Signals   []string // Signal IDs
	BestScore int
	Status    string // one of the Lead* constants
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidLeadStatus reports whether s is a status a Lead may hold.
func ValidLeadStatus(s string) bool {
	switch s {
	case LeadNew, LeadReviewed, LeadContacted, LeadIgnored, LeadCustomer:
		return true
	}
	return false
}
