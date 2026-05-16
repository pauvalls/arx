# Tasks: Arx v0.21.0 — Audit Report Improvements

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350 (HTML reporter/template/tests, JSON reporter/tests, cmd wiring, vet fixes) |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR — all changes are additive and tightly coupled around reporter wiring |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium

## Phase 1: HTML Audit Report

- [ ] 1.1 Add `ReportAudit(report *domain.AuditReport) error` to `internal/infrastructure/output/html.go`
- [ ] 1.2 Extend `htmlReportData` to populate `CouplingRows`, `DebtScore`, and `TrendReport` fields
- [ ] 1.3 Extend the HTML template with a coupling matrix table (From, To, Count, Percentage)
- [ ] 1.4 Extend the HTML template with a debt score section (total + severity breakdown)
- [ ] 1.5 Extend the HTML template with a trend section (status, new/resolved deltas)
- [ ] 1.6 Modify `cmd/arx/audit.go` `renderHTML` to call `HTMLReporter.ReportAudit(report)` instead of `Report(report.Violations, ...)`
- [ ] 1.7 Add unit tests in `html_test.go` asserting `ReportAudit` output contains coupling table rows, debt score, and trend status

## Phase 2: JSON Check Improvements

- [ ] 2.1 Extend `JSONOutput` in `internal/infrastructure/output/json.go` with `CouplingMatrix`, `DebtScore`, `TrendReport`, and `Detectors` fields (all `omitempty`)
- [ ] 2.2 Add `DetectorInfo` struct (Name, Applicable, DepCount, Error) to `json.go`
- [ ] 2.3 Add `ReportAudit(report *domain.AuditReport) error` to `JSONReporter`
- [ ] 2.4 Add `ReportWithContext(violations []domain.Violation, detectors []application.DetectorStatus) error` to `JSONReporter`
- [ ] 2.5 Modify `cmd/arx/check.go` `printCheckResult` to pass `detectorStatuses` to `JSONReporter.ReportWithContext` when `--format json`
- [ ] 2.6 Add unit tests in `json_test.go` asserting `ReportAudit` includes `coupling_matrix`, `debt_score`, `trend_report`, and `ReportWithContext` includes `detectors`

## Phase 3: Quality Pass

- [ ] 3.1 Run `go vet ./...`, fix all warnings across packages
- [ ] 3.2 Run all fuzz tests with `-fuzztime=5s`, verify zero crashes
- [ ] 3.3 Search codebase for deprecated API usage (`strings.Title`, `filepath.HasPrefix`), replace or suppress

## Phase 4: Polish

- [ ] 4.1 Update `CHANGELOG.md` and roadmap with v0.21.0 audit improvements
- [ ] 4.2 Run full test suite (`go test ./...`), confirm all 30+ packages pass
