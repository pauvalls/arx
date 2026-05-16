# Tasks: Arx v0.19.0 — Extended Expression Functions

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–600 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (T1–T3 + T4–T9) → PR 2 (T10–T14) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Expression engine + TypeScript quick wins | PR 1 | Core domain changes; all expr tests included |
| 2 | HTML audit report + docs | PR 2 | Depends on PR 1; reporter interface changes |

---

## Phase 1: Quick Wins (parallel)

- [ ] 1.1 TypeScript detector — add `import type { X } from "..."` pattern to `extractImportPaths()` in `internal/infrastructure/detector/typescript/detector.go`
- [ ] 1.2 TypeScript detector — add `export { X } from "..."` re-export pattern
- [ ] 1.3 TypeScript detector — add dynamic `import("...")` pattern
- [ ] 1.4 Write unit tests for new TypeScript patterns in `internal/infrastructure/detector/typescript/detector_test.go`
- [ ] 1.5 Run existing `test/integration/detector_test.go` to verify no regressions
- [ ] 1.6 Verify Python E2E fixture at `test/fixtures/python-project/` passes via `go test ./test/integration/... -run Python`

## Phase 2: Extended Expression Functions

- [ ] 2.1 Extend `EvalContext` in `internal/domain/expr.go` to include `Violations []Violation` field
- [ ] 2.2 Update `ruleCheckMatches()` signature and all call sites to pass violations context
- [ ] 2.3 Add `files(layer string)` builtin to `expr.go` — returns count of files in layer
- [ ] 2.4 Add `ratio(count, total)` builtin — returns percentage as int (0–100)
- [ ] 2.5 Add `violations(ruleID string)` builtin — returns count of violations for given rule ID
- [ ] 2.6 Add `threshold(value, min, max)` builtin — returns bool if value within range
- [ ] 2.7 Register all four new builtins in the `builtins` map
- [ ] 2.8 Update `EvaluateRules()` in `internal/domain/audit.go` to pass violations into expression evaluation context
- [ ] 2.9 Write unit tests for `files()`, `ratio()`, `violations()`, `threshold()` in `internal/domain/expr_test.go`
- [ ] 2.10 Write integration test for expression rules using new functions in `test/integration/audit_test.go`

## Phase 3: HTML Report Improvements

- [ ] 3.1 Modify `HTMLReporter.Report()` signature in `internal/infrastructure/output/html.go` to accept `*domain.AuditReport` instead of `[]domain.Violation`
- [ ] 3.2 Update `internal/ports/reporter.go` `Reporter` interface — add new `ReportAudit(report *domain.AuditReport, format OutputFormat) error` method (or extend existing)
- [ ] 3.3 Wire coupling matrix data into HTML template — populate `CouplingRows` from `report.CouplingMatrix`
- [ ] 3.4 Wire debt score data into HTML template — populate `DebtScore` from `report.DebtScore`
- [ ] 3.5 Wire trend report data into HTML template — populate `TrendReport` from `report.TrendReport`
- [ ] 3.6 Extend HTML template sections for coupling matrix, debt score, and trends
- [ ] 3.7 Update `renderHTML()` in `cmd/arx/audit.go` to pass full `AuditReport` to HTML reporter
- [ ] 3.8 Update `renderAuditReport()` to use new report-aware HTML path for `ports.OutputFormatHTML`
- [ ] 3.9 Write tests for HTML audit report rendering in `internal/infrastructure/output/html_test.go` — verify coupling/debt/trend sections appear
- [ ] 3.10 Update existing HTML reporter tests to use new signature or adapter

## Phase 4: Polish

- [ ] 4.1 Update `docs/roadmap.md` — mark v0.19.0 features as completed
- [ ] 4.2 Update `docs/configuration.md` — document new expression functions with examples
- [ ] 4.3 Update `docs/output-formats.md` — document HTML report sections (coupling, debt, trends)
- [ ] 4.4 Update `CHANGELOG.md` with v0.19.0 entry
- [ ] 4.5 Run full test suite: `go test ./...`
- [ ] 4.6 Run integration tests: `go test ./test/integration/...`
- [ ] 4.7 Verify build: `go build ./cmd/arx`
- [ ] 4.8 Verify linting: `golangci-lint run` (if configured)

---

## Dependency Notes

- Phase 2 depends on Phase 1 only for parallelization hygiene (no actual code deps)
- Phase 3 depends on Phase 2 because `violations()` builtin requires violation context plumbing
- Phase 4 depends on all previous phases
- `EvalContext` change (2.1) is a breaking internal change — must update all call sites before tests pass
- HTML reporter interface change (3.2) affects `ports.Reporter` — check if other reporters need adapter stubs
