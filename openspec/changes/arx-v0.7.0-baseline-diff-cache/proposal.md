# Proposal: Arx v0.7.0 — Baseline, Diff Mode, and Performance Cache

## Intent

Arx works well for greenfield projects but has three adoption blockers for teams inheriting existing codebases:

1. **Existing violations flood CI**: Teams enabling `arx check` in CI get hundreds of failures from legacy violations. They can't distinguish new regressions from existing debt.
2. **No PR-level review**: Architects cannot see the architectural impact of a single PR or commit — only the full-project state.
3. **Slow re-runs on large projects**: `arx check` re-parses every file on every run, making iterative development painful on 100K+ LOC projects.

This change makes arx production-ready for team adoption by solving all three problems without breaking existing workflows.

## Scope

### In Scope
- `arx baseline` — generates `.arx-baseline.json` from current violations
- Modified `arx check` — skips violations present in baseline, fails only on NEW ones
- `arx diff <ref-before> <ref-after>` — compares violations between two git refs
- `internal/infrastructure/cache/file_cache.go` — file-hash-keyed detector result cache
- Config section `cache:` in `arx.yaml` (enabled/disabled, TTL, max size)

### Out of Scope
- Auto-baseline generation on first run (manual `arx baseline` only)
- Diff mode without git (filesystem-only diff deferred)
- Distributed/remote cache (local file cache only)
- Baseline merge/conflict resolution for teams
- SARIF baseline support (JSON only for v0.7.0)

## Capabilities

### New Capabilities
- `baseline-suppressions`: Baseline generation, storage, and violation filtering during check
- `diff-mode`: Git-ref-based violation comparison showing added/removed/fixed violations
- `performance-cache`: File-hash-keyed caching of detector results with invalidation

### Modified Capabilities
- `check-command`: Modified to filter baseline violations; exit code changes when baseline exists (only new violations cause failure)

## Approach

**Implementation order: cache → baseline → diff**

1. **Performance Cache** (foundation): Add `Cache` interface to detector contract. Implement `FileCache` backed by `.arx-cache/` directory keyed on SHA-256 of file content. Detectors check cache before parsing. Cache invalidation on config change or cache TTL expiry.

2. **Baseline** (high value): New `BaselineService` reads/writes `.arx-baseline.json` (violation fingerprints: file + rule + line hash). `arx baseline` command generates it. `arx check` loads baseline and filters matched violations. Exit code = 1 only if new violations exist beyond baseline.

3. **Diff Mode** (complex): New `DiffService` runs check on two git refs using `git worktree` or `git show` to extract files at each ref. Compares violation sets and renders added/removed/fixed. Requires `git` binary on PATH.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/arx/baseline.go` | New | Baseline command handler |
| `cmd/arx/diff.go` | New | Diff command handler |
| `internal/domain/baseline.go` | New | Baseline model and fingerprint logic |
| `internal/application/baseline.go` | New | Baseline service (generate, load, filter) |
| `internal/application/diff.go` | New | Diff service (git ref comparison) |
| `internal/infrastructure/cache/file_cache.go` | New | File-hash cache implementation |
| `internal/infrastructure/baseline/storage.go` | New | JSON file persistence for baseline |
| `internal/domain/detector.go` | Modified | Add optional cache parameter to interface |
| `internal/application/check.go` | Modified | Filter violations against baseline |
| `cmd/arx/check.go` | Modified | Exit code logic with baseline awareness |
| `arx.yaml` schema | Modified | Add `cache:` configuration section |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Cache invalidation misses config changes | Medium | Include config hash in cache key; invalidate on `arx.yaml` change |
| Diff mode fails on dirty working tree | Medium | Require clean working tree or use `git stash` temporarily; document requirement |
| Baseline file grows large on big projects | Low | Baseline is compact (fingerprints, not full violations); compress if >1MB |
| Git binary not available in CI | Low | Graceful degradation: diff command returns error with helpful message |
| Cache race conditions on parallel runs | Low | Use file locking or atomic writes for cache entries |

## Rollback Plan

1. **Cache**: Remove `cache:` config section, delete `.arx-cache/` directory. Detectors fall back to no-cache path. Zero behavioral change.
2. **Baseline**: Remove `.arx-baseline.json`, delete baseline commands/services. `arx check` reverts to original "fail on all violations" behavior.
3. **Diff**: Remove `cmd/arx/diff.go` and `internal/application/diff.go`. No persistent state created.

All three features are additive. No breaking changes to existing commands, config, or output formats.

## Dependencies

- Go 1.21+ (already required)
- `git` binary on PATH (required for diff mode only)
- Existing detector infrastructure (Go, TypeScript, Python, Java)
- Existing `arx check` command and violation model

## Success Criteria

- [ ] `arx baseline` generates `.arx-baseline.json` with all current violations
- [ ] `arx check` with baseline exits 0 when no new violations exist
- [ ] `arx check` with baseline exits 1 when new violations are detected
- [ ] `arx diff HEAD~1 HEAD` shows added/removed violations correctly
- [ ] Cache reduces re-run time by >50% on unchanged files
- [ ] Cache invalidates correctly when `arx.yaml` changes
- [ ] All existing tests pass without modification
- [ ] 80%+ test coverage for baseline, diff, and cache packages
