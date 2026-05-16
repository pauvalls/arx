# Tasks: Maturity Features (v0.20.0)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 250-350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (5 independent features, all small) |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | JSON Schema + $schema injection | PR 1 | New file + 2 small mods; tests included |
| 2 | NO_COLOR support | PR 2 | terminal.go mod + tests; independent |
| 3 | Smart Init (.gitignore) | PR 3 | init.go mod + tests; independent |
| 4 | Verbose Check (per-detector status) | PR 4 | check.go + application mod + tests; independent |
| 5 | Polish (roadmap + full suite) | PR 5 | Docs + verify all pass |

## Phase 1: Foundation (JSON Schema + Config)

- [ ] 1.1 Create `arx-schema.json` at project root with JSON Schema draft-07 covering all Config fields (version, layers, rules, language_overrides, exclude, severity_config, max_violations, severity_mapping) per design.md schema contract
- [ ] 1.2 Add `Schema string` field to `domain.Config` struct with json/yaml tag `"$schema"` in `internal/domain/config.go`
- [ ] 1.3 Add test in `internal/domain/config_test.go` verifying `$schema` field marshals/unmarshals correctly with yaml tag

## Phase 2: Core Implementation

- [ ] 2.1 In `cmd/arx/init.go`, add `isDefaultConfigPath(path string) bool` helper that returns true for "arx.yaml" or paths ending in "/arx.yaml"
- [ ] 2.2 In `cmd/arx/init.go` `runInit`, after service returns config and before WriteConfig, set `config.Schema = "./arx-schema.json"` when `isDefaultConfigPath(initOutput)` is true
- [ ] 2.3 In `internal/infrastructure/output/terminal.go`, add `var noColor bool` and `func init()` that reads `os.Getenv("NO_COLOR")` and sets `noColor = v != "" && v != "0"`
- [ ] 2.4 In `terminal.go`, add helper `func style(s lipgloss.Style, text string) string` that returns plain text when `noColor` is true, otherwise `s.Render(text)`
- [ ] 2.5 Replace all direct `style.Render(text)` calls in `TerminalReporter.Report()` with `style(styleVar, text)` helper calls
- [ ] 2.6 In `cmd/arx/init.go`, add `ensureGitignoreEntries(projectRoot string) error` that checks for `.git` dir, reads/creates `.gitignore`, and appends `.arx-cache/` and `.arx-baseline.json` if missing
- [ ] 2.7 In `cmd/arx/init.go` `runInit`, call `ensureGitignoreEntries(projectRoot)` after the success message print
- [ ] 2.8 In `internal/application/check.go`, add `DetectorStatus` struct and `RunDetectorsWithStatus` function that wraps existing errgroup logic to collect per-detector name, applicable bool, depCount, and error

## Phase 3: Integration (Verbose Check Wiring)

- [ ] 3.1 In `cmd/arx/check.go` `runCheckWithService`, replace `service.DetectCached` call with `RunDetectorsWithStatus` when `checkVerbose` is true, capturing `[]DetectorStatus`
- [ ] 3.2 In `cmd/arx/check.go`, after detector run in verbose mode, print per-detector status lines to stderr: `✓ <name> (applicable, N deps extracted)` or `✗ <name> (not applicable)` or `✗ <name> (error: <msg>)`

## Phase 4: Testing

- [ ] 4.1 Test: `arx-schema.json` is valid JSON (parse with `encoding/json` in `cmd/arx/init_test.go` or new `schema_test.go`)
- [ ] 4.2 Test: `$schema` field present in generated `arx.yaml` when using default output path (test `runInit` with default `initOutput`)
- [ ] 4.3 Test: `$schema` field absent when using custom `--output` path (test `runInit -o custom.yaml`)
- [ ] 4.4 Test: `NO_COLOR=1` sets `noColor=true` in `terminal.go` (set env in test, verify output has no ANSI escape codes)
- [ ] 4.5 Test: `NO_COLOR=0` keeps `noColor=false` (set env to "0", verify colors remain)
- [ ] 4.6 Test: `NO_COLOR` unset keeps `noColor=false` (unset env, verify colors remain)
- [ ] 4.7 Test: `.gitignore` entries appended when missing (create temp git repo with `git init`, run init logic, verify entries present)
- [ ] 4.8 Test: `.gitignore` entries not duplicated on repeated runs (run init logic twice, verify no duplicate entries)
- [ ] 4.9 Test: `.gitignore` skipped outside git repo (create temp dir without `.git`, verify no `.gitignore` created/modified)
- [ ] 4.10 Test: `RunDetectorsWithStatus` returns correct status for applicable detector with deps (use `mockDetector`)
- [ ] 4.11 Test: `RunDetectorsWithStatus` returns correct status for non-applicable detector (use `mockDetector` with `detectResult=false`)
- [ ] 4.12 Test: `RunDetectorsWithStatus` returns error status for failing detector (use `mockDetector` with `extractErr`)
- [ ] 4.13 Integration: `arx check --verbose` shows detector status lines (run against arx project itself, verify stderr contains detector lines)

## Phase 5: Polish

- [ ] 5.1 Update `docs/roadmap.md` (or equivalent) to mark v0.20.0 features as complete
- [ ] 5.2 Run `go test ./...` and verify full test suite passes with no regressions
