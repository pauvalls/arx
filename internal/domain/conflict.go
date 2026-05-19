package domain

// HunkRange represents a section of a file affected by a diff hunk.
type HunkRange struct {
	StartLine int
	EndLine   int
}

// Conflict represents an overlap between two fix suggestions for the same file.
type Conflict struct {
	File        string
	Suggestions [2]FixSuggestion
	Description string
}
