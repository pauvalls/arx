package domain

import (
	"errors"
	"fmt"
)

// PRInfo holds the metadata for a pull request.
type PRInfo struct {
	BaseSHA   string
	HeadSHA   string
	BaseRef   string
	HeadRef   string
	RepoPath  string
	PRNumber  int
}

// Validate checks that all required PR fields are set.
func (p PRInfo) Validate() error {
	if p.BaseSHA == "" {
		return errors.New("base SHA is required")
	}
	if p.HeadSHA == "" {
		return errors.New("head SHA is required")
	}
	if p.BaseRef == "" {
		return errors.New("base ref is required")
	}
	if p.HeadRef == "" {
		return errors.New("head ref is required")
	}
	if p.PRNumber <= 0 {
		return errors.New("PR number must be positive")
	}
	return nil
}

// DiffHunk represents a single hunk in a unified diff.
type DiffHunk struct {
	File    string
	OldLine int // line number in old file (0 for new files)
	NewLine int // line number in new file (0 for deleted files)
	Content string
}

// PRDiffSummary holds the parsed diff for a PR.
type PRDiffSummary struct {
	Hunks []DiffHunk
	Stats map[string]int // files changed, insertions, deletions
}

// CheckRunConclusion represents the conclusion of a GitHub check run.
type CheckRunConclusion string

const (
	CheckRunSuccess     CheckRunConclusion = "success"
	CheckRunFailure     CheckRunConclusion = "failure"
	CheckRunNeutral     CheckRunConclusion = "neutral"
	CheckRunCancelled   CheckRunConclusion = "cancelled"
	CheckRunTimedOut    CheckRunConclusion = "timed_out"
	CheckRunActionReq   CheckRunConclusion = "action_required"
)

// Valid returns true if the conclusion is a valid GitHub check run conclusion.
func (c CheckRunConclusion) Valid() bool {
	switch c {
	case CheckRunSuccess, CheckRunFailure, CheckRunNeutral,
		CheckRunCancelled, CheckRunTimedOut, CheckRunActionReq:
		return true
	default:
		return false
	}
}

// CheckRunOutput holds the output of a check run.
type CheckRunOutput struct {
	Title       string               `json:"title"`
	Summary     string               `json:"summary"`
	Text        string               `json:"text,omitempty"`
	Annotations []CheckRunAnnotation `json:"annotations,omitempty"`
}

// CheckRunAnnotation holds a single annotation for a check run.
type CheckRunAnnotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	AnnotationLevel string `json:"annotation_level"` // notice, warning, failure
	Message         string `json:"message"`
	Title           string `json:"title,omitempty"`
}

// ValidAnnotationLevel returns true if the level is a valid GitHub annotation level.
func ValidAnnotationLevel(level string) bool {
	switch level {
	case "notice", "warning", "failure":
		return true
	default:
		return false
	}
}

// FormatPRInfo returns a human-readable summary of PR info.
func FormatPRInfo(pr PRInfo) string {
	return fmt.Sprintf("PR #%d (%s...%s)", pr.PRNumber, pr.BaseRef, pr.HeadRef)
}
