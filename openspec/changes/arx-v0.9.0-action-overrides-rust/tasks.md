# Tasks: Arx v0.9.0 — Action Overrides & Rust Detector

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~910 (across all phases) |
| 400-line budget risk | **High** |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (overrides) → PR 2 (Rust detector) → PR 3 (GH Action) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Overrides foundation + check command + polish | PR 1 | base=main; phases 1, 4, 5 (tightly coupled) |
| 2 | Rust detector | PR 2 | base=main; independent, follows Kotlin pattern |
| 3 | GitHub Action | PR 3 | base=main; independent, wraps existing CLI |

## Phase 1: Per-Directory Overrides (Foundation)

- [ ] 1.1 Add `RuleOverride` struct to `internal/domain/rule.go` — `Path string`, `Severity Severity`, `Enabled *bool`
- [ ] 1.2 Add `Overrides []RuleOverride` field (`yaml:"overrides,omitempty"`) to `Rule` struct in `internal/domain/rule.go`
- [ ] 1.3 Add `GetEffectiveSeverity(filePath string) (Severity, bool)` — longest-prefix match wins
- [ ] 1.4 Add `IsEnabledFor(filePath string) bool` — returns false if any override matches with `Enabled: false`
- [ ] 1.5 Modify `EvaluateRules` in `internal/domain/audit.go` — call `IsEnabledFor` (skip disabled), call `GetEffectiveSeverity` (set on violation)
- [ ] 1.6 Add override validation to `Rule.Validate()` in `rule.go` — validate `path` non-empty, `severity` valid, no empty overrides
- [ ] 1.7 Write table-driven tests for `GetEffectiveSeverity` — empty, non-matching, single-match, multi-match (longest prefix)
- [ ] 1.8 Write table-driven tests for `IsEnabledFor` — no override → enabled, `Enabled: false` → disabled, multiple overrides
- [ ] 1.9 Write tests for `EvaluateRules` with overrides — rule disabled for path, severity override, unaffected files pass through

## Phase 2: Rust Detector (follows Kotlin pattern)

- [x] 2.1 Create `internal/infrastructure/detector/rust/detector.go` — `RustDetector` struct with `modulePrefix`, `sourceDirs`; `New()`, `Name()`, `Detect()` (checks `Cargo.toml`), `ExtractImports()`, `FindRustFiles()`, `parseFile()`, `resolveImport()`, `resolveSourcePath()`
- [x] 2.2 Create `internal/infrastructure/detector/rust/parser.go` — regex patterns for `use`, `use crate::`, `use self::`, `use super::`, `pub use`, `pub mod`; `extractImportsFromLine()`; `isExternalDependency()` (skip `std::`, `core::`, `alloc::`, `test::`)
- [x] 2.3 Register Rust detector in `internal/infrastructure/detector/registry.go` — import `rust` package, append `rust.New()` to detector list
- [x] 2.4 Create `internal/infrastructure/detector/rust/parser_test.go` — table-driven tests for all regex patterns, external skip list, edge cases
- [x] 2.5 Create `internal/infrastructure/detector/rust/detector_test.go` — `Detect` with fake `Cargo.toml`, `FindRustFiles` skips `*_test.rs` and `tests/`, `resolveImport` for crate-relative and external, integration test with real `Cargo.toml` + `src/lib.rs`

## Phase 3: GitHub Action

- [ ] 3.1 Create `.github/actions/arx-action/action.yml` — Docker action with inputs: `path`, `config`, `format`, `baseline`, `diagram`
- [ ] 3.2 Create `.github/actions/arx-action/Dockerfile` — multi-stage Go build or pre-built binary copy
- [ ] 3.3 Create `.github/actions/arx-action/entrypoint.sh` — wraps `arx check --ci --format sarif` with input args
- [ ] 3.4 Create `.github/workflows/arx-ci.yml` — `push`/`pull_request` triggers, checkout → arx-action → upload SARIF (uses `github/codeql-action/upload-sarif`)

## Phase 4: Check Command Modifications

- [ ] 4.1 Modify `runCheckWithService` in `cmd/arx/check.go` — split violations into active/overridden after `EvaluateRules`, populate `overriddenCount` in `checkResult`
- [x] 4.2 Add `OverriddenCount int` to `Summary` in `internal/infrastructure/output/json.go` — serialize with `omitempty`; use the violation's `Severity` field (not rule-ID heuristics) for error/warning/info counts
- [x] 4.3 Modify `ExitCode()` in `internal/infrastructure/output/terminal.go` — return 0 if only overridden violations remain
- [x] 4.4 Show overridden count in terminal verbose mode — print count to stderr when `checkVerbose && overriddenCount > 0`
- [x] 4.5 Write tests: JSON `Summary.OverriddenCount` serialize/deserialize, `ExitCode` with only-overridden vs mixed violations

## Phase 5: Polish

- [x] 5.1 Update `README.md` — document `overrides` field, Rust language support, `.github/actions/arx-action/`, exit code behavior with overrides
- [x] 5.2 Write end-to-end test: config with overrides → `arx check` → verify violations filtered and severity adjusted
- [x] 5.3 Write end-to-end test: Rust project (`Cargo.toml` + `src/lib.rs`) → `arx check` → verify dependencies extracted

## Critical Path

```
1.1-1.2 (RuleOverride model) → 1.3-1.4 (methods) → 1.5 (EvaluateRules) → 4.1-4.4 (check command)
                                                      ↓
                                             1.6 (config validation) → 1.7-1.9 (tests)

2.1-2.2 (detector + parser) → 2.3 (registry) → 2.4-2.5 (tests)   [fully parallel to Phase 1]

3.1-3.4 (GitHub Action)                                             [fully parallel to Phase 1 & 2]

4.5 (test) + 5.1-5.3 (docs + e2e)                                   [final gate]
```

Phases 1-4 each block only within themselves. Phase 4 depends on Phase 1 (needs the override domain model). Phases 2 and 3 are fully independent — can be parallelized across the PR split.
