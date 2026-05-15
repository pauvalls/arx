package application

import (
	"context"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestDiffResult_HasChanges(t *testing.T) {
	tests := []struct {
		name string
		dr   DiffResult
		want bool
	}{
		{
			name: "empty diff has no changes",
			dr:   DiffResult{},
			want: false,
		},
		{
			name: "added violations means changes",
			dr: DiffResult{
				Added: []domain.Violation{{RuleID: "R001"}},
			},
			want: true,
		},
		{
			name: "resolved violations means changes",
			dr: DiffResult{
				Resolved: []domain.Violation{{RuleID: "R001"}},
			},
			want: true,
		},
		{
			name: "only unchanged means no changes",
			dr: DiffResult{
				Unchanged: []domain.Violation{{RuleID: "R001"}},
			},
			want: false,
		},
		{
			name: "mixed with added and resolved",
			dr: DiffResult{
				Added:     []domain.Violation{{RuleID: "R001"}},
				Resolved:  []domain.Violation{{RuleID: "R002"}},
				Unchanged: []domain.Violation{{RuleID: "R003"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dr.HasChanges()
			if got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiffResult_Summary(t *testing.T) {
	tests := []struct {
		name string
		dr   DiffResult
		want string
	}{
		{
			name: "empty diff",
			dr:   DiffResult{},
			want: "+0 violations, -0 resolved, 0 unchanged",
		},
		{
			name: "added only",
			dr: DiffResult{
				Added: []domain.Violation{{RuleID: "R001"}, {RuleID: "R002"}, {RuleID: "R003"}},
			},
			want: "+3 violations, -0 resolved, 0 unchanged",
		},
		{
			name: "resolved only",
			dr: DiffResult{
				Resolved: []domain.Violation{{RuleID: "R001"}},
			},
			want: "+0 violations, -1 resolved, 0 unchanged",
		},
		{
			name: "mixed",
			dr: DiffResult{
				Added:     []domain.Violation{{RuleID: "R001"}, {RuleID: "R002"}, {RuleID: "R003"}},
				Resolved:  []domain.Violation{{RuleID: "R004"}},
				Unchanged: make([]domain.Violation, 12),
			},
			want: "+3 violations, -1 resolved, 12 unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dr.Summary()
			if got != tt.want {
				t.Errorf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareViolations(t *testing.T) {
	v1 := domain.Violation{
		RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
		Import: "github.com/example/db", File: "user.go", Line: 10,
	}
	v2 := domain.Violation{
		RuleID: "R002", SourceLayer: "application", TargetLayer: "domain",
		Import: "github.com/example/entity", File: "service.go", Line: 20,
	}
	v3 := domain.Violation{
		RuleID: "R003", SourceLayer: "domain", TargetLayer: "presentation",
		Import: "github.com/example/handler", File: "handler.go", Line: 30,
	}
	v4 := domain.Violation{
		RuleID: "R004", SourceLayer: "infrastructure", TargetLayer: "domain",
		Import: "github.com/example/repo", File: "repo.go", Line: 40,
	}

	tests := []struct {
		name          string
		before        []domain.Violation
		after         []domain.Violation
		wantAdded     int
		wantResolved  int
		wantUnchanged int
	}{
		{
			name:          "both empty",
			before:        []domain.Violation{},
			after:         []domain.Violation{},
			wantAdded:     0,
			wantResolved:  0,
			wantUnchanged: 0,
		},
		{
			name:          "empty before — all added",
			before:        []domain.Violation{},
			after:         []domain.Violation{v1, v2},
			wantAdded:     2,
			wantResolved:  0,
			wantUnchanged: 0,
		},
		{
			name:          "empty after — all resolved",
			before:        []domain.Violation{v1, v2},
			after:         []domain.Violation{},
			wantAdded:     0,
			wantResolved:  2,
			wantUnchanged: 0,
		},
		{
			name:          "identical sets — all unchanged",
			before:        []domain.Violation{v1, v2},
			after:         []domain.Violation{v1, v2},
			wantAdded:     0,
			wantResolved:  0,
			wantUnchanged: 2,
		},
		{
			name:          "mixed — added, resolved, unchanged",
			before:        []domain.Violation{v1, v2, v3},
			after:         []domain.Violation{v1, v3, v4},
			wantAdded:     1, // v4
			wantResolved:  1, // v2
			wantUnchanged: 2, // v1, v3
		},
		{
			name:          "fingerprint matching ignores file and line",
			before:        []domain.Violation{v1},
			after:         []domain.Violation{{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user_v2.go", Line: 99}},
			wantAdded:     0,
			wantResolved:  0,
			wantUnchanged: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareViolations(tt.before, tt.after)

			if len(got.Added) != tt.wantAdded {
				t.Errorf("Added count = %d, want %d", len(got.Added), tt.wantAdded)
			}
			if len(got.Resolved) != tt.wantResolved {
				t.Errorf("Resolved count = %d, want %d", len(got.Resolved), tt.wantResolved)
			}
			if len(got.Unchanged) != tt.wantUnchanged {
				t.Errorf("Unchanged count = %d, want %d", len(got.Unchanged), tt.wantUnchanged)
			}
		})
	}
}

func TestDiffResult_ConfigChanged(t *testing.T) {
	dr := DiffResult{
		RefBefore:      "HEAD~1",
		RefAfter:       "HEAD",
		ConfigChanged:  true,
	}

	if !dr.ConfigChanged {
		t.Error("ConfigChanged should be true")
	}

	dr2 := DiffResult{
		RefBefore:      "HEAD~1",
		RefAfter:       "HEAD",
		ConfigChanged:  false,
	}

	if dr2.ConfigChanged {
		t.Error("ConfigChanged should be false")
	}
}

func TestDiffService_GitNotFound(t *testing.T) {
	svc := NewDiffService(nil, nil).WithGitPath("/nonexistent/git")
	_, err := svc.Compare(context.Background(), ".", "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error when git not found")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error should mention git: %v", err)
	}
}

func TestDiffService_InvalidRef(t *testing.T) {
	// Create a temp directory that is NOT a git repo
	tmpDir := t.TempDir()

	svc := NewDiffService(nil, nil)
	_, err := svc.Compare(context.Background(), tmpDir, "arx.yaml", "nonexistent-ref", "HEAD")
	if err == nil {
		t.Fatal("expected error for invalid ref in non-git directory")
	}
}

func TestDiffService_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	svc := NewDiffService(nil, nil)
	_, err := svc.Compare(context.Background(), tmpDir, "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error when directory is not a git repo")
	}
}

func TestSanitizeRef(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"HEAD", "HEAD"},
		{"HEAD~1", "HEAD~1"},
		{"feature/branch", "feature_branch"},
		{"refs/heads/main", "refs_heads_main"},
		{"v1.0.0", "v1.0.0"},
		{"../escape", "_escape"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := sanitizeRef(tt.ref)
			if got != tt.want {
				t.Errorf("sanitizeRef(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}
