# Implementation Tasks: Arx v0.10.0 — Maturity & Developer Experience

**Change ID:** `arx-v0.10.0-maturity-dx`
**Version:** 0.10.0
**Theme:** CLI maturity, developer experience, build infrastructure

---

## Overview

Nineteen (19) implementation tasks organized in 6 phases. Total estimated effort: **~45-55 hours** of focused development work.

### Parallelization Summary

| Phase | Tasks | Sequential | Parallelizable | Est. Duration |
|-------|-------|------------|----------------|---------------|
| 1. Build Infrastructure | T1-T3 | T1, T2 | T3 (independent) | 2-3 hours |
| 2. Diagram CLI + Mermaid | T4-T6 | T4 → T5 → T6 | — | 6-8 hours |
| 3. DX Commands | T7-T10 | T7, T8, T9 | T10 (after T7-T9) | 8-10 hours |
| 4. C# Detector | T11-T14 | T11 → T12 → T13 → T14 | — | 10-12 hours |
| 5. CI Output Formats | T15-T17 | T15, T16 | T17 (after T15-T16) | 6-8 hours |
| 6. Polish | T18-T19 | T18 → T19 | — | 3-4 hours |

**Critical Path:** T11 → T12 → T13 → T14 (C# Detector) — longest sequential chain at 10-12 hours

---

## Phase 1: Build Infrastructure (Foundation)

### T1: Create Makefile with build/test/lint/clean targets

**Type:** `infrastructure`
**Priority:** P0 (blocking for consistent builds)
**Estimate:** 30-45 minutes
**Review Load:** Light (~10 min)

**Acceptance Criteria:**
- [x] `make build` compiles binary to `./arx`
- [x] `make test` runs `go test ./...` with verbose output
- [x] `make lint` runs `golangci-lint run` (graceful skip if not installed)
- [x] `make clean` removes compiled binary
- [x] `make` (default target) runs build
- [x] Makefile uses `.PHONY` declarations
- [x] Makefile includes help target with descriptions

**Implementation Notes:**
- Check for `golangci-lint` availability; print warning if missing instead of failing
- Use `go build -o arx ./cmd/arx` for build target
- Include `GO111MODULE=on` explicitly for compatibility

**Files:**
- `Makefile` (create)

**Dependencies:** None

---

### T2: Create CHANGELOG.md with history from v0.1.0 to v0.10.0

**Type:** `documentation`
**Priority:** P0 (release artifact)
**Estimate:** 45-60 minutes
**Review Load:** Light (~10 min)

**Acceptance Criteria:**
- [x] Entries for v0.1.0 through v0.9.0 (historical reconstruction)
- [x] Entry for v0.10.0 (current work) marked as `[Unreleased]`
- [x] Format: `## [version] - YYYY-MM-DD` followed by `### Features`, `### Fixes`, `### Changes`
- [x] Each entry includes notable PRs or commits if available
- [x] v0.10.0 lists all 8 new features from this change

**Implementation Notes:**
- Reconstruct historical releases from git tags and commit history
- For v0.10.0, list: diagram CLI, Mermaid output, shell completion, config validate, doctor command, C# detector, JUnit output, GitHub Annotations
- Follow Keep a Changelog format (https://keepachangelog.com/)

**Files:**
- `CHANGELOG.md` (create)

**Dependencies:** None

---

### T3: Fix deprecated APIs

**Type:** `bugfix`
**Priority:** P0 (technical debt)
**Estimate:** 30-45 minutes
**Review Load:** Light (~10 min)

**Acceptance Criteria:**
- [x] `strings.Title()` replaced with `cases.Title(language.Und)` from `golang.org/x/text/cases`
- [x] `filepath.HasPrefix()` replaced with `strings.HasPrefix()` (function doesn't exist)
- [x] All deprecated API warnings resolved in `go vet`
- [x] All tests pass after changes
- [x] `golangci-lint` reports no deprecation warnings

**Implementation Notes:**
- Search codebase for `strings.Title` — likely in terminal output or ASCII diagram rendering
- `filepath.HasPrefix` is a common mistake — the function never existed; should be `strings.HasPrefix`
- Import `golang.org/x/text/cases` and `golang.org/x/text/language` for cases.Title

**Files to Modify:**
- `cmd/arx/*.go` (search and replace)
- `internal/infrastructure/output/*.go` (likely in ascii.go or terminal.go)

**Dependencies:** None (can run in parallel with T1, T2)

---

## Phase 2: Diagram CLI + Mermaid (High Visibility)

### T4: Wire diagram cobra command

**Type:** `feature`
**Priority:** P0 (core feature)
**Estimate:** 1-1.5 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [ ] `arx diagram --help` shows usage with flags
- [ ] `--format` flag accepts `ascii`, `dot`, `mermaid` (default: `ascii`)
- [ ] `--output` flag writes to file instead of stdout
- [ ] Command invokes `DiagramService.Generate()` with project root
- [ ] Output routed to correct formatter based on `--format`
- [ ] Error handling: invalid format returns non-zero exit code
- [ ] Integration with root command (registered in `root.go`)

**Implementation Notes:**
- Follow pattern from existing commands (`check.go`, `audit.go`)
- Reuse `DiagramService` from `internal/application/diagram.go`
- Output format switching happens after service returns result
- Use `cmd/arx/diagram.go` as command file

**Files:**
- `cmd/arx/diagram.go` (create)
- `cmd/arx/root.go` (modify — register command)

**Dependencies:** T3 (deprecation fixes — avoid introducing deprecated APIs in new code)

---

### T5: Implement Mermaid output format

**Type:** `feature`
**Priority:** P0 (core feature)
**Estimate:** 2-3 hours
**Review Load:** Medium (~25 min)

**Acceptance Criteria:**
- [ ] `GenerateMermaid(result *application.DiagramResult) string` function
- [ ] Output valid Mermaid flowchart syntax: `flowchart TD`
- [ ] Each layer rendered as subgraph with distinct color
- [ ] Dependencies rendered as edges: `A[Layer1] --> B[Layer2]`
- [ ] Violations rendered as red edges: `A -.->|violation| B`
- [ ] Layer nodes styled with fill colors matching layer type
- [ ] Output escapes special characters in layer names
- [ ] Unit tests verify syntax validity

**Implementation Notes:**
- Mermaid flowchart syntax: https://mermaid.js.org/syntax/flowchart.html
- Use subgraphs for layers: `subgraph Layer1["Layer1"] ... end`
- Violations: use dashed red line with label
- Layer colors: use consistent palette (blue=domain, green=application, etc.)
- Escape special chars: `[`, `]`, `(`, `)`, `-` in node labels

**Files:**
- `internal/infrastructure/output/mermaid.go` (create)
- `internal/infrastructure/output/mermaid_test.go` (create)

**Dependencies:** T4 (need to understand result structure)

---

### T6: Tests for diagram command + mermaid output

**Type:** `test`
**Priority:** P1 (quality)
**Estimate:** 2-3 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [ ] Table-driven tests for `--format` flag (ascii, dot, mermaid)
- [ ] Test `--output` writes to file (verify file exists, content matches stdout)
- [ ] Test invalid format returns error
- [ ] Test Mermaid output with 0 violations (clean graph)
- [ ] Test Mermaid output with violations (red edges present)
- [ ] Test Mermaid output with multiple layers (subgraphs correct)
- [ ] Integration test: real project → diagram command → valid output

**Implementation Notes:**
- Use `ArxFakeProject` test fixture pattern from existing tests
- Mock `DiagramService` for command-level tests
- For Mermaid tests, verify syntax elements present (not full parsing)
- Use `os.CreateTemp` for `--output` flag tests

**Files:**
- `cmd/arx/diagram_test.go` (create)
- `internal/infrastructure/output/mermaid_test.go` (create, may be done with T5)

**Dependencies:** T4, T5

---

## Phase 3: DX Commands (Quick Wins)

### T7: Shell completion command

**Type:** `feature`
**Priority:** P1 (developer convenience)
**Estimate:** 1-1.5 hours
**Review Load:** Light (~15 min)

**Acceptance Criteria:**
- [ ] `arx completion --help` shows available shells
- [ ] `arx completion bash` outputs valid bash completion script
- [ ] `arx completion zsh` outputs valid zsh completion script
- [ ] `arx completion fish` outputs valid fish completion script
- [ ] `arx completion powershell` outputs valid PowerShell completion script
- [ ] Each subcommand writes to stdout (can be redirected)
- [ ] Command is hidden from main `arx --help` (discoverable via `completion --help`)

**Implementation Notes:**
- Use Cobra's built-in generators: `GenBashCompletion()`, `GenZshCompletion()`, etc.
- Create subcommand for each shell
- Mark parent `completion` command as `Hidden: true`
- Follow pattern from Cobra documentation

**Files:**
- `cmd/arx/completion.go` (create)
- `cmd/arx/root.go` (modify — register command)

**Dependencies:** None (can parallelize with Phase 2)

---

### T8: arx config validate command

**Type:** `feature`
**Priority:** P1 (developer convenience)
**Estimate:** 1-1.5 hours
**Review Load:** Light (~15 min)

**Acceptance Criteria:**
- [ ] `arx config validate --help` shows usage
- [ ] `--path` flag accepts config file path (default: `arx.yaml`)
- [ ] Valid config: prints `✓ Config valid` and exits 0
- [ ] Invalid config: prints `✗ Error: <details>` and exits 1
- [ ] Missing file: prints clear error `Config file not found: <path>`
- [ ] Validation errors include field path (e.g., `rules[0].from: invalid layer`)
- [ ] Reuses existing `configReader.Read()` and `configReader.Validate()`

**Implementation Notes:**
- Create `config` subcommand group (like `hook`, `baseline`)
- Use existing config reader from `internal/infrastructure/config/`
- Terminal output: use checkmark/X symbols for clarity
- Exit codes: 0 = valid, 1 = invalid/error

**Files:**
- `cmd/arx/config.go` (create)
- `cmd/arx/root.go` (modify — register command group)

**Dependencies:** None

---

### T9: arx doctor command + DoctorService

**Type:** `feature`
**Priority:** P1 (developer convenience)
**Estimate:** 2-3 hours
**Review Load:** Medium (~25 min)

**Acceptance Criteria:**
- [ ] `arx doctor --help` shows usage
- [ ] DoctorService with `Check(projectRoot string) DoctorResult` method
- [ ] Check 1: Project root exists (✓/✗)
- [ ] Check 2: Config file exists (✓/✗)
- [ ] Check 3: Detectors find files (✓/✗ + count)
- [ ] Check 4: Version info (prints arx version)
- [ ] Check 5: Git status (if repo — shows branch, dirty/clean)
- [ ] Output: one line per check with ✓/✗ and message
- [ ] Exit 0 unless critical error (cannot read directory)

**Implementation Notes:**
- `DoctorService` in `internal/application/doctor.go`
- `DoctorResult` struct with `CheckResult` per concern
- `CheckResult` has `OK bool`, `Message string`
- Git status: use `git rev-parse --abbrev-ref HEAD` and `git status --porcelain`
- Detectors check: run all detectors, count how many return `true`

**Files:**
- `cmd/arx/doctor.go` (create)
- `internal/application/doctor.go` (create)
- `cmd/arx/root.go` (modify — register command)

**Dependencies:** None

---

### T10: Tests for completion/validate/doctor

**Type:** `test`
**Priority:** P1 (quality)
**Estimate:** 2-3 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [ ] Completion: smoke test for each shell (output non-empty, starts with `#`)
- [ ] Config validate: valid config → success message
- [ ] Config validate: invalid config → error with details
- [ ] Config validate: missing file → clear error
- [ ] DoctorService: mock project root check (exists/missing)
- [ ] DoctorService: mock config check (exists/missing)
- [ ] DoctorService: mock detectors (0 found, 1+ found)
- [ ] Doctor command: integration test on real project

**Implementation Notes:**
- Completion tests: verify output starts with shell-specific comment
- Config validate: use temp files with valid/invalid YAML
- DoctorService: use dependency injection for testability
- Mock git commands or skip git check in tests

**Files:**
- `cmd/arx/completion_test.go` (create)
- `cmd/arx/config_test.go` (create)
- `cmd/arx/doctor_test.go` (create)
- `internal/application/doctor_test.go` (create)

**Dependencies:** T7, T8, T9

---

## Phase 4: C# Detector (Medium Effort)

### T11: C# detector skeleton

**Type:** `feature`
**Priority:** P0 (new language support)
**Estimate:** 1-1.5 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [ ] `CSharpDetector` struct with `modulePrefix`, `sourceDirs` fields
- [ ] `New()` constructor returns `*CSharpDetector`
- [ ] `Name()` returns `"csharp"`
- [ ] `Detect(ctx, projectRoot)` checks for `.csproj` or `.sln` files
- [ ] `Detect()` returns `true` if either file exists at project root
- [ ] `Detect()` returns `false` if neither exists
- [ ] Follows exact pattern from `RustDetector`

**Implementation Notes:**
- Mirror `RustDetector` structure exactly
- Search for `.csproj` and `.sln` at project root (not recursive)
- Use `os.Stat()` for file existence checks
- Source dirs: default to `["src/"]` like other detectors

**Files:**
- `internal/infrastructure/detector/csharp/detector.go` (create)

**Dependencies:** None (can parallelize with earlier phases)

---

### T12: C# import extraction

**Type:** `feature`
**Priority:** P0 (new language support)
**Estimate:** 3-4 hours
**Review Load:** Medium (~30 min)

**Acceptance Criteria:**
- [ ] `ExtractImports(ctx, projectRoot, layers)` walks source directories
- [ ] `parseFile(filePath)` extracts `using` directives
- [ ] Regex for standard using: `using\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;`
- [ ] Regex for static using: `using\s+static\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;`
- [ ] Regex for alias using: `using\s+([a-zA-Z_]+)\s*=\s*([a-zA-Z_][a-zA-Z0-9_.]+)\s*;`
- [ ] `resolveImport(importPath)` matches to layers
- [ ] `isExternalDependency(importPath)` skips `System.*`, `Microsoft.*`
- [ ] Skips test files (`*Test.cs`, `*Tests.cs`)
- [ ] Skips build directories (`obj/`, `bin/`, `.vs/`)

**Implementation Notes:**
- Mirror `RustDetector` parser structure
- Use `regexp.MustCompile` for regex patterns (compile once)
- Line numbers are 1-indexed
- External skip list: `System.`, `Microsoft.`, `Mono.`, `UnityEditor.`, `UnityEngine.`

**Files:**
- `internal/infrastructure/detector/csharp/parser.go` (create)
- `internal/infrastructure/detector/csharp/detector.go` (modify — add ExtractImports)

**Dependencies:** T11

---

### T13: Register in detector registry

**Type:** `integration`
**Priority:** P0 (new language support)
**Estimate:** 15-30 minutes
**Review Load:** Light (~10 min)

**Acceptance Criteria:**
- [ ] Import `csharp` package in `registry.go`
- [ ] `csharp.New()` added to `GetDetectors()` return slice
- [ ] All existing tests pass with new detector registered
- [ ] `arx doctor` on C# project shows C# detector active

**Implementation Notes:**
- Add import: `csharpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/csharp"`
- Add to slice: `csharpdetector.New(),`
- No other changes needed — registry pattern is automatic

**Files:**
- `internal/infrastructure/detector/registry.go` (modify)

**Dependencies:** T11, T12

---

### T14: C# test fixtures + integration tests

**Type:** `test`
**Priority:** P1 (quality)
**Estimate:** 4-5 hours
**Review Load:** Medium (~30 min)

**Acceptance Criteria:**
- [ ] Unit test: `Detect()` with `.csproj` → `true`
- [ ] Unit test: `Detect()` with `.sln` → `true`
- [ ] Unit test: `Detect()` with neither → `false`
- [ ] Unit test: standard using extraction (table-driven)
- [ ] Unit test: static using extraction
- [ ] Unit test: alias using extraction
- [ ] Unit test: external dependency skip (System.*, Microsoft.*)
- [ ] Unit test: layer resolution (MyApp.Domain → domain layer)
- [ ] Integration test: ArxFakeProject with `.csproj` + `src/*.cs`
- [ ] Integration test: full `arx check` on C# project

**Implementation Notes:**
- Create test fixtures in `test/fixtures/csharp/`
- Use `ArxFakeProject` pattern for integration tests
- Table-driven parser tests: input line → expected import path
- Integration test: seed project with known violations, verify detection

**Files:**
- `internal/infrastructure/detector/csharp/detector_test.go` (create)
- `internal/infrastructure/detector/csharp/parser_test.go` (create)
- `test/fixtures/csharp/` (create — test project)

**Dependencies:** T11, T12, T13

---

## Phase 5: CI Output Formats

### T15: JUnit XML output format

**Type:** `feature`
**Priority:** P1 (CI integration)
**Estimate:** 2-3 hours
**Review Load:** Medium (~25 min)

**Acceptance Criteria:**
- [x] `JUnitReporter` struct with `Report(violations, format) error` method
- [x] XML structure with `testsuites` root element
- [x] `testsuite` child with `name="arx"`, `tests=N`, `failures=M`
- [x] `testcase` per violation with `name`, `file`, `line` attributes
- [x] `failure` child with `message` and `type="architecture"`
- [x] Uses `encoding/xml` marshaling (not string building)
- [x] Proper XML escaping for special characters
- [x] Output to stdout (can be redirected in CI)

**Implementation Notes:**
- Define structs with `xml` tags for marshaling
- XML namespace: none (simple JUnit format)
- Test count = number of violations (each violation is a "failed test")
- Failure message = violation description
- Follow JUnit schema for CI tool compatibility

**Files:**
- `internal/infrastructure/output/junit.go` (create)
- `internal/infrastructure/output/junit_test.go` (create)

**Dependencies:** None

---

### T16: GitHub Annotations output format

**Type:** `feature`
**Priority:** P1 (CI integration)
**Estimate:** 1.5-2 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [x] `GitHubAnnotationsReporter` struct with `Report(violations, format) error`
- [x] Output format: `::error file={file},line={line},title={title}::{message}`
- [x] Severity mapping: `error` → `::error`, `warning` → `::warning`, `info` → `::notice`
- [x] One line per violation
- [x] Proper URL encoding for file paths (if contains special chars)
- [x] Title truncated to 50 chars if longer
- [x] Message escaped for workflow command syntax

**Implementation Notes:**
- GitHub Actions workflow commands: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions
- Escape `%` as `%25`, `\r` as `%0D`, `\n` as `%0A`
- File paths: use relative paths from project root
- Title: use violation type (e.g., "Architecture Violation")

**Files:**
- `internal/infrastructure/output/gh_annotations.go` (create)
- `internal/infrastructure/output/gh_annotations_test.go` (create)

**Dependencies:** None (can parallelize with T15)

---

### T17: Tests for new output formats

**Type:** `test`
**Priority:** P1 (quality)
**Estimate:** 2-3 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [x] JUnit: XML structure valid (parse with xml.Decoder)
- [x] JUnit: test count matches violation count
- [x] JUnit: failure message contains violation details
- [x] JUnit: special characters properly escaped
- [x] GitHub Annotations: workflow command syntax correct
- [x] GitHub Annotations: severity mapping correct (error/warning/info)
- [x] GitHub Annotations: file paths relative to project root
- [x] Integration test: `arx check --format junit` outputs valid XML
- [x] Integration test: `arx check --format gh-annotations` outputs workflow commands

**Implementation Notes:**
- Use `encoding/xml` decoder to validate JUnit output
- Regex match for GitHub Annotations format
- Integration tests: use `ArxFakeProject` with known violations

**Files:**
- `internal/infrastructure/output/junit_test.go` (create, may be done with T15)
- `internal/infrastructure/output/gh_annotations_test.go` (create, may be done with T16)
- `cmd/arx/check_test.go` (modify — add format flag tests)

**Dependencies:** T15, T16

---

## Phase 6: Polish

### T18: Update README docs with new commands

**Type:** `documentation`
**Priority:** P2 (user-facing)
**Estimate:** 1.5-2 hours
**Review Load:** Medium (~20 min)

**Acceptance Criteria:**
- [ ] `diagram` command documented with examples
- [ ] `completion` command documented (all shells listed)
- [ ] `config validate` command documented
- [ ] `doctor` command documented with sample output
- [ ] C# detector mentioned in supported languages
- [ ] JUnit and GitHub Annotations formats documented
- [ ] Makefile targets documented in Contributing section
- [ ] CHANGELOG linked from README

**Implementation Notes:**
- Add command examples with expected output
- Update "Supported Languages" section to include C#
- Update "Output Formats" section with JUnit and GitHub Annotations
- Include Mermaid example with rendered output (if GitHub supports it)

**Files:**
- `README.md` (modify)
- `docs/output-formats.md` (modify — add JUnit, GitHub Annotations)

**Dependencies:** T4-T17 (all features must be implemented first)

---

### T19: Full end-to-end tests

**Type:** `test`
**Priority:** P1 (quality gate)
**Estimate:** 2-3 hours
**Review Load:** Medium (~30 min)

**Acceptance Criteria:**
- [ ] E2E: `arx diagram --format mermaid` on real project → valid output
- [ ] E2E: `arx completion bash` → install → tab completion works
- [ ] E2E: `arx config validate` on valid/invalid configs
- [ ] E2E: `arx doctor` on real project → all checks run
- [ ] E2E: `arx check --format junit` → valid XML
- [ ] E2E: `arx check --format gh-annotations` → workflow commands
- [ ] E2E: C# detector on real C# project → dependencies extracted
- [ ] All E2E tests pass in CI environment
- [ ] Test suite completes in < 5 minutes

**Implementation Notes:**
- Use `test/e2e/` directory for E2E tests
- Spin up temporary projects for each test
- Clean up temp files after tests
- Use `exec.Command` to run CLI and capture output
- Skip tests requiring git if not in git repo

**Files:**
- `test/e2e/diagram_test.go` (create)
- `test/e2e/completion_test.go` (create)
- `test/e2e/config_test.go` (create)
- `test/e2e/doctor_test.go` (create)
- `test/e2e/output_formats_test.go` (create)
- `test/e2e/csharp_detector_test.go` (create)

**Dependencies:** T4-T17 (all features must be implemented first)

---

## Review Workload Forecast

| Phase | Total Review Time | Peak Concurrent Reviews | Bottleneck |
|-------|------------------|------------------------|------------|
| 1. Build Infrastructure | 30 min | 1 | None |
| 2. Diagram CLI + Mermaid | 45 min | 1 | T5 (Mermaid complexity) |
| 3. DX Commands | 1h 10 min | 2 | T10 (test coverage) |
| 4. C# Detector | 1h 20 min | 1 | T14 (test fixtures) |
| 5. CI Output Formats | 45 min | 2 | None |
| 6. Polish | 50 min | 1 | T19 (E2E flakiness) |
| **Total** | **~5 hours** | **2-3** | — |

**Review Distribution:**
- Light reviews (<15 min): T1, T2, T3, T7, T8, T13
- Medium reviews (20-30 min): T4, T5, T6, T9, T10, T11, T12, T14, T15, T16, T17, T18, T19

**Recommendation:** Batch Phase 3 reviews (T7-T10) together — same reviewer can handle all DX commands efficiently.

---

## Critical Path Analysis

**Critical Path:** T11 → T12 → T13 → T14 (C# Detector)

```
T11 (1.5h) → T12 (4h) → T13 (0.5h) → T14 (5h) = 11 hours sequential
```

**Why this is critical:**
- C# detector is the most complex single feature
- Parser regex and import resolution require careful testing
- Test fixtures need realistic C# project structure
- No parallelization opportunity within this chain

**Mitigation:**
- Start T11 early in the implementation cycle
- Can parallelize T1-T10 while T11-T14 are in progress
- T15-T17 (CI formats) can also run in parallel

**Secondary Critical Path:** T4 → T5 → T6 (Diagram + Mermaid)

```
T4 (1.5h) → T5 (3h) → T6 (3h) = 7.5 hours sequential
```

**Risk:** Mermaid output complexity may extend T5 beyond estimate

---

## Parallelization Opportunities

### Wave 1: Foundation (Hours 0-3)
**Parallel:** T1, T2, T3
- All independent
- Can be done by different developers
- T3 (deprecation fixes) requires no coordination

### Wave 2: Core Features (Hours 3-15)
**Parallel Tracks:**
- **Track A:** T4 → T5 → T6 (Diagram + Mermaid) — 7.5 hours
- **Track B:** T7, T8, T9 → T10 (DX Commands) — 6-8 hours
- **Track C:** T11 → T12 → T13 → T14 (C# Detector) — 11 hours

**Optimal Assignment:**
- Developer 1: Track A (Diagram expertise)
- Developer 2: Track B (DX commands — quick wins)
- Developer 3: Track C (C# detector — most complex)

### Wave 3: CI Formats (Hours 8-18)
**Parallel:** T15, T16 → T17
- Can start after Wave 2 begins
- Independent of Diagram and DX commands
- Requires only output interface knowledge

### Wave 4: Polish (Hours 15-20)
**Sequential:** T18 → T19
- Must wait for all features complete
- T19 (E2E) validates everything

---

## Task Dependency Graph

```
T1 ──┐
T2 ──┼── T4 ── T5 ── T6 ──┐
T3 ──┘                    │
                          ├── T18 ── T19
T7 ──┐                    │
T8 ──┼── T10 ─────────────┤
T9 ──┘                    │
                          │
T11 ── T12 ── T13 ── T14 ─┤
                          │
T15 ──┐                   │
T16 ──┼── T17 ────────────┘
```

---

## Definition of Done (Per Task)

Each task is complete when:
- [ ] Implementation matches acceptance criteria
- [ ] Unit tests pass (`go test ./...`)
- [ ] Integration tests pass (if applicable)
- [ ] Code reviewed and approved
- [ ] No new `golangci-lint` warnings introduced
- [ ] Documentation updated (if user-facing)
- [ ] CHANGELOG entry added (if feature/fix)

---

## Next Step

Ready for **implementation phase** (sdd-apply). Start with Phase 1 (T1-T3) to establish build infrastructure, then proceed to Phase 2-3 in parallel.

(End of file)
