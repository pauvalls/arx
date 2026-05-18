# arx suggest — Auto-Fix Suggestions

The `arx suggest` command analyzes violations and generates concrete, diff-formatted fix suggestions. It can optionally auto-apply fixes with safety backups.

## Usage

```bash
# Show suggestions for all violations
arx suggest

# Show suggestion for a specific violation
arx suggest D-01

# Auto-apply suggestions (with confirmation prompt)
arx suggest --apply

# Auto-apply without confirmation
arx suggest --apply --force

# Write diffs to a file instead of stdout
arx suggest --output fixes.diff
```

## Example

Running `arx suggest` on a project with a domain→infrastructure violation:

```bash
$ arx suggest

╔══════════════════════════════════════════════════════════════════╗
║                    FIX SUGGESTIONS                               ║
╚══════════════════════════════════════════════════════════════════╝

D-01: domain → infrastructure
  File: internal/domain/service.go:42
  Rule: domain-no-infra
  Fix:  Extract an interface from infra/db and inject it via constructor

--- a/internal/domain/service.go
+++ b/internal/domain/service.go
@@ -42 +42 @@
-// TODO: replace direct import with interface
+// Define interface in domain layer, inject via constructor
```

## Auto-Apply

When `--apply` is used, arx:

1. Creates a timestamped backup in `.arx-backup/YYYYMMDDTHHMMSS/`
2. Applies the fix to the source file
3. Reports success or rolls back on failure

```bash
$ arx suggest --apply

╔══════════════════════════════════════════════════════════════════╗
║                    APPLYING FIXES                                ║
╚══════════════════════════════════════════════════════════════════╝

D-01: internal/domain/service.go → Extract interface
  Backup: .arx-backup/20260518T120000/internal/domain/service.go.bak
  ✓ Applied

Summary: 1 fix applied, 0 errors
```

## arx explain — Detailed Guidance

The `explain` command shows comprehensive violation context, including fix suggestions:

```bash
$ arx explain D-01

╔══════════════════════════════════════════════════════════════════╗
║              ARCHITECTURE VIOLATION EXPLAINED                    ║
╚══════════════════════════════════════════════════════════════════╝

Violation: D-01
Severity:   ❌ Error
File:       internal/domain/service.go:42
Rule:       domain-no-infra
Import:     infra/db

┌──────────────────────────────────────────────────────────────────┐
│ WHY THIS MATTERS                                                 │
└──────────────────────────────────────────────────────────────────┘
The Dependency Inversion Principle (SOLID-D) states that high-level
modules (domain) should not depend on low-level modules...

┌──────────────────────────────────────────────────────────────────┐
│ AUTO-FIX SUGGESTION                                              │
└──────────────────────────────────────────────────────────────────┘

  Run 'arx suggest D-01' to auto-apply this fix.

--- a/internal/domain/service.go
+++ b/internal/domain/service.go
@@ -42 +42 @@
-// TODO: replace direct import with interface
+// Define interface in domain layer, inject via constructor
```

## Supported Fix Templates

| Rule Pattern | Fix Strategy |
|-------------|--------------|
| `domain → infrastructure` | Extract interface + inject via constructor |
| `application → infrastructure` | Move dependency behind port interface |
| Unknown pattern | Generic refactoring advice |
