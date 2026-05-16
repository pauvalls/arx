# Design: Arx v0.16.0 — Audit HTML Output, Expanded Presets, Completion Instructions, Quality Pass

## Technical Approach

Four independent work streams touching CLI commands, embedded preset assets, and a code quality fix. All changes are additive or corrective — no breaking changes to existing interfaces.

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|----------|---------|----------|----------|
| HTML output to file | Fix `HTMLReporter.Report()` to accept writer vs add new `ReportTo()` method | `Report()` signature comes from `ports.Reporter` interface — changing it ripples through all reporters. `renderHTML` already receives `io.Writer` but ignores it. Best fix: make `renderHTML` use the writer directly instead of delegating to `Report()` | Extend `renderHTML` to render template to the passed `io.Writer` instead of calling `reporter.Report()` which hardcodes `os.Stdout` |
| Preset storage | Separate YAML files vs single multi-doc YAML | Existing convention uses one YAML per preset with `//go:embed *.yaml`. Follow existing pattern. | One YAML file per preset, placed alongside existing presets |
| Preset validation | Single source of truth vs duplicate lists in `init.go` and `presets.go` | `init.go` hardcodes `[]string{"clean", "hexagonal", "ddd"}` while `presets.go` has `AvailablePresets()`. Duplication is a bug waiting to happen. | Replace hardcoded list in `init.go` with call to `application.AvailablePresets()` |
| sortStrings replacement | `sort.Strings()` vs keep custom bubble sort | Custom `sortStrings()` is O(n²) bubble sort — unnecessary when stdlib provides optimized `sort.Strings()`. | Replace with `sort.Strings()`, add `"sort"` import, remove `sortStrings()` function |

## Data Flow

### audit-html-output
```
runAudit()
  ├── -o flag → os.Create(auditOutput) → out = file
  └── renderAuditReport(out, report, format, ...)
        └── renderReport(out, report, ports.OutputFormatHTML)
              └── renderHTML(out, report)
                    ├── NEW: render HTML template to `out` (not os.Stdout)
                    └── include full AuditReport data (coupling, debt, trends)
```

### init-presets-expanded
```
arx init --preset layered
  ├── runInit() validates preset via application.AvailablePresets()
  └── service.InitWithPreset("layered", ...)
        ├── LoadPreset("layered") → reads embedded layered.yaml
        └── ApplyPreset(template, projectRoot) → domain.Config
```

### completion-instructions
```
arx completion --help
  └── completionCmd.Long (enhanced with per-shell install instructions)
```

### quality-pass
```
dot.go: sortStrings(layerNames)  →  sort.Strings(layerNames)
         (remove sortStrings function entirely)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/infrastructure/output/html.go` | Modify | Add `ReportTo(w io.Writer, violations []domain.Violation)` method that renders to any writer; keep `Report()` for interface compatibility |
| `cmd/arx/audit.go` | Modify | Update `renderHTML()` to call `reporter.ReportTo(out, ...)` instead of `reporter.Report()` which writes to stdout |
| `internal/application/presets/layered.yaml` | Create | Layered architecture preset: presentation, business, persistence, infrastructure layers |
| `internal/application/presets/onion.yaml` | Create | Onion architecture preset: domain, application, infrastructure, ports layers |
| `internal/application/presets/presets.go` | Modify | Add `"layered"`, `"onion"` to `AvailablePresets()`; add descriptions in `getPresetDescription()` |
| `cmd/arx/init.go` | Modify | Replace hardcoded preset list with `application.AvailablePresets()` call; update `--preset` flag help text |
| `cmd/arx/completion.go` | Modify | Enhance `completionCmd.Long` with structured per-shell install instructions |
| `internal/infrastructure/output/dot.go` | Modify | Replace `sortStrings()` call with `sort.Strings()`; add `"sort"` import; remove `sortStrings()` function |

## Interfaces / Contracts

**HTMLReporter extended** (`internal/infrastructure/output/html.go`):

```go
// ReportTo renders violations as HTML to the given writer (not just stdout).
func (r *HTMLReporter) ReportTo(w io.Writer, violations []domain.Violation) error {
    // Same logic as Report() but uses w instead of os.Stdout
    // ...
    if err := htmlTemplate.Execute(w, data); err != nil {
        return fmt.Errorf("executing HTML template: %w", err)
    }
    return nil
}
```

**renderHTML updated** (`cmd/arx/audit.go`):

```go
func renderHTML(out io.Writer, report *domain.AuditReport) error {
    reporter := output.NewHTMLReporter()
    return reporter.ReportTo(out, report.Violations)
}
```

**AvailablePresets** (`internal/application/presets/presets.go`):

```go
func AvailablePresets() []string {
    return []string{"clean", "hexagonal", "ddd", "layered", "onion"}
}
```

**init.go preset validation** — replace hardcoded list:

```go
// Before:
validPresets := []string{"clean", "hexagonal", "ddd"}

// After:
validPresets := application.AvailablePresets()
```

**dot.go sort fix** — replace custom bubble sort:

```go
// Before:
sortStrings(layerNames)

// After (add "sort" to imports):
sort.Strings(layerNames)

// Remove the sortStrings function entirely (lines 157-166)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit — HTML | `ReportTo()` writes to buffer, not stdout | `bytes.Buffer` + compare output contains expected HTML |
| Unit — HTML | `renderHTML` in audit.go uses passed writer | Table-driven with `io.Writer` mock |
| Unit — Presets | `LoadPreset("layered")` returns valid config | Parse YAML, verify 4 layers exist |
| Unit — Presets | `LoadPreset("onion")` returns valid config | Parse YAML, verify 4 layers exist |
| Unit — Presets | `AvailablePresets()` includes all 5 presets | Assert slice length and membership |
| Unit — Init | `arx init --preset layered` generates config | Temp dir + check generated YAML |
| Unit — Init | `arx init --preset unknown` returns error with all presets listed | Assert error message contains "layered, onion" |
| Unit — Completion | `arx completion --help` shows shell instructions | Assert help text contains "bash", "zsh", "fish", "powershell" |
| Unit — dot.go | `GenerateDOT` output is deterministic (sorted layers) | Run twice, compare output equality |
| Build | `go build ./...` succeeds | CI build |
| Test | `go test ./...` passes | Full test suite |

## Migration / Rollout

No migration required. All changes are additive or corrective:
- HTML output: existing `-f html` behavior unchanged; `-o` flag now works correctly with HTML (previously wrote to stdout regardless)
- Presets: new presets are opt-in via `--preset` flag; existing presets unchanged
- Completion: help text enhancement only; no functional change
- sortStrings: internal implementation detail; no API change

## Open Questions

- [ ] Should `renderHTML` eventually include coupling matrix, debt score, and trend data in the HTML output? Currently `renderHTML` only passes `report.Violations` to the HTML reporter. The HTML template already has fields for coupling, debt, and trend but they are not populated. This is out of scope for this change but worth noting as a TODO.
- [ ] Should the `Report()` interface method be updated to accept `io.Writer` instead of hardcoding stdout? This would be a broader refactor affecting all reporters. Deferred — `ReportTo()` is a pragmatic addition without breaking the interface.
