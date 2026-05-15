# Diff

Compare architecture between two git refs to see what changed in a PR or commit.

## Problem

Code reviews need to catch architectural changes, but manual review is error-prone.
"Did this PR introduce a new violation?" should be answered automatically.

## Solution

`arx diff` runs an audit on two git refs and compares the results:

```bash
# Compare last commit vs previous
arx diff HEAD~1 HEAD
# → +2 violations, -1 resolved, 12 unchanged

# Compare branch vs main
arx diff main feature/new-service

# JSON output for CI
arx diff main HEAD --format json
```

## Output

### Terminal (default)

![green]"- RESOLVED: domain → infrastructure"  
![red]"+ NEW: application → infrastructure"  
![dim]"= UNCHANGED: 12 violations"  

```
Architecture diff: +3 violations, -1 resolved, 12 unchanged
  + domain/order.go:14 → infrastructure/postgres.go  (NEW)
  - domain/user.go:22 → domain/user.go:22  (RESOLVED)
```

### JSON

```json
{
  "ref_before": "main",
  "ref_after": "feature/new-service",
  "added": [...],
  "resolved": [...],
  "unchanged": [...],
  "config_changed": false
}
```

## Exit Codes

- **0**: No architectural changes
- **1**: New violations introduced

## How It Works

1. `git worktree add` checks out each ref into a temp directory
2. Runs the full audit pipeline on each
3. Compares violations by fingerprint (`rule_id:source_layer:target_layer:import`)
4. Clean up worktrees (even on error)

## Use Cases

### PR Review

```bash
# In CI, compare PR branch against base
arx diff origin/main origin/HEAD

# Fail if new violations introduced
if arx diff origin/main origin/HEAD --format json | jq -e '.added | length > 0'; then
  echo "Architecture violations found!"
  exit 1
fi
```

### Release Gate

```bash
# Before releasing, check nothing degraded since last tag
arx diff v0.8.0 HEAD --format json
```
