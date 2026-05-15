# Design: Arx v0.11.0 — CI/CD & HTML Quality

## Technical Approach

Six additive capabilities to improve CI/CD integration and output quality: (1) GitLab CI template with check and audit jobs producing JUnit and JSON artifacts, (2) pre-commit framework configuration for local hooks, (3) Docker multi-stage build with GitHub Actions publishing workflow, (4) HTML output renderer with embedded Go templates and CSS, (5) fuzz testing for config and parser robustness, (6) benchmark tests for performance-critical operations. All features are independent and can be implemented in parallel.

## Architecture Decisions

### Decision: GitLab CI template as static file (not CLI-generated)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `arx init gitlab-ci` command to generate | Adds CLI complexity, one-time setup | Rejected |
| Static `.gitlab-ci.yml` in project root | Simple, version-controlled, editable by users | **Chosen** |
| GitHub Actions only | Excludes GitLab users, limits adoption | Rejected |

Users need a copy-paste template. Static file is simpler and follows convention (like `.gitignore`). CLI generation adds unnecessary complexity for a one-time setup.

### Decision: JUnit report for `check`, JSON for `audit`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| JUnit for both | JUnit is test-centric; audit is inventory, not pass/fail | Rejected |
| JSON for both | JUnit has native GitLab integration for test reports | Rejected |
| JUnit for check (pass/fail), JSON for audit (structured data) | JUnit maps to violations as failures; JSON preserves full audit structure | **Chosen** |

`arx check` is a quality gate (pass/fail) — JUnit is ideal. `arx audit` produces rich data (coupling matrix, debt score, trends) — JSON preserves structure for downstream processing.

### Decision: pre-commit uses `language: system` (not Docker)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `language: docker` with arx image | Self-contained but slow (pull + startup per run) | Rejected |
| `language: golang` with repo build | Complex, duplicates build logic | Rejected |
| `language: system` requiring arx in PATH | Fast, assumes user has arx installed (reasonable for dev workflow) | **Chosen** |

pre-commit is a developer workflow tool. Developers should have arx installed locally. Docker overhead (pull, startup) makes hook sluggish. System language is instant.

### Decision: Multi-stage Dockerfile with distroless runtime

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Single-stage `golang:1.23-alpine` | Larger image (~100MB), includes build tools | Rejected |
| Multi-stage with `alpine` runtime | Small (~10MB) but requires libc, potential CVEs | Rejected |
| Multi-stage with `gcr.io/distroless/base` | Minimal (~20MB), no shell, security-hardened | **Chosen** |

Distroless images are security best practice: no shell, minimal attack surface. Arx is a static Go binary — no libc dependencies. Build stage uses official Go image for compatibility.

### Decision: HTML output with embedded `html/template` (not external files)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| External `.html` template files | Requires file system access, complicates distribution | Rejected |
| `embed` directive for template files | Adds build step, template loading logic | Rejected |
| Go string constants with `html/template` | Single binary, no external dependencies, compile-time checked | **Chosen** |

Arx should be a single binary. Embedding templates as Go strings ensures portability. `html/template` provides automatic escaping (XSS protection) and is part of stdlib.

### Decision: HTML sections mirror terminal audit report structure

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom HTML layout | Diverges from familiar terminal output | Rejected |
| Mirror terminal report sections (header, violations, coupling, debt, trends) | Consistent UX, users already understand structure | **Chosen** |
| Interactive dashboard (JavaScript) | Adds complexity, requires browser, overkill for CLI tool | Rejected |

Users know the terminal report structure. HTML should be a richer version of the same information, not a different UI. No JavaScript — keeps it simple, fast, and works in email/CI viewers.

### Decision: Fuzz tests for config and parser entry points only

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Fuzz every function | Excessive, most functions are trivial | Rejected |
| Fuzz public API entry points (ConfigParse, parser.Parse) | Tests real-world input, catches edge cases | **Chosen** |
| No fuzzing, only unit tests | Misses unexpected input combinations | Rejected |

Fuzzing is most valuable at system boundaries where untrusted input enters. Config YAML parsing and language-specific import parsers are the critical surfaces. Internal helper functions are exercised indirectly.

### Decision: Benchmark tests for hot paths only

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Benchmark every function | Noise, most functions are not performance-critical | Rejected |
| Benchmark coupling matrix, rule evaluation, parser extraction | These are O(n²) or called per-file in loops | **Chosen** |
| No benchmarks, profile when slow | Reactive, not proactive | Rejected |

Coupling matrix construction is O(n²) in layers. Rule evaluation runs per-violation. Parser extraction runs per-file. These are the hot paths. Benchmarks establish baselines and catch regressions.

## Data Flow

### GitLab CI template

```
.gitlab-ci.yml (static file)
  → stages: [check, audit]
  → check job:
      image: golang:1.23-alpine
      script: go build, ./arx check --format junit
      artifacts:
        reports:
          junit: arx-junit.xml
  → audit job:
      image: golang:1.23-alpine
      script: go build, ./arx audit --format json
      artifacts:
        paths: [arx-audit.json]
```

### pre-commit hook

```
.pre-commit-config.yaml (static file)
  → repo: local
    hooks:
      - id: arx-check
        entry: arx check --no-cache
        language: system
        types: [file]
        pass_filenames: false
```

### Docker publishing

```
Dockerfile (multi-stage)
  → Stage 1 (builder): golang:1.23-alpine
      → go build -o /arx ./cmd/arx
  → Stage 2 (runtime): gcr.io/distroless/base
      → COPY --from=builder /arx /arx
      → ENTRYPOINT ["/arx"]

.github/workflows/docker-publish.yml
  → on: push tags v*
  → jobs:
      build-and-push:
        → docker build -t ghcr.io/pauvalls/arx:${VERSION}
        → docker push ghcr.io/pauvalls/arx:${VERSION}
        → docker tag ...:latest
        → docker push ...:latest
```

### HTML output

```
cmd/arx/check.go or audit.go
  → --format html flag
  → HTMLReporter.Report(auditReport, OutputFormatHTML)

internal/infrastructure/output/html.go
  → htmlTemplate.Execute(w, auditReport)
  → Template sections:
      - {{template "header" .}}
      - {{template "violations" .}}
      - {{template "coupling" .}}
      - {{template "debt" .}}
      - {{template "trends" .}}
  → CSS embedded in <style> tag (Go string constant)
  → html/template auto-escapes all data
```

### Fuzz testing

```
internal/infrastructure/config/config_fuzz.go
  → FuzzConfigParse(f *testing.F)
    → f.Fuzz(func(t *testing.T, data []byte) {
        yamlReader.Read(bytes.NewReader(data))
        // Ignore errors — expected for random input
      })

internal/infrastructure/detector/{csharp,rust,java}/parser_fuzz.go
  → Fuzz{Language}Parse(f *testing.F)
    → f.Fuzz(func(t *testing.T, data []byte) {
        parseFile(string(data))
        // Ignore errors — expected for random input
      })
```

### Benchmark testing

```
internal/domain/coupling_bench_test.go
  → BenchmarkCouplingMatrix(b *testing.B)
    → Setup: create matrix with N layers
    → b.ResetTimer()
    → for i := 0; i < b.N; i++ {
        matrix.Add(from, to)
      }

internal/domain/audit_bench_test.go
  → BenchmarkRuleEvaluation(b *testing.B)
    → Setup: create rules, violations
    → b.ResetTimer()
    → for i := 0; i < b.N; i++ {
        evaluateRules(violations, rules)
      }

internal/infrastructure/detector/{java,csharp}/parser_bench_test.go
  → Benchmark{Language}Extraction(b *testing.B)
    → Setup: realistic source file content
    → b.ResetTimer()
    → for i := 0; i < b.N; i++ {
        ExtractImports(content, layers)
      }
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `.gitlab-ci.yml` | Create | GitLab CI template with check (JUnit) and audit (JSON) jobs |
| `.pre-commit-config.yaml` | Create | pre-commit hook configuration for arx-check |
| `Dockerfile` | Create | Multi-stage Docker build (golang → distroless) |
| `.github/workflows/docker-publish.yml` | Create | GitHub Actions workflow for Docker publishing on tags |
| `internal/infrastructure/output/html.go` | Create | HTML reporter with embedded templates and CSS |
| `internal/infrastructure/output/html_test.go` | Create | Tests: valid HTML5, empty report, full report, special chars escaping |
| `internal/infrastructure/config/config_fuzz.go` | Create | FuzzConfigParse for YAML config parser |
| `internal/infrastructure/detector/csharp/parser_fuzz.go` | Create | FuzzCSharpParse for C# import parser |
| `internal/infrastructure/detector/rust/parser_fuzz.go` | Create | FuzzRustParse for Rust import parser |
| `internal/infrastructure/detector/java/parser_fuzz.go` | Create | FuzzJavaParse for Java import parser |
| `internal/domain/coupling_bench_test.go` | Create | BenchmarkCouplingMatrix for dependency matrix operations |
| `internal/domain/audit_bench_test.go` | Create | BenchmarkRuleEvaluation for rule evaluation performance |
| `internal/infrastructure/detector/java/parser_bench_test.go` | Create | BenchmarkJavaExtraction for Java parser performance |
| `internal/infrastructure/detector/csharp/parser_bench_test.go` | Create | BenchmarkCSharpExtraction for C# parser performance |
| `internal/ports/reporter.go` | Modify | Add `OutputFormatHTML` constant |
| `cmd/arx/check.go` | Modify | Add `--format html` support (if not already present) |
| `cmd/arx/audit.go` | Modify | Add `--format html` support (if not already present) |

## Interfaces / Contracts

```go
// internal/ports/reporter.go
const (
    // ... existing formats ...
    OutputFormatHTML OutputFormat = "html"
)

// internal/infrastructure/output/html.go
type HTMLReporter struct {
    tool    string
    version string
}

func NewHTMLReporter() *HTMLReporter

func (r *HTMLReporter) Report(report domain.AuditReport, format ports.OutputFormat) error
    // Renders complete HTML document:
    // <!DOCTYPE html>
    // <html>
    //   <head>
    //     <style>{{cssStyles}}</style>
    //   </head>
    //   <body>
    //     {{template "header" .}}
    //     {{template "violations" .}}
    //     {{template "coupling" .}}
    //     {{template "debt" .}}
    //     {{template "trends" .}}
    //   </body>
    // </html>

// CSS embedded as Go string constant
const htmlStyles = `
    :root {
        --color-error: #dc3545;
        --color-warning: #ffc107;
        --color-info: #17a2b8;
        --color-success: #28a745;
    }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .violation { border-left: 4px solid var(--color-error); }
    .violation.warning { border-left-color: var(--color-warning); }
    .matrix-table { border-collapse: collapse; }
    .matrix-table td { border: 1px solid #ddd; padding: 8px; }
    /* ... more styles ... */
`

// HTML template (simplified)
var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
    "escapeHTML": html.EscapeString,
}).Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Arx Architecture Report</title>
    <style>{{.Styles}}</style>
</head>
<body>
    <header>
        <h1>Architecture Audit Report</h1>
        <p>Project: {{.ProjectRoot}}</p>
        <p>Date: {{.Timestamp.Format "2006-01-02 15:04"}}</p>
    </header>

    <section id="violations">
        <h2>Violations ({{len .Violations}})</h2>
        {{range .Violations}}
        <div class="violation {{.Severity}}">
            <strong>{{.File}}:{{.Line}}</strong>
            <p>{{.SourceLayer}} → {{.TargetLayer}}</p>
            <p>{{.Message}}</p>
        </div>
        {{end}}
    </section>

    <section id="coupling">
        <h2>Coupling Matrix</h2>
        <table class="matrix-table">
            <thead>
                <tr><th>From</th><th>To</th><th>Count</th><th>%</th></tr>
            </thead>
            <tbody>
            {{range $from, $targets := .CouplingMatrix.FromTo}}
                {{range $to, $count := $targets}}
                <tr>
                    <td>{{$from}}</td>
                    <td>{{$to}}</td>
                    <td>{{$count}}</td>
                    <td>{{printf "%.1f" (div $count $.TotalCoupling)}}%</td>
                </tr>
                {{end}}
            {{end}}
            </tbody>
        </table>
    </section>

    <section id="debt">
        <h2>Technical Debt</h2>
        <p>Score: <strong>{{.DebtScore.Total}}</strong></p>
        <p>Trend: {{.DebtScore.Trend}} ({{.DebtScore.TrendDelta}})</p>
    </section>

    {{if .TrendReport.Status}}
    <section id="trends">
        <h2>Trend Report</h2>
        <p>Status: {{.TrendReport.Status}}</p>
        <p>{{.TrendReport.Summary}}</p>
    </section>
    {{end}}
</body>
</html>
`))

// internal/infrastructure/config/config_fuzz.go
func FuzzConfigParse(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        reader := config.NewYAMLReader()
        // Fuzz target: parsing should not panic
        // Errors are expected for random input
        _, _ = reader.ReadFrom(bytes.NewReader(data))
    })
}

// internal/infrastructure/detector/csharp/parser_fuzz.go
func FuzzCSharpParse(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        content := string(data)
        // Fuzz target: parsing should not panic
        extractImportsFromLine(content)
        // Errors/empty results are expected for random input
    })
}

// internal/infrastructure/detector/rust/parser_fuzz.go
func FuzzRustParse(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        content := string(data)
        extractImportsFromLine(content)
    })
}

// internal/infrastructure/detector/java/parser_fuzz.go
func FuzzJavaParse(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        content := string(data)
        extractImportsFromLine(content)
    })
}

// internal/domain/coupling_bench_test.go
func BenchmarkCouplingMatrix(b *testing.B) {
    // Setup: realistic layer count (5-10 layers)
    layers := []string{"domain", "application", "infrastructure", "presentation", "api"}
    matrix := domain.NewCouplingMatrix()

    // Pre-populate with some data
    for _, from := range layers {
        for _, to := range layers {
            if from != to {
                matrix.Add(from, to)
            }
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Benchmark Add operation
        matrix.Add(layers[i%len(layers)], layers[(i+1)%len(layers)])
    }
}

// internal/domain/audit_bench_test.go
func BenchmarkRuleEvaluation(b *testing.B) {
    // Setup: realistic violation count (100-1000 violations)
    violations := make([]domain.Violation, 500)
    for i := range violations {
        violations[i] = domain.Violation{
            ID:           fmt.Sprintf("viol-%d", i),
            File:         "file.go",
            Line:         i,
            SourceLayer:  "domain",
            TargetLayer:  "presentation",
            Severity:     domain.SeverityError,
            Message:      "Dependency violation",
        }
    }

    rules := []domain.Rule{
        {Name: "no-domain-to-presentation", Source: "domain", Target: "presentation"},
        // ... more rules ...
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Benchmark rule matching
        for _, v := range violations {
            for _, rule := range rules {
                _ = v.SourceLayer == rule.Source && v.TargetLayer == rule.Target
            }
        }
    }
}

// internal/infrastructure/detector/java/parser_bench_test.go
func BenchmarkJavaExtraction(b *testing.B) {
    // Setup: realistic Java file content
    content := `
package com.example.app;
import java.util.List;
import java.util.ArrayList;
import com.example.domain.Order;
import com.example.infrastructure.Database;
import static java.lang.Math.PI;
import static org.junit.Assert.assertEquals;

public class OrderService {
    // ... class content ...
}
`
    layers := []domain.Layer{
        {Name: "domain", Path: "com.example.domain.*"},
        {Name: "infrastructure", Path: "com.example.infrastructure.*"},
        {Name: "application", Path: "com.example.app.*"},
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Benchmark full extraction
        _, _ = java.ExtractImports(content, layers)
    }
}

// internal/infrastructure/detector/csharp/parser_bench_test.go
func BenchmarkCSharpExtraction(b *testing.B) {
    // Setup: realistic C# file content
    content := `
using System;
using System.Collections.Generic;
using Microsoft.EntityFrameworkCore;
using MyApp.Domain.Entities;
using MyApp.Infrastructure.Repositories;
using static System.Math;

namespace MyApp.Application.Services {
    public class OrderService {
        // ... class content ...
    }
}
`
    layers := []domain.Layer{
        {Name: "domain", Path: "MyApp.Domain.*"},
        {Name: "infrastructure", Path: "MyApp.Infrastructure.*"},
        {Name: "application", Path: "MyApp.Application.*"},
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = csharp.ExtractImports(content, layers)
    }
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | HTML output valid HTML5 | Parse with `golang.org/x/net/html`, verify no errors |
| Unit | HTML empty report | AuditReport with 0 violations → valid HTML, empty sections |
| Unit | HTML full report | AuditReport with violations, coupling, debt, trends → all sections populated |
| Unit | HTML special chars escaping | Violations with `<script>`, `&`, `"` → escaped in output, no XSS |
| Unit | CSS embedded correctly | `<style>` tag present, CSS valid (basic syntax check) |
| Unit | GitLab CI YAML valid | Parse with `gopkg.in/yaml.v3`, verify structure |
| Unit | pre-commit YAML valid | Parse with `gopkg.in/yaml.v3`, verify hooks structure |
| Unit | Dockerfile syntax | `docker build --no-push` (dry run) or lint with `hadolint` |
| Unit | GitHub workflow YAML valid | Parse with `gopkg.in/yaml.v3`, verify triggers, jobs |
| Fuzz | Config parser | FuzzConfigParse: random bytes → no panic (errors OK) |
| Fuzz | C# parser | FuzzCSharpParse: random bytes → no panic |
| Fuzz | Rust parser | FuzzRustParse: random bytes → no panic |
| Fuzz | Java parser | FuzzJavaParse: random bytes → no panic |
| Benchmark | Coupling matrix ops | Report ops/ns for Add, Get, Count operations |
| Benchmark | Rule evaluation | Report ops/ns for matching violations against rules |
| Benchmark | Java extraction | Report files/s for full Java file parsing |
| Benchmark | C# extraction | Report files/s for full C# file parsing |
| Integration | GitLab CI template | Run in GitLab CI (or local `gitlab-ci-local`), verify artifacts |
| Integration | pre-commit hook | Run `pre-commit run arx-check`, verify execution |
| Integration | Docker build | `docker build .`, verify image runs `arx --version` |
| Integration | HTML output end-to-end | `arx audit --format html`, verify output renders in browser |

Fuzz tests run with `go test -fuzz=FuzzConfigParse -fuzztime=10s`. Benchmarks run with `go test -bench=. -benchmem`.

## Migration / Rollout

No migration required. All features are additive:
- `.gitlab-ci.yml` is opt-in — users copy template to their repo
- `.pre-commit-config.yaml` is opt-in — users install pre-commit and configure
- `Dockerfile` and workflow are opt-in — users build/push manually or enable workflow
- HTML output is selected via `--format html` flag — existing formats unchanged
- Fuzz tests are new test files — no production code impact
- Benchmark tests are new test files — run only with `-bench` flag

## Open Questions

- [x] Should GitLab CI template include cache configuration? **Yes** — add `cache:` for Go modules and arx cache (`.arx-cache/`). Speeds up subsequent runs.
- [x] Should pre-commit hook run on all files or only changed files? **Changed files** — pre-commit default is `types: [file]` with git staging. Faster, standard behavior.
- [x] Should Docker workflow publish to Docker Hub in addition to GHCR? **No** — GHCR is sufficient, simpler. Users can retag if needed.
- [x] Should HTML output include inline CSS or external stylesheet? **Inline** — Single file output, works offline, no external dependencies.
- [x] Should fuzz tests run in CI? **Yes** — add `go test -fuzz=FuzzConfigParse -fuzztime=30s` to CI, but short duration (30s) to avoid slowing pipeline.
- [x] Should benchmark tests run in CI? **No** — benchmarks are for local development. CI runs unit tests only. Document `make bench` for local use.
- [x] Should HTML template support custom themes? **No** — Keep v0.11.0 scoped. Single theme is sufficient. Add theming in future version if requested.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| HTML template XSS vulnerability | High — malicious content in file paths could execute scripts | Use `html/template` (not `text/template`) — automatic escaping. Test with `<script>` payloads. |
| Docker image too large | Medium — slow pulls, storage costs | Use distroless base (~20MB). Multi-stage build ensures no build tools in final image. |
| Fuzz tests slow down CI | Medium — pipeline timeouts | Limit fuzz time to 30s in CI. Run full fuzz (10m+) locally only. |
| GitLab CI template outdated | Low — users copy once, may not update | Document template as "starting point". Users own their `.gitlab-ci.yml`. |
| pre-commit hook fails if arx not installed | Low — user error, not tool bug | Document requirement: `arx` must be in PATH. Provide install instructions in README. |
| HTML output not rendering in some email clients | Low — HTML emails are notoriously broken | Design for browser viewing. Email is secondary. Use simple, semantic HTML. |

(End of file)
