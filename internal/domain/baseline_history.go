package domain

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// BaselineSnapshot captures the state of violations at a point in time.
// Used for history tracking and comparing violation trends over time.
type BaselineSnapshot struct {
	Violations        []Violation        `json:"violations"`
	CreatedAt         time.Time          `json:"created_at"`
	TotalCount        int                `json:"total_count"`
	SeverityBreakdown map[Severity]int   `json:"severity_breakdown"`
}

// Fingerprint returns a hash of violation IDs for quick equality comparison.
func (s BaselineSnapshot) Fingerprint() string {
	ids := make([]string, len(s.Violations))
	for i, v := range s.Violations {
		ids[i] = v.ID
	}
	sort.Strings(ids)

	h := sha256.New()
	for _, id := range ids {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// TrendPoint represents a single data point in a trend over time.
type TrendPoint struct {
	Date     time.Time `json:"date"`
	Total    int       `json:"total"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
	Info     int       `json:"info"`
}

// BaselineTrack tracks consecutive clean checks and snapshot metadata.
type BaselineTrack struct {
	ConsecutiveClean int       `json:"consecutive_clean"`
	LastCheck        time.Time `json:"last_check"`
	LastSnapshot     time.Time `json:"last_snapshot"`
	SnapshotCount    int       `json:"snapshot_count"`
}

// NewBaselineSnapshot creates a BaselineSnapshot from violations at the current time.
func NewBaselineSnapshot(violations []Violation) BaselineSnapshot {
	if violations == nil {
		violations = []Violation{}
	}

	severityBreakdown := make(map[Severity]int)
	for _, v := range violations {
		severityBreakdown[v.Severity]++
	}

	return BaselineSnapshot{
		Violations:        violations,
		CreatedAt:         time.Now(),
		TotalCount:        len(violations),
		SeverityBreakdown: severityBreakdown,
	}
}

// TrendPointFromSnapshot creates a TrendPoint from a BaselineSnapshot.
func TrendPointFromSnapshot(snapshot BaselineSnapshot) TrendPoint {
	return TrendPoint{
		Date:     snapshot.CreatedAt,
		Total:    snapshot.TotalCount,
		Errors:   snapshot.SeverityBreakdown[SeverityError],
		Warnings: snapshot.SeverityBreakdown[SeverityWarning],
		Info:     snapshot.SeverityBreakdown[SeverityInfo],
	}
}
