# Tasks: Arx v0.8.0 — Kotlin Detector, Watch Mode, Pre-commit Hooks, Custom Rules

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,200 (82 + 458 + 220 + 331 + 110) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Custom Rules) → PR 2 (Kotlin Detector) → PR 3 (Hook) → PR 4 (Watch + Polish) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Base | Est. Δ | Notes |
|------|------|-----------|------|--------|-------|
| 1 | Custom Rules Patterns (Phase 1) | PR 1 | main | ~82 | Domain-only, no infra deps |
| 2 | Kotlin Detector (Phase 2) | PR 2 | main | ~458 | Follows Java pattern exactly; shared glob extraction |
| 3 | Pre-commit Hook (Phase 3) | PR 3 | main | ~220 | Standalone CLI command |
| 4 | Watch Mode + Polish (Phases 4–5) | PR 4 | main | ~441 | Most complex; combines watcher + CLI flag + docs |

Each unit is independent — no cross-phase dependency. Apply in any order.

---

## Phase 1: Custom Rules Patterns (foundation)

- [x] 1.1 Add `Pattern string` + `compiledPattern *regexp.Regexp` fields to `domain.Rule` struct (`internal/domain/rule.go`)
- [x] 1.2 Add regex compilation in `Rule.Validate()`: compile `Pattern` into `compiledPattern`, return error on invalid regex
- [x] 1.3 Modify `Rule.Violates()` to check `compiledPattern` against `importPath` when set (AND with from/to checks)
- [x] 1.4 Add table-driven tests: pattern match, pattern no-match, combined pattern+from/to, invalid pattern rejected at Validate (`internal/domain/rule_test.go`)

## Phase 2: Kotlin Detector (follows Java pattern)

- [x] 2.1 Extract `importMatchesLayer` from `java/parser.go` into `detector/shared/glob.go` as exported `ImportMatchesLayer`; refactor Java parser to call shared version
- [x] 2.2 Create `detector/kotlin/detector.go`: `KotlinDetector` struct, `New()`, `Name()`, `Detect()` (check `build.gradle.kts`, `settings.gradle.kts`, `pom.xml` + `.kt` files), `ExtractImports()`, `FindKotlinFiles()`, `shouldSkip`/`isExternalDependency` matching Java pattern
- [x] 2.3 Create `detector/kotlin/parser.go`: regex patterns for Kotlin imports (standard `import pkg.Class`, wildcard `import pkg.*`, alias `import pkg as alias`, nested class `import pkg.Class.Nested`), `extractImportsFromLine()`, `extractPackage()`
- [x] 2.4 Register Kotlin detector in `detector/registry.go`: import `kotlindetector` + append `kotlindetector.New()` to `GetDetectors()`
- [x] 2.5 Add test fixtures + table-driven tests: Detect() with fake `build.gradle.kts`, `pom.xml` + `.kt`, empty dir; import extraction for all Kotlin syntax variants (`detector/kotlin/` `*_test.go`)

## Phase 3: Pre-commit Hook (high value)

- [x] 3.1 Create `cmd/arx/hook.go`: `arx hook install` subcommand — verify `.git` directory exists, generate POSIX shell script, write to `.git/hooks/pre-commit`, set executable bit (`os.Chmod 0755`)
- [x] 3.2 Add `arx hook uninstall` subcommand — `os.Remove(.git/hooks/pre-commit)` with "no hook installed" error case
- [x] 3.3 Hook script content: `SKIP=arx` guard at top, `git rev-parse --show-toplevel` to detect root, calls `arx check --no-cache`; script respects `.arx-baseline.json` by default and passes through arx's exit code
- [x] 3.4 Add tests: script content string comparison, file permissions, non-git error, idempotent install, uninstall removes file, uninstall when no hook is graceful (`cmd/arx/hook_test.go` + `test/integration/hook_test.go`)

## Phase 4: Watch Mode (complex)

- [x] 4.1 Create `internal/domain/watch_result.go`: `WatchResult` struct with `Added`, `Resolved`, `Unchanged` violation slices + JSON tags; `ViolationKey` and `DiffViolations` functions
- [x] 4.2 Create `internal/infrastructure/watcher/watcher.go`: `Watcher` with fsnotify, debounce timer (default 500ms), `.gitignore` pattern matching, recursive dir watching, `Events() <-chan WatchEvent`, `Close()`
- [x] 4.3 Modify `cmd/arx/check.go`: add `--watch` flag + `--interval` flag; after initial check, start watcher event loop — on change: re-run detect+cached+evaluate, diff old/new violations, print diff summary
- [x] 4.4 Add diff output: terminal format prints "+N violations, -M resolved"; JSON format (`--watch --json`) outputs `WatchResult` JSON with `added`/`resolved`/`unchanged` arrays
- [x] 4.5 Add `github.com/fsnotify/fsnotify` to `go.mod` (`go get github.com/fsnotify/fsnotify@v1.7.0`)
- [x] 4.6 Add tests: debounce groups rapid events, event filtering by .gitignore, graceful shutdown via context cancel (`internal/infrastructure/watcher/watcher_test.go`)

## Phase 5: Polish

- [ ] 5.1 Update README with `--watch`, `--interval`, `hook install`, `hook uninstall` commands and examples
- [ ] 5.2 End-to-end integration test: create temp Kotlin project with `build.gradle.kts` + `.kt` files + `arx.yaml`, run `arx check`, verify violations detected
