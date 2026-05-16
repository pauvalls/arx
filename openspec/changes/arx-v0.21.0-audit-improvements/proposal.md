# Proposal: Arx v0.21.0 — Audit Report Improvements

## Intent

Arx v0.20.0 delivers a complete audit pipeline that computes coupling matrices, debt scores, and trend reports — but these never reach the user. `arx audit --format html` only renders violations, and `arx check --format json` omits all audit metadata. This release wires the full audit data into both reporters and runs a quality pass to keep the codebase clean.

## Scope

### In Scope
- Full HTML audit report: coupling matrix table, debt score with breakdown, trend report section
- JSON check output: coupling matrix, debt score, detector metadata
- `go vet ./...` across all packages, fix any issues
- Run all fuzz tests for 5s each, verify 0 crashes
- Check for deprecated API usage

### Out of Scope
- New language detectors
- New CLI commands or flags
- Changes to expression syntax or rule evaluation
- Breaking changes to existing JSON/HTML field structure (additive only)

## Capabilities

### New Capabilities
- None — all changes extend existing reporters

### Modified Capabilities
- `html-audit-report`: Extend HTML reporter to accept full `AuditReport` and render coupling matrix, debt score, and trend sections
- `json-check-output`: Extend JSON reporter to include coupling matrix, debt score, and detector metadata

## Approach

1. Extend reporter interface or add `ReportAudit` method that accepts `*domain.AuditReport`
2. Update `HTMLReporter` template and data structs to render coupling table, debt breakdown, and trend summary
3. Update `JSONReporter` output struct to include `coupling_matrix`, `debt_score`, and `detectors` fields
4. Update `cmd/arx/audit.go` `renderHTML` to pass full report instead of only violations
5. Update `cmd/arx/check.go` to pass audit context to JSON reporter when available
6. Run `go vet ./...`, fix all reported issues
7. Run all fuzz targets with `-fuzztime=5s`, confirm 0 crashes
8. Audit imports for deprecated Go standard library API usage

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/ports/reporter.go` | Modified | Extend Reporter interface for full AuditReport support |
| `internal/infrastructure/output/html.go` | Modified | Wire coupling, debt, trends into template and data structs |
| `internal/infrastructure/output/html_test.go` | Modified | Add tests for new HTML sections |
| `internal/infrastructure/output/json.go` | Modified | Add audit metadata to JSON output struct |
| `cmd/arx/audit.go` | Modified | Pass full AuditReport to HTML reporter |
| `cmd/arx/check.go` | Modified | Pass audit context to JSON reporter |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| JSON schema change breaks consumers | Low | Changes are purely additive; existing fields unchanged |
| HTML template bloat | Low | Keep CSS minimal; reuse existing design system |
| `go vet` reveals significant issues | Low | v0.20.0 is already clean; expect minor fixes only |

## Rollback Plan

All changes are additive and isolated:
- Revert reporter changes — HTML/JSON fall back to violation-only output
- Revert `go vet` fixes — minor, low-risk
- Revert any fuzz infrastructure tweaks — no production code changes

Revert individual commits. No migrations, no data changes, no breaking changes.

## Dependencies

- None

## Success Criteria

- [ ] HTML report shows coupling matrix table with layer-to-layer dependency counts
- [ ] HTML report shows debt score with severity breakdown
- [ ] HTML report shows trend section (new/resolved violations, debt delta)
- [ ] JSON output includes `coupling_matrix`, `debt_score`, and `detectors` fields
- [ ] `go vet ./...` returns 0 issues
- [ ] All fuzz tests run for 5s with 0 crashes
- [ ] All 30+ test packages continue passing
