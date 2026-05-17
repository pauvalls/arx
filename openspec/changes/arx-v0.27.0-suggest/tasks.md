# Tasks: arx v0.27.0 — Suggest Command

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~600-800 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: FixEngine core + templates → PR 2: CLI command + apply/rollback → PR 3: Tests + polish |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | FixEngine, Fix struct, template registry, 3 fix templates + unit tests | PR 1 | Base branch: main; self-contained, no CLI dependency |
| 2 | `arx suggest` cobra command, --apply/--force/--output flags, backup/rollback | PR 2 | Base branch: main; depends on PR 1 types |
| 3 | Integration tests, README update, roadmap update | PR 3 | Base branch: main; depends on PR 1 + PR 2 |

## Phase 1: Fix Engine Core

- [ ] 1.1 Create `internal/application/suggest.go` with `Fix` struct (`ViolationID`, `File`, `Original`, `Suggested`, `Description`) and `UnifiedDiff()` method using `diffmatchpatch` or `go-diff`
- [ ] 1.2 Add `FixTemplate` type (`func(violation domain.Violation, fileContent string) Fix`) and `FixEngine` struct with `templates map[string]FixTemplate`
- [ ] 1.3 Implement `NewFixEngine()` registering templates by key: `"domain-infrastructure"`, `"domain-application"`, `"application-infrastructure"`, `"circular"`, `"default"`
- [ ] 1.4 Implement `FixEngine.Suggest(violation domain.Violation) (Fix, error)` — reads file content, looks up template by `SourceLayer-TargetLayer` key, falls back to `"default"`
- [ ] 1.5 Implement `extractInterfaceTemplate` for `"domain-infrastructure"`: detects struct field with infra import, generates interface in domain + DI constructor pattern
- [ ] 1.6 Implement `definePortTemplate` for `"application-infrastructure"`: detects concrete infra dependency, generates port interface suggestion
- [ ] 1.7 Implement `extractSharedAbstractionTemplate` for `"circular"`: identifies shared types between cyclic packages, suggests extraction to shared package
- [ ] 1.8 Implement `genericAdviceTemplate` for `"default"`: returns text-based fix guidance using existing `getDetailedFixGuidance` patterns from `explain.go`
- [x] 1.9 Implement `FixEngine.Apply(fix Fix, backupDir string) error` — creates timestamped backup in `.arx-backup/`, writes suggested content atomically
- [x] 1.10 Implement `FixEngine.Rollback(file string, backupDir string) error` — restores file from latest backup

## Phase 2: CLI Command

- [x] 2.1 Create `cmd/arx/suggest.go` with cobra command `suggest [violation-id]`, flags `--apply`, `--force`, `--output`
- [x] 2.2 Wire command registration in `init()` via `rootCmd.AddCommand(suggestCmd)`
- [x] 2.3 Implement `runSuggest` — loads violations via `output.LoadViolations()`, resolves target (single by ID or all), creates `FixEngine`
- [x] 2.4 Format output as unified diff to stdout for each violation; support `--output` to write diffs to file instead
- [x] 2.5 Implement `--apply` flow: without `--force`, prompt for interactive confirmation; with `--force`, apply directly
- [x] 2.6 Implement backup/rollback on `--apply`: backup all originals before writing, restore all on any write error, clean up backup dir on success
- [x] 2.7 Handle edge cases: no cache → error "run arx check first"; unknown violation ID → clear error; non-Go file → output generic advice as text (not diff)

## Phase 3: Testing

- [ ] 3.1 Create `internal/application/suggest_test.go` — test template matching for each layer pattern key
- [ ] 3.2 Test `Fix.UnifiedDiff()` output with golden file comparison for diff format
- [ ] 3.3 Test `FixEngine.Suggest()` with missing file returns descriptive error
- [ ] 3.4 Test `FixEngine.Apply()` creates backup with original content in `.arx-backup/`
- [ ] 3.5 Test `FixEngine.Rollback()` restores file content matching backup
- [ ] 3.6 Test `FixEngine.Apply()` error triggers rollback and original content is restored
- [x] 3.7 Create `cmd/arx/suggest_test.go` — test `arx suggest` without cache returns error
- [ ] 3.8 Test `arx suggest D-01` outputs unified diff for specific violation
- [ ] 3.9 Test `arx suggest` (no ID) outputs diffs for all cached violations
- [ ] 3.10 Test `arx suggest --apply --force` modifies files and creates backups
- [x] 3.11 Test `arx suggest --output diff.patch` writes diff to file, nothing to stdout

## Phase 4: Polish

- [ ] 4.1 Update `ROADMAP.md` to mark v0.27.0-suggest as in progress/completed
- [ ] 4.2 Update `README.md` with `arx suggest` usage examples in commands section
- [ ] 4.3 Run `go test ./...` — all existing tests pass, new tests pass
- [ ] 4.4 Run `go vet ./...` and `golangci-lint run` — no new warnings
