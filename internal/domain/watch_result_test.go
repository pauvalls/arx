package domain

import (
	"testing"
	"time"
)

func TestWatchResult_EmptyResult(t *testing.T) {
	r := DiffViolations(nil, nil)

	if len(r.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(r.Added))
	}
	if len(r.Resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(r.Resolved))
	}
	if len(r.Unchanged) != 0 {
		t.Errorf("expected 0 unchanged, got %d", len(r.Unchanged))
	}
	if r.HasChanges() {
		t.Error("empty result should not have changes")
	}
	if r.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestWatchResult_AddedOnly(t *testing.T) {
	prev := []Violation{}
	curr := []Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"},
	}

	r := DiffViolations(prev, curr)

	if len(r.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(r.Added))
	}
	if len(r.Resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(r.Resolved))
	}
	if len(r.Unchanged) != 0 {
		t.Errorf("expected 0 unchanged, got %d", len(r.Unchanged))
	}
	if !r.HasChanges() {
		t.Error("should have changes")
	}
}

func TestWatchResult_ResolvedOnly(t *testing.T) {
	prev := []Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"},
	}
	curr := []Violation{}

	r := DiffViolations(prev, curr)

	if len(r.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(r.Added))
	}
	if len(r.Resolved) != 1 {
		t.Errorf("expected 1 resolved, got %d", len(r.Resolved))
	}
	if len(r.Unchanged) != 0 {
		t.Errorf("expected 0 unchanged, got %d", len(r.Unchanged))
	}
	if !r.HasChanges() {
		t.Error("should have changes")
	}
}

func TestWatchResult_Mixed(t *testing.T) {
	prev := []Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"},
		{RuleID: "R2", SourceLayer: "app", TargetLayer: "domain", Import: "pkg/core"},
	}
	curr := []Violation{
		{RuleID: "R2", SourceLayer: "app", TargetLayer: "domain", Import: "pkg/core"},
		{RuleID: "R3", SourceLayer: "web", TargetLayer: "app", Import: "pkg/service"},
	}

	r := DiffViolations(prev, curr)

	if len(r.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(r.Added))
	}
	if len(r.Resolved) != 1 {
		t.Errorf("expected 1 resolved, got %d", len(r.Resolved))
	}
	if len(r.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(r.Unchanged))
	}
	if r.Added[0].RuleID != "R3" {
		t.Errorf("expected added R3, got %s", r.Added[0].RuleID)
	}
	if r.Resolved[0].RuleID != "R1" {
		t.Errorf("expected resolved R1, got %s", r.Resolved[0].RuleID)
	}
	if r.Unchanged[0].RuleID != "R2" {
		t.Errorf("expected unchanged R2, got %s", r.Unchanged[0].RuleID)
	}
}

func TestWatchResult_Summary(t *testing.T) {
	tests := []struct {
		name string
		r    WatchResult
		want string // we check prefix since timing varies
	}{
		{
			name: "no changes",
			r:    WatchResult{Timestamp: time.Now(), Elapsed: time.Second},
			want: "no changes in 1.0s",
		},
		{
			name: "added only",
			r: WatchResult{
				Added:     []Violation{{RuleID: "R1"}},
				Timestamp: time.Now(),
				Elapsed:   500 * time.Millisecond,
			},
			want: "+1 violations, -0 resolved in 500ms",
		},
		{
			name: "resolved only",
			r: WatchResult{
				Resolved:  []Violation{{RuleID: "R1"}},
				Timestamp: time.Now(),
				Elapsed:   500 * time.Millisecond,
			},
			want: "+0 violations, -1 resolved in 500ms",
		},
		{
			name: "mixed",
			r: WatchResult{
				Added:     []Violation{{RuleID: "R1"}, {RuleID: "R2"}},
				Resolved:  []Violation{{RuleID: "R3"}},
				Timestamp: time.Now(),
				Elapsed:   1200 * time.Millisecond,
			},
			want: "+2 violations, -1 resolved in 1.2s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.r.Summary()
			if summary != tt.want {
				t.Errorf("Summary() = %q, want %q", summary, tt.want)
			}
		})
	}
}

func TestWatchResult_SummaryTiming(t *testing.T) {
	r := WatchResult{
		Timestamp: time.Now(),
		Elapsed:   1234 * time.Millisecond,
	}
	summary := r.Summary()
	if summary != "no changes in 1.2s" {
		t.Errorf("Summary() = %q, want %q", summary, "no changes in 1.2s")
	}
}

func TestWatchResult_HasChanges(t *testing.T) {
	tests := []struct {
		name string
		r    WatchResult
		want bool
	}{
		{"empty", WatchResult{}, false},
		{"added only", WatchResult{Added: []Violation{{RuleID: "R1"}}}, true},
		{"resolved only", WatchResult{Resolved: []Violation{{RuleID: "R1"}}}, true},
		{"both", WatchResult{Added: []Violation{{RuleID: "R1"}}, Resolved: []Violation{{RuleID: "R2"}}}, true},
		{"unchanged only", WatchResult{Unchanged: []Violation{{RuleID: "R1"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestViolationKey(t *testing.T) {
	v1 := Violation{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"}
	v2 := Violation{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"}
	v3 := Violation{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/other"}

	k1 := ViolationKey(v1)
	k2 := ViolationKey(v2)
	k3 := ViolationKey(v3)

	if k1 != k2 {
		t.Error("same violations should produce same key")
	}
	if k1 == k3 {
		t.Error("different imports should produce different keys")
	}
}

func TestWatchResult_UnchangedWithFileChange(t *testing.T) {
	// Verify that violations with same fingerprint but different file/line are matched
	prev := []Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db", File: "old/file.go", Line: 10},
	}
	curr := []Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db", File: "new/file.go", Line: 20},
	}

	r := DiffViolations(prev, curr)

	if len(r.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged (matched by fingerprint), got %d unchanged, %d added, %d resolved",
			len(r.Unchanged), len(r.Added), len(r.Resolved))
	}
	if len(r.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(r.Added))
	}
	if len(r.Resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(r.Resolved))
	}
}
