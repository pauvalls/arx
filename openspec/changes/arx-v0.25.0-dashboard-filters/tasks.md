# Tasks: arx v0.25.0 — Dashboard Filters, State Persistence, and Check Diff

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350-450 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR (3 independent but small scopes) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Dashboard filter UI + JS logic | PR 1 | Pure client-side, no Go changes, self-contained |
| 2 | State persistence + check --diff | PR 2 | Go changes only, depends on no PR, tests included |

## Phase 1: Dashboard Filters (client-side JS/CSS)

- [ ] 1.1 Add filter bar HTML above violations table in `dashboard.html`: severity checkboxes (error, warning, info), source layer `<select>` dropdown, and `<input type="text">` search field with placeholder text.
- [ ] 1.2 Add CSS for filter bar layout (flex row, gap, responsive wrap) and active filter indicator styles (highlighted checkbox/dropdown when filter is active).
- [ ] 1.3 Add `data-sortable` attribute to each `<th>` in the violations table header (Severity, Rule, File, Line, From, To, Message) and CSS cursor/indicator for sortable columns.
- [ ] 1.4 Add `<p id="filter-summary">` element after the filter bar to display "Showing X of Y violations" text.
- [ ] 1.5 Add JS `activeFilters` state object in `dashboard.html` script: `severities`, `sourceLayer`, `searchText`, `sortColumn`, `sortDirection` with defaults matching design.
- [ ] 1.6 Implement `applyFilters(violations)` function: filters by severity (includes all by default), sourceLayer (empty = all), searchText (matches rule_id, file, message case-insensitive).
- [ ] 1.7 Implement `sortByColumn(violations, column, direction)` function: single-column sort with toggle cycle asc → desc → none (original order).
- [ ] 1.8 Implement `populateLayerDropdown(violations)` function: extracts unique source_layer values from violations and populates the dropdown options.
- [ ] 1.9 Wire event listeners: checkbox `change` → update `activeFilters.severities` + re-render; dropdown `change` → update `activeFilters.sourceLayer` + re-render; search `input` → debounced (300ms `setTimeout`/`clearTimeout`) update `activeFilters.searchText` + re-render; `th` click → update sort state + re-render.
- [ ] 1.10 Update `renderViolations()` and `fetchData()` to call `applyFilters()` → `sortByColumn()` before rendering; update `#filter-summary` text content with count.

## Phase 2: State Persistence (Go)

- [x] 2.1 Add `serverStateSnapshot` struct to `state.go` with JSON tags: `LastCheck`, `Violations`, `Coupling`, `Debt`, `Config`, `CheckError` (matching design interface).
- [x] 2.2 Add `SaveToFile(path string) error` method on `ServerState`: acquire read lock, build snapshot, `json.MarshalIndent`, `os.WriteFile` with 0644 perms; ensure `.arx-cache/` directory exists via `os.MkdirAll`.
- [x] 2.3 Add `LoadFromFile(path string) error` method on `ServerState`: `os.ReadFile`, `json.Unmarshal` into snapshot, acquire write lock, restore fields; return nil error if file does not exist (`os.IsNotExist`), return error on corrupt JSON.
- [x] 2.4 Add `cachePath string` field to `Server` struct in `server.go` and update `New()` constructor to accept it.
- [x] 2.5 In `Start()`, call `s.state.LoadFromFile(s.cachePath)` before the initial `RunCheck()` call; log warning on error (non-fatal).
- [x] 2.6 In `runCheck()` (or `RunCheck()`), call `s.state.SaveToFile(s.cachePath)` after `SetCheckResult` succeeds; log warning on error (non-fatal).
- [x] 2.7 Update `cmd/arx/server.go` to pass `cachePath` (`.arx-cache/server-state.json`) when constructing the `Server`.

## Phase 3: Check Diff (Go)

- [ ] 3.1 Add `checkDiff bool` var and register `--diff` bool flag on `checkCmd` in `check.go` init (default false).
- [ ] 3.2 Add `saveLastCheck(violations, configHash, projectRoot string)` helper: marshal violations to `lastCheckCache` JSON format (version, timestamp, config_hash, violations array), write to `.arx-cache/last-check.json` via `os.MkdirAll` + `os.WriteFile`.
- [ ] 3.3 Add `loadLastCheck(projectRoot string) ([]domain.Violation, error)` helper: read `.arx-cache/last-check.json`, unmarshal, return violations slice; return nil+nil if file missing.
- [ ] 3.4 In `runCheck`, after `printCheckResult`, call `saveLastCheck()` unconditionally (always persist for next run).
- [ ] 3.5 When `--diff` is set and format is terminal: call `loadLastCheck()`, if no previous file print "no previous run to compare"; otherwise call `domain.DiffViolations(prev, current)` and print diff summary ("+N violations, -N resolved, N unchanged").
- [ ] 3.6 Skip diff output in JSON mode (JSON consumers implement their own diff).

## Phase 4: Testing

- [ ] 4.1 In `dashboard_test.go`, add `TestDashboard_ContainsFilterBar`: assert HTML contains severity checkboxes, layer dropdown (`<select>`), and search input.
- [ ] 4.2 In `dashboard_test.go`, add `TestDashboard_ContainsSortableHeaders`: assert each `<th>` in violations table has `data-sortable` attribute.
- [ ] 4.3 In `dashboard_test.go`, add `TestDashboard_ContainsFilterSummary`: assert HTML contains `id="filter-summary"` element.
- [ ] 4.4 In `state_test.go` (new file), add `TestServerState_SaveLoadRoundTrip`: create state with violations, save to temp file, load into new state, assert violations/coupling/debt match.
- [ ] 4.5 In `state_test.go`, add `TestServerState_LoadFromFile_MissingFile`: assert no error returned, state remains at defaults.
- [ ] 4.6 In `state_test.go`, add `TestServerState_LoadFromFile_CorruptJSON`: write invalid JSON to temp file, assert error returned.
- [ ] 4.7 In `check_test.go`, add `TestCheckCommand_HasDiffFlag`: assert `--diff` flag registered, bool type, default false (follows existing pattern).
- [ ] 4.8 Run full test suite: `go test ./...` passes with no failures.

## Phase 5: Polish

- [ ] 5.1 Update roadmap file (if exists) to mark v0.25.0 scope as in-progress.
- [ ] 5.2 Verify `go vet ./...` and `go build ./...` pass cleanly.
