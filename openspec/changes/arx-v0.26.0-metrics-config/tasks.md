# Tasks: arx v0.26.0 — Metrics, Config Set Improvements, and Quality Pass

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350 (additions) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Metrics struct + endpoint + dashboard cards | PR 1 | Self-contained, tests included |
| 2 | Config set dotted path + JSON array parsing | PR 2 | Independent of metrics, tests included |
| 3 | Quality pass (fuzz, race, vet) | PR 2 or 3 | Verification only, no new code |

## Phase 1: Performance Metrics — State & API

- [x] 1.1 Add `Metrics` struct to `internal/infrastructure/server/state.go` with fields: `CheckDurationMs int64`, `FilesScanned int`, `TotalDeps int`, `DetectorsRun int`, `UptimeSeconds int64` and JSON tags matching design.
- [x] 1.2 Add `metrics Metrics` field to `ServerState` struct in `state.go`.
- [x] 1.3 Update `SetCheckResult` signature in `state.go` to accept `metrics Metrics` parameter; store it atomically within the existing mutex lock.
- [x] 1.4 Add `Metrics() Metrics` getter to `ServerState` in `state.go` following the existing RWMutex pattern.
- [x] 1.5 Update `CacheData` struct in `state.go` to include `Metrics` field; update `SaveToFile` and `LoadFromFile` to persist/restore metrics.
- [x] 1.6 Add `MetricsResponse` struct to `internal/infrastructure/server/server.go` matching the JSON contract from the design.
- [x] 1.7 Add `handleMetrics` handler to `server.go` — GET returns JSON metrics via `state.Metrics()`, non-GET returns 405.
- [x] 1.8 Register `/api/metrics` route in `Server.Start()` mux in `server.go`.

## Phase 2: Performance Metrics — Check Cycle & Dashboard

- [x] 2.1 Update `RunCheck` in `server.go` to capture timing (`time.Since(start)`), count unique files from `deps`, count detectors run, and build a `Metrics` struct to pass to `SetCheckResult`.
- [x] 2.2 Add 4 metric cards HTML to `.summary-cards` grid in `internal/infrastructure/server/dashboard.html`: Check (ms), Files, Dependencies, Detectors — with `id` attributes `metric-duration`, `metric-files`, `metric-deps`, `metric-detectors`.
- [x] 2.3 Add CSS rule `.summary-card.metric .value` to `dashboard.html` style block: muted color, smaller font to distinguish from severity counts.
- [x] 2.4 Add `fetchOne('/api/metrics', 'metrics')` to `fetchData()` in `dashboard.html` JS, update `checkDone()` to read metrics JSON and populate the 4 card values (zero as fallback).

## Phase 3: Config Set Improvements

- [x] 3.1 Add `resolvePath(key string) ([]string, error)` helper to `cmd/arx/config.go` — splits dotted key, validates non-empty segments.
- [x] 3.2 Add `getAtPath(doc map[string]interface{}, path []string) (interface{}, error)` to `config.go` — navigates nested maps, returns error on missing path or type mismatch.
- [x] 3.3 Add `setAtPath(doc map[string]interface{}, path []string, value interface{}) error` to `config.go` — navigates/creates intermediate maps, sets value at leaf.
- [x] 3.4 Add `parseValue(raw string) (interface{}, error)` to `config.go` — tries `json.Unmarshal` first, falls back to raw string.
- [x] 3.5 Refactor `configSetCmd` in `config.go` to use `resolvePath` + `parseValue` + `setAtPath` instead of the hardcoded switch; keep backward compat for `max_violations`.
- [x] 3.6 Refactor `configGetCmd` in `config.go` to use `resolvePath` + `getAtPath` instead of the hardcoded switch; YAML-marshal complex types for display.
- [x] 3.7 Create `cmd/arx/config_test.go` with tests: `resolvePath` (top-level, nested, empty), `setAtPath` (creates intermediate maps, leaf set), `getAtPath` (nested navigation, missing path, type mismatch), `parseValue` (JSON array, JSON object, number, bool, string fallback).
- [x] 3.8 Add integration test in `config_test.go`: `arx config set severity_mapping.critical '["vendor/**"]'` → read arx.yaml → assert nested array; `arx config get severity_mapping.critical` → assert output.

## Phase 4: Metrics Tests

- [x] 4.1 Add test in `internal/infrastructure/server/state_test.go` (create if needed): `Metrics` JSON marshal/unmarshal round-trip, `Metrics()` getter thread safety, `SetCheckResult` stores metrics atomically.
- [x] 4.2 Add test in `internal/infrastructure/server/server_test.go`: `handleMetrics` returns 200 + valid JSON on GET, returns 405 on POST.
- [x] 4.3 Add test in `internal/infrastructure/server/dashboard_test.go`: parse rendered HTML, assert metric card element IDs exist in the DOM.

## Phase 5: Quality Pass

- [ ] 5.1 Run `go test -fuzz=FuzzConfigParse -fuzztime=10s ./internal/infrastructure/config/` — fix any crashes.
- [ ] 5.2 Run `go test -fuzz=FuzzPHPParse -fuzztime=10s ./internal/infrastructure/detector/php/` — fix any crashes.
- [ ] 5.3 Run `go test -fuzz=FuzzSwiftParse -fuzztime=10s ./internal/infrastructure/detector/swift/` — fix any crashes.
- [ ] 5.4 Run `go test -fuzz=FuzzKotlinParse -fuzztime=10s ./internal/infrastructure/detector/kotlin/` — fix any crashes.
- [ ] 5.5 Run `go test -fuzz=FuzzRubyParse -fuzztime=10s ./internal/infrastructure/detector/ruby/` — fix any crashes.
- [ ] 5.6 Run `go test -fuzz=FuzzRustParse -fuzztime=10s ./internal/infrastructure/detector/rust/` — fix any crashes.
- [ ] 5.7 Run `go test -fuzz=FuzzCSharpParse -fuzztime=10s ./internal/infrastructure/detector/csharp/` — fix any crashes.
- [ ] 5.8 Run `go test -fuzz=FuzzJavaParse -fuzztime=10s ./internal/infrastructure/detector/java/` — fix any crashes.
- [ ] 5.9 Run `go test -race ./...` — fix any data races.
- [ ] 5.10 Run `go vet ./...` — fix any vet issues.

## Phase 6: Polish & Verification

- [ ] 6.1 Update roadmap to mark metrics, config set improvements, and quality pass as complete.
- [ ] 6.2 Run full test suite `go test ./...` — all tests must pass.
