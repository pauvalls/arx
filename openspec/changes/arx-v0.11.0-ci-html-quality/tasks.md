# Implementation Tasks: Arx v0.11.0 — CI/CD & HTML Quality

**Change ID:** `arx-v0.11.0-ci-html-quality`
**Status:** Ready for implementation
**Estimated effort:** 3-4 days

---

## Phase 1: CI/CD Templates (No Go Changes — Quick Wins)

### T1: Create `.gitlab-ci.yml` Template
**File:** `.gitlab-ci.yml`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Static GitLab CI template with two jobs:
- `check`: Runs `arx check` with JUnit output, uploads to GitLab test reports
- `audit`: Runs `arx audit` with JSON output, saves as artifact

**Requirements:**
- Use `golang:1.23-alpine` image
- Include cache for Go modules and `.arx-cache/`
- `check` job produces `arx-junit.xml` → GitLab JUnit report
- `audit` job produces `arx-audit.json` → downloadable artifact
- Both jobs run on `push` and `merge_request` events

**Acceptance Criteria:**
- [ ] YAML parses without errors
- [ ] Cache configuration present
- [ ] JUnit artifact path correct
- [ ] JSON artifact path correct
- [ ] Documented in README under "GitLab CI" section

---

### T2: Create `.pre-commit-config.yaml` Template
**File:** `.pre-commit-config.yaml`
**Effort:** 20 min
**Dependencies:** None

**Description:**
pre-commit framework configuration for local arx-check hook.

**Requirements:**
- `repo: local` with `hooks` array
- Hook ID: `arx-check`
- Entry: `arx check --no-cache`
- `language: system` (assumes arx in PATH)
- `types: [file]`, `pass_filenames: false`

**Acceptance Criteria:**
- [ ] YAML parses without errors
- [ ] Hook runs successfully when arx is installed
- [ ] Documented in `docs/pre-commit-hook.md` with install instructions

---

### T3: Create Multi-Stage Dockerfile
**File:** `Dockerfile`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Multi-stage Docker build with distroless runtime image.

**Requirements:**
- Stage 1 (builder): `golang:1.23-alpine`
  - Copy go.mod, go.sum → `go mod download`
  - Copy source → `go build -o /arx ./cmd/arx`
- Stage 2 (runtime): `gcr.io/distroless/base-debian12`
  - Copy binary from builder
  - `ENTRYPOINT ["/arx"]`

**Acceptance Criteria:**
- [ ] Docker build succeeds: `docker build -t test/arx .`
- [ ] Image runs: `docker run --rm test/arx --version`
- [ ] Image size < 25MB
- [ ] Hadolint passes (or documented exceptions)

---

### T4: Create GitHub Workflow for Docker Publish
**File:** `.github/workflows/docker-publish.yml`
**Effort:** 30 min
**Dependencies:** T3

**Description:**
GitHub Actions workflow to build and push Docker image to GHCR on version tags.

**Requirements:**
- Trigger: `push` tags matching `v*`
- Job: `build-and-push`
  - Checkout repository
  - Set up Docker Buildx
  - Login to GHCR (`github.actor` with `secrets.GITHUB_TOKEN`)
  - Build with tags: `ghcr.io/pauvalls/arx:${VERSION}` and `:latest`
  - Push to GHCR

**Acceptance Criteria:**
- [ ] YAML parses without errors
- [ ] Workflow triggers on tag push
- [ ] Image published to GHCR
- [ ] Both version tag and `latest` tag applied

---

## Phase 2: HTML Reports

### T5: Implement HTML Reporter with Embedded Go Template + CSS
**File:** `internal/infrastructure/output/html.go`
**Effort:** 2-3 hours
**Dependencies:** None

**Description:**
HTML reporter using Go's `html/template` package with embedded CSS styles.

**Requirements:**
- Type: `HTMLReporter` with fields `tool`, `version`
- Constructor: `NewHTMLReporter()`
- Method: `Report(report domain.AuditReport) error`
- Embed CSS as Go string constant (`htmlStyles`)
- Embed HTML template using `template.Must(template.New(...).Parse(...))`
- Template sections: header, violations, coupling, debt, trends
- Use `html/template` for automatic XSS escaping

**CSS Requirements:**
- CSS variables for colors (`--color-error`, `--color-warning`, etc.)
- `.violation` class with colored left border
- `.matrix-table` for coupling matrix
- Responsive design (max-width, readable fonts)

**Acceptance Criteria:**
- [ ] File compiles without errors
- [ ] Template syntax valid (no parse errors)
- [ ] CSS embedded correctly
- [ ] All audit report sections rendered

---

### T6: Register HTML Format in Ports + Check/Audit Commands
**Files:**
- `internal/ports/reporter.go`
- `cmd/arx/check.go`
- `cmd/arx/audit.go`

**Effort:** 45 min
**Dependencies:** T5

**Description:**
Add `html` as a valid output format and wire it into check/audit commands.

**Requirements:**

**`internal/ports/reporter.go`:**
- Add constant: `OutputFormatHTML OutputFormat = "html"`

**`cmd/arx/check.go`:**
- Add `"html"` case in format switch (line ~140)
- Map to `ports.OutputFormatHTML`

**`cmd/arx/audit.go`:**
- Add `"html"` case in format validation (line ~100)
- Add HTML render function in `renderAuditReport`
- Call `output.NewHTMLReporter().Report(report)`

**Acceptance Criteria:**
- [ ] `arx check --format html` produces HTML output
- [ ] `arx audit --format html` produces HTML output
- [ ] HTML output valid HTML5 (parse with `golang.org/x/net/html`)
- [ ] No compilation errors

---

### T7: Tests for HTML Reporter
**File:** `internal/infrastructure/output/html_test.go`
**Effort:** 1-2 hours
**Dependencies:** T5, T6

**Description:**
Unit tests for HTML reporter covering validity, edge cases, and security.

**Test Cases:**
1. `TestHTMLReporter_ValidHTML5` — Parse output with `golang.org/x/net/html`, no errors
2. `TestHTMLReporter_EmptyReport` — Zero violations → valid HTML with empty sections
3. `TestHTMLReporter_FullReport` — All sections populated (violations, coupling, debt, trends)
4. `TestHTMLReporter_SpecialCharsEscaping` — Violations with `<script>`, `&`, `"` → escaped, no XSS

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Code coverage > 80% for `html.go`
- [ ] XSS test confirms `<script>` tags are escaped

---

## Phase 3: Fuzz Testing

### T8: Fuzz Test for Config YAML Parsing
**File:** `internal/infrastructure/config/config_fuzz.go`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Fuzz test for YAML config parser to catch panics on malformed input.

**Requirements:**
- Function: `FuzzConfigParse(f *testing.F)`
- Fuzz target: `configReader.ReadFrom(bytes.NewReader(data))`
- Errors expected (random input), panics not acceptable

**Acceptance Criteria:**
- [ ] Test passes with `go test -fuzz=FuzzConfigParse -fuzztime=10s`
- [ ] No panics on random byte input
- [ ] Documented in spec that fuzz runs 30s in CI

---

### T9: Fuzz Test for Java Import Regex
**File:** `internal/infrastructure/detector/java/parser_fuzz.go`
**Effort:** 20 min
**Dependencies:** None

**Description:**
Fuzz test for Java import parser to catch regex panics.

**Requirements:**
- Function: `FuzzJavaParse(f *testing.F)`
- Fuzz target: `extractImportsFromLine(string(data))`
- Errors/empty results expected, panics not acceptable

**Acceptance Criteria:**
- [ ] Test passes with `go test -fuzz=FuzzJavaParse -fuzztime=10s`
- [ ] No panics on random byte input

---

### T10: Fuzz Test for C# Import Regex
**File:** `internal/infrastructure/detector/csharp/parser_fuzz.go`
**Effort:** 20 min
**Dependencies:** None

**Description:**
Fuzz test for C# import parser to catch regex panics.

**Requirements:**
- Function: `FuzzCSharpParse(f *testing.F)`
- Fuzz target: `extractImportsFromLine(string(data))`

**Acceptance Criteria:**
- [ ] Test passes with `go test -fuzz=FuzzCSharpParse -fuzztime=10s`
- [ ] No panics on random byte input

---

### T11: Fuzz Test for Rust Import Regex
**File:** `internal/infrastructure/detector/rust/parser_fuzz.go`
**Effort:** 20 min
**Dependencies:** None

**Description:**
Fuzz test for Rust import parser to catch regex panics.

**Requirements:**
- Function: `FuzzRustParse(f *testing.F)`
- Fuzz target: `extractImportsFromLine(string(data))`

**Acceptance Criteria:**
- [ ] Test passes with `go test -fuzz=FuzzRustParse -fuzztime=10s`
- [ ] No panics on random byte input

---

## Phase 4: Benchmarks

### T12: Benchmark for Coupling Matrix Calculation
**File:** `internal/domain/coupling_bench_test.go`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Benchmark for coupling matrix `Add` operation (O(n²) hot path).

**Requirements:**
- Function: `BenchmarkCouplingMatrix(b *testing.B)`
- Setup: 5-10 layers, pre-populate matrix
- Benchmark: `matrix.Add(from, to)` in loop
- Report: ops/ns, allocs/op

**Acceptance Criteria:**
- [ ] Benchmark runs: `go test -bench=BenchmarkCouplingMatrix -benchmem`
- [ ] Baseline recorded in comments
- [ ] No memory leaks detected

---

### T13: Benchmark for Rule Evaluation
**File:** `internal/domain/audit_bench_test.go`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Benchmark for rule evaluation loop (called per-violation).

**Requirements:**
- Function: `BenchmarkRuleEvaluation(b *testing.B)`
- Setup: 500 violations, 5-10 rules
- Benchmark: rule matching loop
- Report: ops/ns, allocs/op

**Acceptance Criteria:**
- [ ] Benchmark runs: `go test -bench=BenchmarkRuleEvaluation -benchmem`
- [ ] Baseline recorded in comments

---

### T14: Benchmark for Java Import Extraction
**File:** `internal/infrastructure/detector/java/parser_bench_test.go`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Benchmark for full Java file import extraction.

**Requirements:**
- Function: `BenchmarkJavaExtraction(b *testing.B)`
- Setup: Realistic Java file content (10-20 imports)
- Benchmark: `java.ExtractImports(content, layers)`
- Report: files/s (derived from ns/op)

**Acceptance Criteria:**
- [ ] Benchmark runs: `go test -bench=BenchmarkJavaExtraction -benchmem`
- [ ] Baseline recorded in comments

---

### T15: Benchmark for C# Import Extraction
**File:** `internal/infrastructure/detector/csharp/parser_bench_test.go`
**Effort:** 30 min
**Dependencies:** None

**Description:**
Benchmark for full C# file import extraction.

**Requirements:**
- Function: `BenchmarkCSharpExtraction(b *testing.B)`
- Setup: Realistic C# file content (10-20 using directives)
- Benchmark: `csharp.ExtractImports(content, layers)`

**Acceptance Criteria:**
- [ ] Benchmark runs: `go test -bench=BenchmarkCSharpExtraction -benchmem`
- [ ] Baseline recorded in comments

---

## Phase 5: Polish

### T16: Update README Docs with New Features
**File:** `README.md`
**Effort:** 45 min
**Dependencies:** T1-T7

**Description:**
Document new CI/CD templates and HTML output format.

**Sections to Add/Update:**
1. **CI/CD Integration** table row: GitLab CI, pre-commit, Docker
2. **Output Formats** table row: `html` format
3. **Quickstart** example: `arx check --format html`
4. Link to new docs pages (if created)

**Acceptance Criteria:**
- [ ] GitLab CI mentioned in README
- [ ] pre-commit hook mentioned
- [ ] Docker image publishing documented
- [ ] HTML format listed in output formats
- [ ] All links valid

---

### T17: Verify All CI/CD Templates
**File:** N/A (verification task)
**Effort:** 1 hour
**Dependencies:** T1-T4

**Description:**
End-to-end verification of all CI/CD templates in real environments.

**Verification Checklist:**
- [ ] `.gitlab-ci.yml`: Run with `gitlab-ci-local` or push to GitLab
  - Check job produces JUnit report
  - Audit job produces JSON artifact
- [ ] `.pre-commit-config.yaml`: Run `pre-commit run arx-check`
  - Hook executes successfully
- [ ] `Dockerfile`: Build and run
  - Image size < 25MB
  - `arx --version` works
- [ ] `.github/workflows/docker-publish.yml`: Trigger on test tag
  - Image appears in GHCR

**Acceptance Criteria:**
- [ ] All templates validated
- [ ] Any issues fixed
- [ ] Verification results documented in PR description

---

## Review Workload Forecast

| Task | Review Complexity | Est. Review Time | Notes |
|------|------------------|------------------|-------|
| T1-T4 (CI/CD templates) | Low | 15 min each | YAML validation, simple logic |
| T5 (HTML reporter) | Medium | 30 min | Template syntax, CSS review |
| T6 (HTML registration) | Low | 15 min | Simple switch cases |
| T7 (HTML tests) | Medium | 20 min | Verify XSS test coverage |
| T8-T11 (Fuzz tests) | Low | 10 min each | Verify no panics |
| T12-T15 (Benchmarks) | Low | 10 min each | Verify baseline recorded |
| T16 (README) | Low | 15 min | Docs accuracy check |
| T17 (Verification) | Medium | 30 min | Run all templates |

**Total Review Time:** ~3.5 hours

---

## Critical Path Analysis

### Critical Path (Sequential Dependencies)
```
T5 (HTML reporter) ──→ T6 (HTML registration) ──→ T7 (HTML tests) ──→ T16 (README)
     2-3h                    45min                      1-2h            45min

T3 (Dockerfile) ──→ T4 (Docker workflow)
     30min              30min

T17 (Verification) depends on T1-T4 completion
```

### Parallel Workstreams

**Stream A: CI/CD Templates (can be done in parallel)**
- T1, T2, T3 → T4 → T17

**Stream B: HTML Reports (sequential)**
- T5 → T6 → T7 → T16

**Stream C: Fuzz Tests (fully parallel)**
- T8, T9, T10, T11 (any order)

**Stream D: Benchmarks (fully parallel)**
- T12, T13, T14, T15 (any order)

### Minimum Viable Implementation (MVI)
If time-constrained, implement in this order:
1. **T5, T6, T7** — HTML reports (core feature)
2. **T1, T2** — GitLab CI + pre-commit (most used CI/CD)
3. **T8-T11** — Fuzz tests (security/stability)

### Deferred (if needed)
- T3, T4 — Docker publishing (nice-to-have for v0.11.0)
- T12-T15 — Benchmarks (performance baselines can wait)

---

## Implementation Order Recommendation

**Day 1:** T1, T2, T3, T4 (CI/CD templates — quick wins, no Go code)
**Day 2:** T5, T6, T7 (HTML reporter — core feature)
**Day 3:** T8, T9, T10, T11 (Fuzz tests) + T12, T13, T14, T15 (Benchmarks)
**Day 4:** T16, T17 (Polish + verification)

**Total:** 4 days (conservative estimate)
**Fast track:** 2.5 days (skip Docker, benchmarks)

---

## Notes

- All features are **additive** — no breaking changes
- HTML output selected via `--format html` flag (opt-in)
- Fuzz tests run 30s in CI (documented in spec)
- Benchmarks run locally only (`make bench`)
- CI/CD templates are copy-paste (users own their configs)

(End of tasks)
