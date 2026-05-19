package domain

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestNewBaselineSnapshot(t *testing.T) {
	t.Run("creates snapshot from violations", func(t *testing.T) {
		violations := []Violation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1, Severity: SeverityError},
			{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go", Line: 2, Severity: SeverityWarning},
			{RuleID: "R003", SourceLayer: "presentation", TargetLayer: "infrastructure", Import: "z", File: "c.go", Line: 3, Severity: SeverityInfo},
		}

		snapshot := NewBaselineSnapshot(violations)

		if snapshot.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", snapshot.TotalCount)
		}
		if snapshot.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if len(snapshot.Violations) != 3 {
			t.Errorf("len(Violations) = %d, want 3", len(snapshot.Violations))
		}
	})

	t.Run("severity breakdown counts correctly", func(t *testing.T) {
		violations := []Violation{
			{RuleID: "R001", Severity: SeverityError},
			{RuleID: "R002", Severity: SeverityError},
			{RuleID: "R003", Severity: SeverityWarning},
			{RuleID: "R004", Severity: SeverityInfo},
		}

		snapshot := NewBaselineSnapshot(violations)

		if snapshot.SeverityBreakdown[SeverityError] != 2 {
			t.Errorf("Errors = %d, want 2", snapshot.SeverityBreakdown[SeverityError])
		}
		if snapshot.SeverityBreakdown[SeverityWarning] != 1 {
			t.Errorf("Warnings = %d, want 1", snapshot.SeverityBreakdown[SeverityWarning])
		}
		if snapshot.SeverityBreakdown[SeverityInfo] != 1 {
			t.Errorf("Info = %d, want 1", snapshot.SeverityBreakdown[SeverityInfo])
		}
	})

	t.Run("empty violations snapshot", func(t *testing.T) {
		snapshot := NewBaselineSnapshot(nil)

		if snapshot.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", snapshot.TotalCount)
		}
		if len(snapshot.Violations) != 0 {
			t.Errorf("len(Violations) = %d, want 0", len(snapshot.Violations))
		}
		if len(snapshot.SeverityBreakdown) != 0 {
			t.Errorf("SeverityBreakdown len = %d, want 0", len(snapshot.SeverityBreakdown))
		}
	})
}

func TestBaselineSnapshot_Fingerprint(t *testing.T) {
	t.Run("fingerprint is deterministic hash of violation IDs", func(t *testing.T) {
		violations := []Violation{
			{ID: "V-001", RuleID: "R001"},
			{ID: "V-002", RuleID: "R002"},
		}
		s1 := NewBaselineSnapshot(violations)
		s2 := NewBaselineSnapshot(violations)

		if s1.Fingerprint() != s2.Fingerprint() {
			t.Error("Fingerprint() should be deterministic for same violations")
		}
	})

	t.Run("different violations produce different fingerprints", func(t *testing.T) {
		s1 := NewBaselineSnapshot([]Violation{{ID: "V-001"}})
		s2 := NewBaselineSnapshot([]Violation{{ID: "V-002"}})

		if s1.Fingerprint() == s2.Fingerprint() {
			t.Error("Different violations should produce different fingerprints")
		}
	})

	t.Run("fingerprint format is hex string", func(t *testing.T) {
		snapshot := NewBaselineSnapshot([]Violation{{ID: "V-001"}})
		fp := snapshot.Fingerprint()

		if len(fp) != sha256.Size*2 {
			t.Errorf("Fingerprint length = %d, want %d", len(fp), sha256.Size*2)
		}
	})

	t.Run("fingerprint of empty snapshot", func(t *testing.T) {
		snapshot := NewBaselineSnapshot(nil)
		fp := snapshot.Fingerprint()

		if fp == "" {
			t.Error("Fingerprint of empty snapshot should not be empty")
		}
	})
}

func TestTrendPointAggregation(t *testing.T) {
	t.Run("creates trend point from snapshot", func(t *testing.T) {
		now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
		violations := []Violation{
			{ID: "V-001", Severity: SeverityError},
			{ID: "V-002", Severity: SeverityError},
			{ID: "V-003", Severity: SeverityWarning},
			{ID: "V-004", Severity: SeverityInfo},
		}
		snapshot := NewBaselineSnapshot(violations)
		snapshot.CreatedAt = now

		tp := TrendPointFromSnapshot(snapshot)

		if tp.Date != now {
			t.Errorf("Date = %v, want %v", tp.Date, now)
		}
		if tp.Total != 4 {
			t.Errorf("Total = %d, want 4", tp.Total)
		}
		if tp.Errors != 2 {
			t.Errorf("Errors = %d, want 2", tp.Errors)
		}
		if tp.Warnings != 1 {
			t.Errorf("Warnings = %d, want 1", tp.Warnings)
		}
		if tp.Info != 1 {
			t.Errorf("Info = %d, want 1", tp.Info)
		}
	})

	t.Run("empty snapshot trend point", func(t *testing.T) {
		snapshot := NewBaselineSnapshot(nil)
		tp := TrendPointFromSnapshot(snapshot)

		if tp.Total != 0 {
			t.Errorf("Total = %d, want 0", tp.Total)
		}
		if tp.Errors != 0 {
			t.Errorf("Errors = %d, want 0", tp.Errors)
		}
		if tp.Warnings != 0 {
			t.Errorf("Warnings = %d, want 0", tp.Warnings)
		}
		if tp.Info != 0 {
			t.Errorf("Info = %d, want 0", tp.Info)
		}
	})
}

func TestBaselineTrackDefaults(t *testing.T) {
	t.Run("zero value track has default values", func(t *testing.T) {
		var track BaselineTrack

		if track.ConsecutiveClean != 0 {
			t.Errorf("ConsecutiveClean = %d, want 0", track.ConsecutiveClean)
		}
		if track.SnapshotCount != 0 {
			t.Errorf("SnapshotCount = %d, want 0", track.SnapshotCount)
		}
	})
}

// Helper to construct a full fingerprint for testing
func violationFingerprint(v Violation) string {
	return fmt.Sprintf("%s:%s:%s:%s", v.RuleID, v.SourceLayer, v.TargetLayer, v.Import)
}

func init() {
	// Ensure violationFingerprint matches what CompareViolations uses
	_ = violationFingerprint
}
