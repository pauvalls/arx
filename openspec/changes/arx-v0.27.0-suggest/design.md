# Design: arx v0.27.0 — Suggest Command

## Technical Approach

Add a `suggest` subcommand that reads cached violations (reusing `output.LoadViolations`), maps each violation to a fix template based on its `SourceLayer`→`TargetLayer` pattern, and produces unified diffs. The fix engine lives in `internal/application/suggest.go` as a `FixEngine` struct with a template registry. For Go files, templates use the file content to generate context-aware suggestions; for non-Go, generic text-based advice is returned. The `--apply` flag writes fixes atomically with backup/rollback via `.arx-backup/`.

This reuses existing patterns: Cobra command in `cmd/arx/`, service in `internal/application/`, and violation caching in `internal/infrastructure/output/`.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Fix generation | Template registry with pattern matching | LLM-based or AST rewriting | Templates are deterministic, fast, and require no external dependencies. AST rewriting (go/ast) adds complexity for marginal gain in v0.27. |
| Violation source | Cache-first, fallback to running check | Always run check | Cache is instant; check is slow. Cache already exists and is used by `explain`. |
| Apply safety | Backup to `.arx-backup/` with atomic restore | In-place edit without backup | Developers need a safety net. `.arx-backup/` mirrors `.arx-cache/` convention. |
| Output format | Unified diff to stdout | JSON or inline patch | Unified diff is universal, reviewable, and pipeable. JSON can be added later. |
| Confirmation | `--apply` requires interactive confirmation unless `--force` | Auto-apply on `--apply` | Destructive writes need explicit opt-in. `--force` is standard for CI/automation. |
| Non-Go support | Generic text advice (no file changes) | Language-specific templates | Go is the primary target; non-Go templates deferred until demand exists. |

## Data Flow

```
arx suggest [violation-id] [--apply] [--output file]
    │
    ├──► Load violations from .arx-cache/violations.json
    │    (if cache missing/expired → run check flow)
    │
    ├──► Resolve target violation(s):
    │    - With ID → single violation
    │    - Without ID → all violations
    │
    ├──► FixEngine.Suggest(violation) → Fix
    │    │
    │    ├──► Lookup template by SourceLayer-TargetLayer
    │    ├──► Read source file content
    │    ├──► Generate suggested change (text patch)
    │    └──► Return Fix{File, Original, Suggested, Description}
    │
    ├──► Format as unified diff → stdout (or --output file)
    │
    └──► If --apply:
         ├──► Prompt for confirmation (unless --force)
         ├──► Backup originals to .arx-backup/
         ├──► Write suggested content to files
         └──► On error → restore from .arx-backup/, clean up
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/arx/suggest.go` | Create | Cobra command for `arx suggest`. Flags: `--apply`, `--force`, `--output`. Reuses `output.LoadViolations` and `output.GetViolationByID`. |
| `internal/application/suggest.go` | Create | `FixEngine` struct with template registry. `Suggest(violation, fileContent) → Fix`. Template functions for Go patterns: domain→infra (extract interface), application→infra (move to port), circular (extract shared abstraction). |
| `internal/application/suggest_test.go` | Create | Unit tests for template matching, fix generation, and edge cases (missing file, unknown pattern). |
| `cmd/arx/suggest_test.go` | Create | Integration tests for command flags, confirmation flow, and apply/rollback. |

## Interfaces / Contracts

### Fix Struct (`internal/application/suggest.go`)

```go
// Fix represents a suggested code change for a violation.
type Fix struct {
	ViolationID string
	File        string
	Original    string
	Suggested   string
	Description string
}

// UnifiedDiff returns the fix as a unified diff string.
func (f Fix) UnifiedDiff() string
```

### FixEngine (`internal/application/suggest.go`)

```go
// FixEngine generates fix suggestions for violations.
type FixEngine struct {
	templates map[string]FixTemplate
}

// FixTemplate generates a fix for a violation given file content.
type FixTemplate func(violation domain.Violation, fileContent string) Fix

// NewFixEngine creates a FixEngine with built-in templates.
func NewFixEngine() *FixEngine

// Suggest generates a fix for the given violation.
// Reads the file content automatically if violation.File is set.
func (e *FixEngine) Suggest(violation domain.Violation) (Fix, error)

// Apply writes the fix to disk, creating a backup first.
func (e *FixEngine) Apply(fix Fix, backupDir string) error

// Rollback restores a file from backup.
func (e *FixEngine) Rollback(file string, backupDir string) error
```

### Template Registry (built-in)

```
Key                      → Template Function
"domain-infrastructure"  → extractInterfaceTemplate
"domain-application"     → moveUseCaseTemplate
"application-infrastructure" → definePortTemplate
"circular"               → extractSharedAbstractionTemplate
"default"                → genericAdviceTemplate
```

### Unified Diff Output Format

```diff
--- a/internal/domain/order.go
+++ b/internal/domain/order.go
@@ -3,7 +3,12 @@
 package domain

-import "github.com/example/app/internal/infrastructure/postgres"
+// OrderRepository defines the interface for persisting orders.
+type OrderRepository interface {
+    Save(order *Order) error
+    FindByID(id string) (*Order, error)
+}

 type Order struct {
-    repo *postgres.OrderRepository
+    repo OrderRepository
 }
```

### Apply Backup Convention

```
.arx-backup/
├── 20260517T143022/
│   ├── internal/domain/order.go.bak
│   └── internal/application/service.go.bak
└── latest → 20260517T143022/  (symlink for easy access)
```

Backup directory name uses ISO 8601 timestamp. Original file path is preserved with `.bak` suffix. On successful apply, backups are retained (user deletes manually). On error, backups are restored and directory cleaned up.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Template matching by layer pattern | Assert correct template selected for each SourceLayer-TargetLayer combo |
| Unit | `Fix.UnifiedDiff()` output | Golden file comparison for diff format |
| Unit | `FixEngine.Suggest()` with missing file | Returns error with clear message |
| Unit | `FixEngine.Apply()` backup creation | Assert `.arx-backup/` contains original content |
| Unit | `FixEngine.Rollback()` restore | Assert file content matches backup |
| Unit | `FixEngine.Apply()` error → rollback | Simulate write failure, assert original restored |
| Integration | `arx suggest` without cache | Error: "run arx check first" |
| Integration | `arx suggest D-01` | Outputs unified diff for specific violation |
| Integration | `arx suggest` (no ID) | Outputs diffs for all violations |
| Integration | `arx suggest --apply --force` | Files modified, backups created |
| Integration | `arx suggest --apply` (interactive) | Prompts for confirmation, applies on "y" |
| Integration | `arx suggest --output diff.patch` | Writes diff to file, nothing to stdout |

## Migration / Rollout

No migration required. The command is additive and does not modify existing behavior:
- `arx check`, `arx explain`, and all existing commands are unaffected.
- `.arx-backup/` is only created when `--apply` is used.
- Cache format is unchanged; `suggest` reads the same `violations.json` as `explain`.

## Open Questions

- [ ] Should `--apply` support a dry-run mode that shows diffs but doesn't write? **Decision: yes** — `--apply` without `--force` already acts as dry-run + confirmation. The diff output is the dry-run. No separate flag needed.
- [ ] Should the fix engine support multi-file fixes (e.g., circular dependency requiring changes in 2+ files)? **Decision: yes** — the `Suggest` method returns a single `Fix`, but the command loop can collect multiple fixes and apply them atomically. The template for circular dependencies should return fixes for all affected files.
- [ ] Should non-Go violations produce any file changes? **Decision: no** — non-Go violations output generic advice as text (not a diff). The diff section is only shown for Go files where concrete patches exist.
