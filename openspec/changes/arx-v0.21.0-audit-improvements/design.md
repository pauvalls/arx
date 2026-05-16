# Design: Arx v0.21.0 — Audit Report Improvements

## Technical Approach

Wire full `domain.AuditReport` data into HTML and JSON reporters without changing existing `ports.Reporter` interface (additive-only constraint from proposal). Add a new `ReportAudit` method on `HTMLReporter` and `JSONReporter` that accepts `*domain.AuditReport`, then update `cmd/arx/audit.go` to call it. For `cmd/arx/check.go`, extend `JSONReporter` to accept optional detector statuses and coupling metadata via a new `ReportWithContext` method. Run `go vet`, fuzz tests, and deprecated API audit as a quality gate.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Reporter interface | Keep `ports.Reporter` unchanged | Extend with `ReportAudit` | Proposal requires additive-only, no breaking changes |
| HTML audit data | Add `ReportAudit` method on `HTMLReporter` | Pass through generic `interface{}` | Type-safe, explicit, follows Go conventions |
| JSON check data | Add `ReportWithContext` method on `JSONReporter` | Add all fields to every JSON output | Check JSON should include audit metadata only when available |
| HTML template sections | Reuse existing template struct fields | Rewrite template from scratch | `htmlReportData` already has `CouplingRows`, `DebtScore`, `TrendReport` — just unpopulated |

## Data Flow

```
AuditReport (domain)
    ├── CouplingMatrix ──→ HTMLReporter.ReportAudit ──→ template (coupling table)
    ├── DebtScore ───────→ HTMLReporter.ReportAudit ──→ template (debt section)
    ├── TrendReport ─────→ HTMLReporter.ReportAudit ──→ template (trend section)
    └── Violations ──────→ HTMLReporter.ReportAudit ──→ template (violations)

AuditReport (domain)
    ├── CouplingMatrix ──→ JSONReporter.ReportAudit ──→ JSONOutput.CouplingMatrix
    ├── DebtScore ───────→ JSONReporter.ReportAudit ──→ JSONOutput.DebtScore
    ├── TrendReport ─────→ JSONReporter.ReportAudit ──→ JSONOutput.TrendReport
    └── Violations ──────→ JSONReporter.ReportAudit ──→ JSONOutput.Violations

checkResult (cmd/arx/check.go)
    └── detectorStatuses ──→ JSONReporter.ReportWithContext ──→ JSONOutput.Detectors
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/ports/reporter.go` | No change | Keep interface stable; new methods on concrete types only |
| `internal/infrastructure/output/html.go` | Modify | Add `ReportAudit` method; populate `CouplingRows`, `DebtScore`, `TrendReport` in template data; extend template with coupling table, debt breakdown, trend sections |
| `internal/infrastructure/output/html_test.go` | Modify | Add tests for `ReportAudit` with coupling matrix, debt score, trend data |
| `internal/infrastructure/output/json.go` | Modify | Add `ReportAudit` and `ReportWithContext` methods; extend `JSONOutput` with `CouplingMatrix`, `DebtScore`, `TrendReport`, `Detectors` fields |
| `cmd/arx/audit.go` | Modify | `renderHTML` calls `HTMLReporter.ReportAudit(report)` instead of `Report(report.Violations, ...)` |
| `cmd/arx/check.go` | Modify | `printCheckResult` passes `detectorStatuses` to `JSONReporter.ReportWithContext` when format is JSON |

## Interfaces / Contracts

```go
// internal/infrastructure/output/html.go

// ReportAudit renders a full audit report including coupling, debt, and trends.
func (r *HTMLReporter) ReportAudit(report *domain.AuditReport) error

// internal/infrastructure/output/json.go

// JSONOutput extended fields
type JSONOutput struct {
    // ... existing fields ...
    CouplingMatrix domain.CouplingMatrix `json:"coupling_matrix,omitempty"`
    DebtScore      domain.DebtScore      `json:"debt_score,omitempty"`
    TrendReport    domain.TrendReport    `json:"trend_report,omitempty"`
    Detectors      []DetectorInfo        `json:"detectors,omitempty"`
}

// DetectorInfo mirrors application.DetectorStatus for JSON serialization
type DetectorInfo struct {
    Name       string `json:"name"`
    Applicable bool   `json:"applicable"`
    DepCount   int    `json:"dep_count"`
    Error      string `json:"error,omitempty"`
}

// ReportAudit renders full audit report as JSON.
func (r *JSONReporter) ReportAudit(report *domain.AuditReport) error

// ReportWithContext renders check results with optional detector metadata.
func (r *JSONReporter) ReportWithContext(violations []domain.Violation, detectors []application.DetectorStatus) error
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | HTML `ReportAudit` renders coupling table with color-coded cells | Assert HTML contains expected `<tr>`/`<td>` with correct CSS classes |
| Unit | HTML `ReportAudit` renders debt score and breakdown | Assert HTML contains debt total and severity breakdown |
| Unit | HTML `ReportAudit` renders trend section | Assert HTML contains trend status and delta values |
| Unit | JSON `ReportAudit` includes coupling_matrix, debt_score, trend_report | Unmarshal output and assert fields present |
| Unit | JSON `ReportWithContext` includes detectors array | Unmarshal output and assert detectors field |
| Integration | `cmd/arx/audit.go --format html` produces complete report | Run command, capture stdout, validate HTML structure |
| Integration | `cmd/arx/check.go --format json` includes detector metadata | Run with `--verbose`, capture stdout, validate JSON fields |
| Quality | `go vet ./...` returns 0 | Run as pre-verify step |
| Quality | All fuzz targets run 5s with 0 crashes | Run `go test -fuzz=Fuzz -fuzztime=5s ./...` |

## Migration / Rollout

No migration required. All changes are additive:
- New methods on reporters; old `Report` method behavior unchanged
- New JSON fields are `omitempty`; existing consumers unaffected
- HTML template additions are new sections; existing violation-only output still works

## Open Questions

- [ ] Should `JSONReporter.ReportWithContext` also accept coupling/debt data for `check` command, or is detector metadata sufficient? Proposal says "pass audit context to JSON reporter" — need clarification on scope.
- [ ] Trend HTML section: should it show new/resolved violation counts derived from `TrendReport.ViolationDelta`, or is delta sufficient?
