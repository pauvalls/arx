package domain

import (
	"testing"
	"time"
)

func TestBaselineViolation_Fingerprint(t *testing.T) {
	tests := []struct {
		name string
		bv   BaselineViolation
		want string
	}{
		{
			name: "standard fingerprint",
			bv: BaselineViolation{
				RuleID:      "R001",
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/db",
				File:        "internal/domain/user.go",
			},
			want: "R001:domain:infrastructure:github.com/example/db",
		},
		{
			name: "fingerprint ignores file path",
			bv: BaselineViolation{
				RuleID:      "R001",
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/db",
				File:        "internal/domain/other.go",
			},
			want: "R001:domain:infrastructure:github.com/example/db",
		},
		{
			name: "different rule produces different fingerprint",
			bv: BaselineViolation{
				RuleID:      "R002",
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/db",
				File:        "internal/domain/user.go",
			},
			want: "R002:domain:infrastructure:github.com/example/db",
		},
		{
			name: "different import produces different fingerprint",
			bv: BaselineViolation{
				RuleID:      "R001",
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/cache",
				File:        "internal/domain/user.go",
			},
			want: "R001:domain:infrastructure:github.com/example/cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.bv.Fingerprint()
			if got != tt.want {
				t.Errorf("Fingerprint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBaseline_IsSuppressed(t *testing.T) {
	baseline := &Baseline{
		Version:    "1.0",
		ConfigHash: "abc123",
		GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "internal/domain/user.go",
			},
			{
				RuleID: "R002", SourceLayer: "application", TargetLayer: "domain",
				Import: "github.com/example/entity", File: "internal/app/service.go",
			},
		},
	}

	tests := []struct {
		name     string
		violation Violation
		want     bool
	}{
		{
			name: "matching fingerprint is suppressed",
			violation: Violation{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "internal/domain/user.go", Line: 10,
			},
			want: true,
		},
		{
			name: "matching fingerprint with different file path is still suppressed",
			violation: Violation{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "internal/domain/user_v2.go", Line: 15,
			},
			want: true,
		},
		{
			name: "matching fingerprint with different line is still suppressed",
			violation: Violation{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "internal/domain/user.go", Line: 99,
			},
			want: true,
		},
		{
			name: "different rule is not suppressed",
			violation: Violation{
				RuleID: "R003", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "internal/domain/user.go", Line: 10,
			},
			want: false,
		},
		{
			name: "different import is not suppressed",
			violation: Violation{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/other", File: "internal/domain/user.go", Line: 10,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseline.IsSuppressed(tt.violation)
			if got != tt.want {
				t.Errorf("IsSuppressed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseline_IsSuppressed_Empty(t *testing.T) {
	empty := &Baseline{
		Version:    "1.0",
		ConfigHash: "abc123",
		GeneratedAt: time.Now(),
		Violations:  []BaselineViolation{},
	}

	v := Violation{
		RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
		Import: "github.com/example/db", File: "internal/domain/user.go", Line: 10,
	}

	if empty.IsSuppressed(v) {
		t.Error("empty baseline should not suppress any violation")
	}
}

func TestBaseline_IsSuppressed_NilBaseline(t *testing.T) {
	var b *Baseline
	v := Violation{RuleID: "R001"}

	if b.IsSuppressed(v) {
		t.Error("nil baseline should not suppress any violation")
	}
}

func TestBaseline_Filter(t *testing.T) {
	baseline := &Baseline{
		Version:    "1.0",
		ConfigHash: "abc123",
		GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "old.go",
			},
		},
	}

	violations := []Violation{
		{
			RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
			Import: "github.com/example/db", File: "user.go", Line: 10,
		},
		{
			RuleID: "R002", SourceLayer: "application", TargetLayer: "domain",
			Import: "github.com/example/entity", File: "service.go", Line: 20,
		},
		{
			RuleID: "R003", SourceLayer: "domain", TargetLayer: "presentation",
			Import: "github.com/example/handler", File: "handler.go", Line: 30,
		},
	}

	filtered := baseline.Filter(violations)

	if len(filtered) != 2 {
		t.Errorf("Filter() returned %d violations, want 2", len(filtered))
	}

	// R001 should be suppressed, R002 and R003 should remain
	for _, v := range filtered {
		if v.RuleID == "R001" {
			t.Error("Filter() should have suppressed R001")
		}
	}
}

func TestBaseline_Filter_EmptyBaseline(t *testing.T) {
	empty := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{},
	}

	violations := []Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R002", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "y", File: "b.go", Line: 2},
	}

	filtered := empty.Filter(violations)

	if len(filtered) != 2 {
		t.Errorf("Filter() with empty baseline returned %d violations, want 2", len(filtered))
	}
}

func TestBaseline_Filter_NilBaseline(t *testing.T) {
	var b *Baseline
	violations := []Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
	}

	filtered := b.Filter(violations)

	if len(filtered) != 1 {
		t.Errorf("Filter() with nil baseline returned %d violations, want 1", len(filtered))
	}
}

func TestBaseline_Filter_EmptyViolations(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
		},
	}

	filtered := baseline.Filter([]Violation{})

	if len(filtered) != 0 {
		t.Errorf("Filter() with empty violations returned %d, want 0", len(filtered))
	}
}

func TestBaseline_IsStale(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "old-hash", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{},
	}

	tests := []struct {
		name      string
		configHash string
		want      bool
	}{
		{"same hash, not stale", "old-hash", false},
		{"different hash, stale", "new-hash", true},
		{"empty hash, stale", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseline.IsStale(tt.configHash)
			if got != tt.want {
				t.Errorf("IsStale(%q) = %v, want %v", tt.configHash, got, tt.want)
			}
		})
	}
}

func TestGenerateBaseline(t *testing.T) {
	violations := []Violation{
		{
			RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
			Import: "github.com/example/db", File: "user.go", Line: 10,
			Message: "domain cannot depend on infrastructure",
		},
		{
			RuleID: "R002", SourceLayer: "application", TargetLayer: "domain",
			Import: "github.com/example/entity", File: "service.go", Line: 20,
			Message: "application cannot depend on domain",
		},
	}

	configHash := "test-hash-123"

	baseline := GenerateBaseline(violations, configHash)

	if baseline == nil {
		t.Fatal("GenerateBaseline() returned nil")
	}

	if baseline.Version != "1.0" {
		t.Errorf("Version = %q, want %q", baseline.Version, "1.0")
	}

	if baseline.ConfigHash != configHash {
		t.Errorf("ConfigHash = %q, want %q", baseline.ConfigHash, configHash)
	}

	if baseline.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should not be zero")
	}

	if len(baseline.Violations) != 2 {
		t.Errorf("Violations count = %d, want 2", len(baseline.Violations))
	}

	// Verify first violation
	bv := baseline.Violations[0]
	if bv.RuleID != "R001" {
		t.Errorf("Violation[0].RuleID = %q, want %q", bv.RuleID, "R001")
	}
	if bv.SourceLayer != "domain" {
		t.Errorf("Violation[0].SourceLayer = %q, want %q", bv.SourceLayer, "domain")
	}
	if bv.TargetLayer != "infrastructure" {
		t.Errorf("Violation[0].TargetLayer = %q, want %q", bv.TargetLayer, "infrastructure")
	}
	if bv.Import != "github.com/example/db" {
		t.Errorf("Violation[0].Import = %q, want %q", bv.Import, "github.com/example/db")
	}
	if bv.File != "user.go" {
		t.Errorf("Violation[0].File = %q, want %q", bv.File, "user.go")
	}
}

func TestGenerateBaseline_EmptyViolations(t *testing.T) {
	baseline := GenerateBaseline([]Violation{}, "hash-123")

	if baseline == nil {
		t.Fatal("GenerateBaseline() returned nil for empty violations")
	}

	if len(baseline.Violations) != 0 {
		t.Errorf("Violations count = %d, want 0", len(baseline.Violations))
	}
}

func TestBaseline_IsSuppressed_DuplicateViolations(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "a.go"},
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "b.go"},
		},
	}

	v := Violation{
		RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
		Import: "github.com/example/db", File: "c.go", Line: 10,
	}

	// Should suppress even with duplicates in baseline
	if !baseline.IsSuppressed(v) {
		t.Error("IsSuppressed() should return true when duplicate entries exist in baseline")
	}
}

func TestBaseline_IsSuppressed_VeryLongImportPath(t *testing.T) {
	longImport := "github.com/example/very/long/import/path/that/keeps/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going/and/going"

	baseline := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: longImport, File: "a.go"},
		},
	}

	v := Violation{
		RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
		Import: longImport, File: "b.go", Line: 10,
	}

	if !baseline.IsSuppressed(v) {
		t.Error("IsSuppressed() should handle very long import paths")
	}
}

func TestBaseline_Filter_AllSuppressed(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
			{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go"},
		},
	}

	violations := []Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go", Line: 2},
	}

	filtered := baseline.Filter(violations)

	if len(filtered) != 0 {
		t.Errorf("Filter() with all suppressed returned %d violations, want 0", len(filtered))
	}
}

func TestBaseline_Filter_NoneSuppressed(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "abc", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
		},
	}

	violations := []Violation{
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go", Line: 1},
		{RuleID: "R003", SourceLayer: "domain", TargetLayer: "presentation", Import: "z", File: "c.go", Line: 2},
	}

	filtered := baseline.Filter(violations)

	if len(filtered) != 2 {
		t.Errorf("Filter() with none suppressed returned %d violations, want 2", len(filtered))
	}
}

func TestBaseline_IsStale_NilBaseline(t *testing.T) {
	var b *Baseline

	if !b.IsStale("any-hash") {
		t.Error("IsStale() on nil baseline should return true")
	}
}

func TestBaseline_IsStale_EmptyHash(t *testing.T) {
	baseline := &Baseline{
		Version: "1.0", ConfigHash: "", GeneratedAt: time.Now(),
		Violations: []BaselineViolation{},
	}

	if !baseline.IsStale("new-hash") {
		t.Error("IsStale() should return true when baseline has empty hash and current hash is non-empty")
	}

	if baseline.IsStale("") {
		t.Error("IsStale() should return false when both hashes are empty")
	}
}

func TestBaseline_Fingerprint_SpecialCharacters(t *testing.T) {
	bv := BaselineViolation{
		RuleID:      "R-001_2",
		SourceLayer: "my-domain",
		TargetLayer: "infra_v2",
		Import:      "github.com/example/pkg:sub/v1.0.0",
		File:        "internal/domain/user.go",
	}

	got := bv.Fingerprint()
	want := "R-001_2:my-domain:infra_v2:github.com/example/pkg:sub/v1.0.0"

	if got != want {
		t.Errorf("Fingerprint() = %q, want %q", got, want)
	}
}

func TestGenerateBaseline_DuplicateViolations(t *testing.T) {
	violations := []Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "b.go", Line: 2},
	}

	baseline := GenerateBaseline(violations, "hash-123")

	if len(baseline.Violations) != 2 {
		t.Errorf("Violations count = %d, want 2 (duplicates preserved)", len(baseline.Violations))
	}
}
