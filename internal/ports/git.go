package ports

import "context"

// GitClient defines the interface for git operations used by arx.
// Extracting this interface enables unit testing of git-dependent code paths
// (diff, doctor, pr_check) without requiring an actual git repository.
type GitClient interface {
	// Diff returns the unified diff between two refs.
	Diff(ctx context.Context, baseRef, headRef, repoPath string) (string, error)

	// Status returns the working tree status (porcelain format).
	Status(ctx context.Context, repoPath string) (string, error)

	// Run executes an arbitrary git command and returns its output.
	Run(ctx context.Context, repoPath string, args ...string) (string, error)

	// CheckGitInstalled checks if git is available on the system PATH.
	CheckGitInstalled() bool
}
