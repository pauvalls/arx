package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// hookScript is the POSIX shell script installed as the git pre-commit hook.
//
// Flow:
//  1. If SKIP=arx env var is set, exit 0 immediately (bypass).
//  2. Detect project root via git rev-parse.
//  3. Run "arx check --no-cache" which:
//     - Loads .arx-baseline.json automatically if present
//     - Exits 0 when only baselined (suppressed) violations exist
//     - Exits 1 when new violations are found
//  4. On non-zero: print guidance and exit with arx's exit code.
//  5. On zero: exit 0 (commit proceeds).
const hookScript = `#!/bin/sh
# arx pre-commit hook — blocks commits with new architecture violations
# Set SKIP=arx to bypass this hook (e.g. SKIP=arx git commit)

if echo "${SKIP-}" | grep -q arx; then
    exit 0
fi

PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "not a git repository" >&2
    exit 1
}

cd "$PROJECT_ROOT" || exit 1
arx check --no-cache
exit_code=$?

if [ $exit_code -ne 0 ]; then
    echo "Architecture violation(s) found. Run 'arx check' for details." >&2
fi

exit $exit_code
`

// hookCmd is the parent for hook sub-commands.
var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage git pre-commit hooks for architecture validation",
	Long: `Manage git pre-commit hooks for architecture validation.

Install a pre-commit hook that runs 'arx check --no-cache' before each commit.
The hook respects the baseline (.arx-baseline.json) and only blocks commits
that introduce NEW violations. Pre-existing (baselined) violations are allowed.

Subcommands:
  install   Install the pre-commit hook
  uninstall Remove the pre-commit hook`,
}

// hookInstallCmd installs the pre-commit hook.
var hookInstallCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install the pre-commit hook",
	Long: `Install the pre-commit hook in the git repository.

Creates .git/hooks/pre-commit as an executable POSIX shell script that runs
'arx check --no-cache' before each commit.

The hook automatically respects .arx-baseline.json — only NEW violations
block the commit. Pre-existing (baselined) violations are allowed.

Bypass the hook for a single commit by setting SKIP=arx:
  SKIP=arx git commit -m "..."

If no path is provided, the current directory is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHookInstall,
}

// hookUninstallCmd removes the pre-commit hook.
var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall [path]",
	Short: "Uninstall the pre-commit hook",
	Long: `Remove the pre-commit hook from the git repository.

Removes .git/hooks/pre-commit if it was previously installed.
If no hook is installed, prints a message and exits successfully.

If no path is provided, the current directory is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHookUninstall,
}

func init() {
	hookCmd.AddCommand(hookInstallCmd)
	hookCmd.AddCommand(hookUninstallCmd)
	rootCmd.AddCommand(hookCmd)
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Verify this is a git repository by checking .git
	gitDir := filepath.Join(projectRoot, ".git")
	gitInfo, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not a git repository")
		}
		return fmt.Errorf("cannot access .git: %w", err)
	}

	// Determine hooks directory.
	// Normal repos: .git/hooks/
	// Worktrees: .git is a file — not yet supported.
	var hooksDir string
	if gitInfo.IsDir() {
		hooksDir = filepath.Join(gitDir, "hooks")
	} else {
		return fmt.Errorf("not a git repository (worktrees are not yet supported)")
	}

	// Ensure hooks directory exists (git init creates it, but be safe).
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("cannot create hooks directory: %w", err)
	}

	// Write hook script as executable.
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("cannot write hook script: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Pre-commit hook installed at %s\n", hookPath)
	return nil
}

func runHookUninstall(cmd *cobra.Command, args []string) error {
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")

	// If no hook exists, print a message and return success.
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		fmt.Fprintln(cmd.OutOrStdout(), "no pre-commit hook installed")
		return nil
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("cannot remove hook: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Pre-commit hook removed from %s\n", hookPath)
	return nil
}
