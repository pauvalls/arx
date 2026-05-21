package application

import (
	"context"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestDiffService_Compare_MockGitClient_GitNotInstalled(t *testing.T) {
	mock := newMockGitClient()
	mock.gitInstalled = false

	svc := NewDiffService(nil, nil, mock)
	_, err := svc.Compare(context.Background(), "/tmp", "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error when git not installed")
	}
}

func TestDiffService_Compare_MockGitClient_NotARepo(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --git-dir", "", &execError{})
	mock.withRun("rev-parse --verify HEAD~1", "abc123", nil)
	mock.withRun("rev-parse --verify HEAD", "def456", nil)

	svc := NewDiffService(nil, nil, mock)
	_, err := svc.Compare(context.Background(), "/tmp", "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error for non-git repo")
	}
}

func TestDiffService_Compare_MockGitClient_RefNotFound(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --git-dir", ".git", nil)
	mock.withRun("rev-parse --verify HEAD~1", "", &execError{})
	mock.withRun("rev-parse --verify HEAD", "", &execError{})

	svc := NewDiffService(nil, nil, mock)
	_, err := svc.Compare(context.Background(), "/tmp", "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error when ref not found")
	}
}

func TestDiffService_Compare_MockGitClient_DirtyTree(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --git-dir", ".git", nil)
	mock.withRun("rev-parse --verify HEAD~1", "abc123", nil)
	mock.withRun("rev-parse --verify HEAD", "def456", nil)
	// diff --quiet returns error when tree is dirty
	mock.withRun("diff --quiet", "", &execError{})

	svc := NewDiffService(nil, nil, mock)
	_, err := svc.Compare(context.Background(), "/tmp", "arx.yaml", "HEAD~1", "HEAD")
	if err == nil {
		t.Fatal("expected error for dirty tree")
	}
}

func TestDiffService_doGitErr_WithGitClient(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --git-dir", ".git", nil)

	svc := NewDiffService(nil, nil, mock)
	err := svc.doGitErr(context.Background(), "/tmp", "rev-parse", "--git-dir")
	if err != nil {
		t.Errorf("doGitErr() unexpected error: %v", err)
	}
}

func TestDiffService_doGitErr_WithGitClient_Error(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse BAD", "", &execError{})

	svc := NewDiffService(nil, nil, mock)
	err := svc.doGitErr(context.Background(), "/tmp", "rev-parse", "BAD")
	if err == nil {
		t.Error("doGitErr() expected error")
	}
}

func TestDiffService_doGit_WithGitClient(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --git-dir", ".git", nil)

	svc := NewDiffService(nil, nil, mock)
	// doGit should not panic or error (it discards the error)
	svc.doGit(context.Background(), "/tmp", "rev-parse", "--git-dir")
}

func TestCompareViolations_FingerprintMatch(t *testing.T) {
	v1 := domain.Violation{
		RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra",
		Import: "pkg/db", File: "service.go", Line: 42,
	}
	v2 := domain.Violation{
		RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra",
		Import: "pkg/db", File: "service_v2.go", Line: 99,
	}

	result := CompareViolations([]domain.Violation{v1}, []domain.Violation{v2})
	if len(result.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged (fingerprint match), got added=%d resolved=%d unchanged=%d",
			len(result.Added), len(result.Resolved), len(result.Unchanged))
	}
}

func TestCompareViolations_DifferentFingerprints(t *testing.T) {
	v1 := domain.Violation{
		RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra",
		Import: "pkg/db",
	}
	v2 := domain.Violation{
		RuleID: "R2", SourceLayer: "domain", TargetLayer: "infra",
		Import: "pkg/cache",
	}

	result := CompareViolations([]domain.Violation{v1}, []domain.Violation{v2})
	if len(result.Added) != 1 || len(result.Resolved) != 1 {
		t.Errorf("expected 1 added and 1 resolved, got added=%d resolved=%d",
			len(result.Added), len(result.Resolved))
	}
}
