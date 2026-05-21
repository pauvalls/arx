package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// client implements ports.GitClient using os/exec to run git commands.
type client struct{}

// NewClient creates a new GitClient that shells out to the git binary.
func NewClient() *client {
	return &client{}
}

// Diff returns the unified diff between two refs using `git diff`.
func (c *client) Diff(ctx context.Context, baseRef, headRef, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--unified=0", baseRef, headRef)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(output), nil
}

// Status returns the working tree status in porcelain format.
func (c *client) Status(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Run executes an arbitrary git command and returns its output.
func (c *client) Run(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v failed: %w", args, err)
	}
	return string(output), nil
}

// CheckGitInstalled checks if git is available on the system PATH.
func (c *client) CheckGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
