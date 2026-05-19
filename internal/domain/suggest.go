package domain

// FixSuggestion represents a single suggested code fix at the domain level.
type FixSuggestion struct {
	ViolationID string
	RuleID      string
	File        string
	Line        int
	Description string
	Diff        string
	HunkRange   HunkRange
}
