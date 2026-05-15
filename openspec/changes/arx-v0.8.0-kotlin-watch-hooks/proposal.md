# Proposal: Arx v0.8.0 — Kotlin Detector, Watch Mode, Pre-commit Hooks, Custom Rules

## Intent

Improve developer experience and language coverage to make arx a daily-use tool, not just a CI gate. Teams need real-time feedback while coding, commit-time enforcement, custom conventions, and Kotlin support — the most requested enterprise gap from v0.6.0 Java support.

## Scope

### In Scope
- Kotlin detector — `.kt` / `.kts` parsing, Gradle Kotlin DSL + Maven detection
- `arx check --watch` — fsnotify-based file watching with debounced re-check
- `arx hook install` — POSIX pre-commit hook generator, baseline-aware
- Custom rule `pattern` field — regex matching on import paths in `arx.yaml`

### Out of Scope
- Android-specific resource analysis (`.xml`, `R.class`)
- GUI watch mode — terminal only
- Pre-commit for Mercurial / SVN — git only
- Interactive rule builder — YAML only

## Capabilities

### New Capabilities
- `kotlin-detector`: Kotlin project detection and import extraction from `.kt` files
- `watch-mode`: Real-time `arx check --watch` with fsnotify and debounced re-runs
- `pre-commit-hook`: Git pre-commit hook install via `arx hook install`
- `custom-rules-patterns`: Regex-based `pattern` field on rules, applied during check

### Modified Capabilities
- `check-command`: Extended with `--watch` flag; output mode optimized for hook exit codes

## Approach

**Implementation order: custom rules → kotlin detector → pre-commit hook → watch mode**

1. **Custom rules** (foundation): Add `pattern` field to `Rule` model. Compile to `regexp.Regexp` at config load. Reject invalid patterns at startup.
2. **Kotlin detector** (quick win): Follow Java detector pattern. Detect `build.gradle.kts` / `pom.xml`. Extract `import`, `package`, `object`, `companion object`, `sealed class` via regex.
3. **Pre-commit hook** (high value): Generate `.git/hooks/pre-commit` that runs `arx check --exit-code`. Respects baseline — only blocks NEW violations.
4. **Watch mode** (complex): fsnotify watcher on project roots. Debounce 500ms. Re-run check, diff output vs last run. Respect `.arxignore`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/infrastructure/detector/kotlin_detector.go` | New | Kotlin detector implementation |
| `internal/domain/rule.go` | Modified | Add `Pattern string` and compiled `*regexp.Regexp` |
| `internal/application/check.go` | Modified | Pattern matching + watch loop orchestration |
| `internal/infrastructure/watch/fswatcher.go` | New | fsnotify-based file watcher |
| `internal/ports/detector.go` | Modified | (if needed) extend detector contract |
| `cmd/arx/hook.go` | New | Hook install/uninstall command |
| `cmd/arx/check.go` | Modified | Add `--watch` flag |
| `arx.yaml` schema | Modified | Add `rules[].pattern` field |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Watch mode CPU on large projects | Med | Debounce 500ms, respect `.arxignore`, limit watcher depth |
| Pre-commit hook portability | Low | POSIX shell only; test on Linux (CI) + macOS (manual) |
| Regex perf on many rules | Low | Compile once at load; cache match results per file |
| Kotlin + Java mixed projects | Med | Run both detectors, merge violation sets |

## Rollback Plan

All features additive:
1. **Custom rules**: Remove `pattern` from config schema and domain model. Existing rules unaffected.
2. **Kotlin detector**: Delete `kotlin_detector.go`; registry skips it gracefully.
3. **Pre-commit hook**: `arx hook uninstall` removes the hook; manual `.git/hooks/pre-commit` deletion also safe.
4. **Watch mode**: Remove `--watch` flag from check command; delete fswatcher package.

## Dependencies

- Go 1.21+ (existing)
- `github.com/fsnotify/fsnotify` v1.7+ (new — watch mode)
- Existing detector infrastructure + check pipeline
- Kotlin detector depends on custom rules pattern matching

## Success Criteria

- [ ] KotlinDetector detects Gradle Kotlin DSL (`build.gradle.kts`) and Maven (`pom.xml`)
- [ ] KotlinDetector extracts imports from `.kt` files with ≥90% precision
- [ ] Kotlin imports resolved to layers correctly
- [ ] `arx check --watch` re-runs within 1s of file save (debounce inclusive)
- [ ] `arx hook install` creates functional pre-commit hook; hook exits 1 on new violations
- [ ] Custom rule `pattern: "com\.legacy\..*"` blocks/reports matching imports
- [ ] Invalid regex in `pattern` field fails config load with clear error
- [ ] All v0.7.0 tests pass without modification
- [ ] ≥80% coverage on kotlin_detector.go, fswatcher.go, hook.go
