package domain

import (
	"fmt"
	"time"
)

// WatchResult represents the diff between two check runs in watch mode.
type WatchResult struct {
	Added     []Violation `json:"added"`
	Resolved  []Violation `json:"resolved"`
	Unchanged []Violation `json:"unchanged"`
	Timestamp time.Time   `json:"timestamp"`
	Elapsed   time.Duration `json:"elapsed_ms"`
}

// ViolationKey returns a stable fingerprint for matching violations across runs.
// Uses rule_id + source_layer + target_layer + import, ignoring file path and line
// number which may change during refactoring.
func ViolationKey(v Violation) string {
	return fmt.Sprintf("%s:%s:%s:%s", v.RuleID, v.SourceLayer, v.TargetLayer, v.Import)
}

// DiffViolations computes the WatchResult by comparing previous and current violation lists.
// Matching is done via ViolationKey fingerprint.
func DiffViolations(previous, current []Violation) WatchResult {
	// Build key sets
	prevKeys := make(map[string]Violation)
	for _, v := range previous {
		prevKeys[ViolationKey(v)] = v
	}
	currKeys := make(map[string]Violation)
	for _, v := range current {
		currKeys[ViolationKey(v)] = v
	}

	var added, resolved, unchanged []Violation

	// Find added and unchanged
	for _, v := range current {
		if _, exists := prevKeys[ViolationKey(v)]; exists {
			unchanged = append(unchanged, v)
		} else {
			added = append(added, v)
		}
	}

	// Find resolved
	for _, v := range previous {
		if _, exists := currKeys[ViolationKey(v)]; !exists {
			resolved = append(resolved, v)
		}
	}

	return WatchResult{
		Added:     added,
		Resolved:  resolved,
		Unchanged: unchanged,
		Timestamp: time.Now(),
	}
}

// Summary returns a human-readable string summarizing the watch result.
func (r WatchResult) Summary() string {
	if !r.HasChanges() {
		return fmt.Sprintf("no changes in %s", formatDuration(r.Elapsed))
	}

	return fmt.Sprintf("+%d violations, -%d resolved in %s", len(r.Added), len(r.Resolved), formatDuration(r.Elapsed))
}

// formatDuration formats a duration to one decimal place for display.
// e.g., 1.234s → "1.2s", 500ms → "0.5s", 50ms → "0.05s"
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	default:
		return "<1ms"
	}
}

// HasChanges returns true if there are added or resolved violations.
func (r WatchResult) HasChanges() bool {
	return len(r.Added) > 0 || len(r.Resolved) > 0
}
