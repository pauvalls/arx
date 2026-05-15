# Pre-commit Hook

Prevent architectural violations from being committed.

## Problem

Teams enforce architecture in CI, but CI feedback takes minutes. By the time CI runs,
the violating code is already committed and pushed. Fixing it requires a follow-up PR.

## Solution

`arx hook install` creates a git pre-commit hook that blocks commits with violations:

```bash
# Install the hook
arx hook install
# → Hook installed: .git/hooks/pre-commit

# Try to commit with violations
git commit -m "feat: add new service"
# → Architecture violation(s) found. Run 'arx check' for details.
# → Commit blocked.

# Fix violations, then commit succeeds
git commit -m "feat: add new service"
# → 1 file changed, 42 insertions(+)
```

## Commands

```bash
arx hook install                # Install pre-commit hook
arx hook install --force        # Overwrite existing hook
arx hook uninstall              # Remove pre-commit hook
```

## How It Works

The hook script:
1. Detects the project root via `git rev-parse --show-toplevel`
2. Runs `arx check --no-cache` on staged files
3. If new violations → prints message and exits 1 (blocks commit)
4. If only suppressed violations → allows commit
5. If `SKIP=arx` environment variable is set → skips the hook

## Bypassing the Hook

```bash
# Skip hook for a single commit
SKIP=arx git commit -m "temp: debugging"

# Uninstall permanently
arx hook uninstall
```

## Best Practices

1. **Install hook per developer** — Hooks are `.git/hooks/` which is not committed
2. **Use baseline first** — Create a baseline before installing the hook to avoid blocking existing violations
3. **CI is the source of truth** — CI still validates architecture even if hooks are bypassed
4. **Hook respects baseline** — Only blocks NEW violations, not suppressed ones
