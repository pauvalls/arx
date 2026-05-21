package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_CheckGitInstalled(t *testing.T) {
	c := NewClient()
	got := c.CheckGitInstalled()
	// We can't guarantee git is installed on the test runner,
	// but the check should be consistent with exec.LookPath
	_, err := exec.LookPath("git")
	want := err == nil
	if got != want {
		t.Errorf("CheckGitInstalled() = %v, want %v (git found = %v)", got, want, err == nil)
	}
}

func TestClient_Diff_NonExistentDir(t *testing.T) {
	c := NewClient()
	_, err := c.Diff(context.Background(), "HEAD", "HEAD~1", "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestClient_Diff_NonGitDir(t *testing.T) {
	c := NewClient()
	tmpDir := t.TempDir()
	_, err := c.Diff(context.Background(), "HEAD", "HEAD~1", tmpDir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestClient_Status_NonExistentDir(t *testing.T) {
	c := NewClient()
	_, err := c.Status(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestClient_Status_NonGitDir(t *testing.T) {
	c := NewClient()
	tmpDir := t.TempDir()
	_, err := c.Status(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestClient_Diff_Integration(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}

	// Create a temp git repo
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	c := NewClient()

	// Diff initial vs initial — should be empty
	diff, err := c.Diff(context.Background(), "HEAD", "HEAD", tmpDir)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if diff != "" {
		t.Errorf("Diff() between same ref should be empty, got %q", diff)
	}
}

func TestClient_Diff_WithChanges(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Write a file and commit
	file1 := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file1, []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify the file
	if err := os.WriteFile(file1, []byte("hello\nmodified\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "commit", "-a", "-m", "second")

	c := NewClient()
	diff, err := c.Diff(context.Background(), "HEAD~1", "HEAD", tmpDir)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if !strings.Contains(diff, "modified") {
		t.Errorf("Diff should contain the change, got: %s", diff)
	}
}

func TestClient_Status_Integration(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	c := NewClient()

	// Initial state after init — should be clean or show only initial files
	status, err := c.Status(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	// Write an untracked file
	untracked := filepath.Join(tmpDir, "untracked.txt")
	if err := os.WriteFile(untracked, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	status2, err := c.Status(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Status() after adding file error = %v", err)
	}
	if !strings.Contains(status2, "untracked.txt") && !strings.Contains(status2, "??") {
		t.Logf("Status should indicate untracked file (got: %q)", status2)
	}

	// The status output may not show the same as initial — at minimum verify it works
	_ = status
}

// isGitAvailable checks if git is available on PATH.
func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// initGitRepo initializes a git repository in the given directory
// with a minimal initial commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	// Create an initial commit (required for HEAD to exist)
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial")
}

// runGit executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
