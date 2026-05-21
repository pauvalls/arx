package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// DiffResult holds the comparison between two architecture audit results.
type DiffResult struct {
	Added         []domain.Violation `json:"added"`
	Resolved      []domain.Violation `json:"resolved"`
	Unchanged     []domain.Violation `json:"unchanged"`
	RefBefore     string             `json:"ref_before"`
	RefAfter      string             `json:"ref_after"`
	ConfigChanged bool               `json:"config_changed"`
}

// HasChanges returns true if there are added or resolved violations.
func (d DiffResult) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Resolved) > 0
}

// Summary returns a human-readable summary string.
// Example: "+3 violations, -1 resolved, 12 unchanged"
func (d DiffResult) Summary() string {
	return fmt.Sprintf("+%d violations, -%d resolved, %d unchanged",
		len(d.Added), len(d.Resolved), len(d.Unchanged))
}

// violationFingerprint returns a stable fingerprint for matching violations.
// Uses rule_id + source_layer + target_layer + import — same as baseline fingerprinting.
func violationFingerprint(v domain.Violation) string {
	return fmt.Sprintf("%s:%s:%s:%s", v.RuleID, v.SourceLayer, v.TargetLayer, v.Import)
}

// CompareViolations compares two sets of violations using fingerprint matching.
// Returns a DiffResult classifying violations as Added, Resolved, or Unchanged.
func CompareViolations(before, after []domain.Violation) DiffResult {
	// Build fingerprint sets for O(1) lookup
	beforeSet := make(map[string]int) // fingerprint → index in before slice
	for i, v := range before {
		beforeSet[violationFingerprint(v)] = i
	}

	afterSet := make(map[string]int)
	for i, v := range after {
		afterSet[violationFingerprint(v)] = i
	}

	var added, resolved, unchanged []domain.Violation

	// Find added and unchanged (iterate after set)
	for fp, idx := range afterSet {
		if _, exists := beforeSet[fp]; exists {
			unchanged = append(unchanged, after[idx])
		} else {
			added = append(added, after[idx])
		}
	}

	// Find resolved (in before but not in after)
	for fp, idx := range beforeSet {
		if _, exists := afterSet[fp]; !exists {
			resolved = append(resolved, before[idx])
		}
	}

	return DiffResult{
		Added:     added,
		Resolved:  resolved,
		Unchanged: unchanged,
	}
}

// DiffService runs architecture audits on two git refs and compares results.
// Uses git worktree for isolated checkouts.
type DiffService struct {
	gitPath     func() string // returns path to git binary (fallback when gitClient not used)
	auditSvc    *AuditService
	baselineSvc *BaselineService
	gitClient   ports.GitClient
}

// NewDiffService creates a DiffService with the given audit and baseline services.
// gitClient is optional (nil is accepted, falls back to direct exec calls).
func NewDiffService(auditSvc *AuditService, baselineSvc *BaselineService, gitClient ...ports.GitClient) *DiffService {
	s := &DiffService{
		gitPath:     func() string { return "git" },
		auditSvc:    auditSvc,
		baselineSvc: baselineSvc,
	}
	if len(gitClient) > 0 {
		s.gitClient = gitClient[0]
	}
	return s
}

// WithGitPath sets a custom git binary path (for testing).
func (s *DiffService) WithGitPath(gitPath string) *DiffService {
	s.gitPath = func() string { return gitPath }
	return s
}

// Compare runs audits on two git refs and returns a DiffResult.
// Uses git worktree for isolated checkouts at .arx-diff/{ref}/.
func (s *DiffService) Compare(ctx context.Context, projectRoot, configPath, refBefore, refAfter string) (*DiffResult, error) {
	gitBin := s.gitPath()

	// Verify git is available
	if s.gitClient != nil {
		if !s.gitClient.CheckGitInstalled() {
			return nil, fmt.Errorf("git not found on PATH\nInstall git or use 'arx check' on each ref manually")
		}
	} else {
		if _, err := exec.LookPath(gitBin); err != nil {
			return nil, fmt.Errorf("git not found on PATH: %w\nInstall git or use 'arx check' on each ref manually", err)
		}
	}

	// Verify project root is a git repository
	var gitOpsErr error
	if s.gitClient != nil {
		_, gitOpsErr = s.gitClient.Run(ctx, projectRoot, "rev-parse", "--git-dir")
	} else {
		gitOpsErr = s.runGit(ctx, projectRoot, "rev-parse", "--git-dir")
	}
	if gitOpsErr != nil {
		return nil, fmt.Errorf("%s is not a git repository: %w", projectRoot, gitOpsErr)
	}

	// Verify refs exist
	for _, ref := range []string{refBefore, refAfter} {
		var err error
		if s.gitClient != nil {
			_, err = s.gitClient.Run(ctx, projectRoot, "rev-parse", "--verify", ref)
		} else {
			err = s.runGit(ctx, projectRoot, "rev-parse", "--verify", ref)
		}
		if err != nil {
			return nil, fmt.Errorf("ref %q does not exist: %w", ref, err)
		}
	}

	// Check for dirty working tree
	var dirtyErr error
	if s.gitClient != nil {
		_, dirtyErr = s.gitClient.Run(ctx, projectRoot, "diff", "--quiet")
	} else {
		dirtyErr = s.runGit(ctx, projectRoot, "diff", "--quiet")
	}
	if dirtyErr != nil {
		return nil, fmt.Errorf("working tree has uncommitted changes. Commit or stash changes before running 'arx diff'")
	}

	// Create worktree base directory
	worktreeBase := filepath.Join(projectRoot, ".arx-diff")
	if err := os.MkdirAll(worktreeBase, 0o755); err != nil {
		return nil, fmt.Errorf("creating worktree directory: %w", err)
	}

	beforePath := filepath.Join(worktreeBase, sanitizeRef(refBefore))
	afterPath := filepath.Join(worktreeBase, sanitizeRef(refAfter))

	// Ensure cleanup on exit
	defer func() {
		s.doGit(ctx, projectRoot, "worktree", "remove", "--force", beforePath)
		s.doGit(ctx, projectRoot, "worktree", "remove", "--force", afterPath)
		// Best effort cleanup of the base directory
		os.RemoveAll(worktreeBase)
	}()

	// Create worktrees
	if err := s.doGitErr(ctx, projectRoot, "worktree", "add", "--detach", beforePath, refBefore); err != nil {
		return nil, fmt.Errorf("creating worktree for %q: %w", refBefore, err)
	}
	if err := s.doGitErr(ctx, projectRoot, "worktree", "add", "--detach", afterPath, refAfter); err != nil {
		return nil, fmt.Errorf("creating worktree for %q: %w", refAfter, err)
	}

	// Run audits on each worktree
	reportBefore, err := s.auditSvc.Audit(ctx, beforePath, filepath.Join(beforePath, configPath))
	if err != nil {
		return nil, fmt.Errorf("auditing %q: %w", refBefore, err)
	}

	reportAfter, err := s.auditSvc.Audit(ctx, afterPath, filepath.Join(afterPath, configPath))
	if err != nil {
		return nil, fmt.Errorf("auditing %q: %w", refAfter, err)
	}

	// Compare results
	result := CompareViolations(reportBefore.Violations, reportAfter.Violations)
	result.RefBefore = refBefore
	result.RefAfter = refAfter
	result.ConfigChanged = reportBefore.ConfigHash != reportAfter.ConfigHash

	return &result, nil
}

// doGit executes a git command in the specified directory, discarding output and errors.
// Used for best-effort cleanup operations.
func (s *DiffService) doGit(ctx context.Context, dir string, args ...string) {
	_ = s.doGitErr(ctx, dir, args...)
}

// doGitErr executes a git command using the gitClient when available, falling back
// to os/exec for commands not covered by the interface.
func (s *DiffService) doGitErr(ctx context.Context, dir string, args ...string) error {
	if s.gitClient != nil {
		_, err := s.gitClient.Run(ctx, dir, args...)
		return err
	}
	// Fallback to direct exec
	cmd := exec.CommandContext(ctx, s.gitPath(), args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// runGit executes a git command in the specified directory.
func (s *DiffService) runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, s.gitPath(), args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// sanitizeRef converts a git ref to a safe directory name.
func sanitizeRef(ref string) string {
	// Prevent path traversal
	if strings.HasPrefix(ref, "..") {
		ref = strings.TrimPrefix(ref, "..")
	}
	// Replace path separators with underscores
	result := strings.ReplaceAll(ref, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	return result
}
