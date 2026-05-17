# Proposal: arx v0.27.0 — Suggest Command

## Why

Arx detects architectural violations and explains them (`arx explain`), but developers still need to manually figure out how to fix each violation. A `suggest` command closes the loop: it generates concrete fix suggestions (as unified diffs) and can optionally apply them.

## What

- New command: `arx suggest [violation-id]`
- Loads cached violations (or runs `check` if cache is stale)
- Generates fix suggestions based on violation type and language
- Outputs unified diffs to stdout
- `--apply` flag writes fixes to disk with backup/rollback
- `--output` flag writes diffs to a file

## Approach

Template-based fix generation in `internal/application/suggest.go`. Templates map violation patterns (domain→infra, application→infra, circular) to concrete AST-aware refactoring suggestions for Go, and generic advice for non-Go languages.

## Rollback

`--apply` backs up files to `.arx-backup/` before writing. On error, all backups are restored.
