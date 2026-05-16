# Roadmap

## ✅ v0.18.0 (Current — Expression-Based Rules, Concurrent Detectors, Severity Mapping)

- [x] Expression engine — Recursive-descent parser + evaluator (zero deps)
- [x] Built-in functions: `count()`, `deps()`, `layers()`, `has_circular()`
- [x] Operators: `>`, `<`, `>=`, `<=`, `==`, `!=`, `&&`, `||`, `!`
- [x] Rule `check` field for inline expressions (backward compatible)
- [x] Concurrent detectors — `errgroup`-based parallel execution
- [x] Severity mapping — `severity_mapping: {critical: error, minor: warning}`

## ✅ v0.17.0 (Custom Rule Templates)

- [x] Template engine — `TemplateFunc` type + `TemplateRegistry` in `internal/domain/template.go`
- [x] `max-deps` template — Max dependencies between two layers
- [x] `no-leak` template — Layer must not import from forbidden layers
- [x] `layer-balance` template — Each layer must have N-M dependencies
- [x] Config integration — `template` + `params` fields on rules with validation
- [x] Evaluation — template rules run alongside standard rules (AND logic)
- [x] Template violations use `T-` prefix (distinct from standard `D-`)

## ✅ v0.16.0 (Audit HTML, More Presets, Quality)

- [x] `arx audit --format html` — HTML output for audit command
- [x] `arx init --preset layered` — Layered Architecture preset
- [x] `arx init --preset onion` — Onion Architecture preset
- [x] Shell completion install instructions (bash/zsh/fish/powershell)
- [x] Quality pass — replaced deprecated bubble sort with `sort.Strings()`

## ✅ v0.15.0 (Man Pages, .arxignore, Fuzz Coverage)

- [x] `arx man` — Man page generation for Linux distros
- [x] `.editorconfig` — Editor settings for contributors
- [x] `.github/dependabot.yml` — Automated dependency updates
- [x] Fuzz tests for Kotlin, PHP, Swift parsers (30k+ execs, 0 crashes)
- [x] `.arxignore` — Project-wide ignore file (like .gitignore)
- [x] ArxIgnore wired into all 10 detectors

## ✅ v0.14.0 (PHP + Swift Detectors)

- [x] PHP detector — `composer.json` detection, `use`/`use as`/`use function` parsing
- [x] Swift detector — `Package.swift` detection, `import`/`@_exported import` parsing
- [x] PHP/Swift test fixtures + integration tests
- [x] E2E tests for all 10 languages

## ✅ v0.13.0 (Ruby Detector + Fuzz Tests)

- [x] Ruby detector — `Gemfile` detection, `require`/`require_relative` parsing
- [x] Parser fuzz tests — Java, C#, Rust, Ruby (170k+ execs, 0 crashes)
- [x] E2E tests for all 8 languages (Go, TS, Python, Java, Kotlin, Rust, C#, Ruby)
- [x] Python fixture + E2E test
- [x] Java/Ruby fixtures with arx.yaml
- [x] All output format E2E verification

## ✅ v0.12.0 (Fail Threshold, Excludes, Releases)

- [x] `max_violations` config field — CI failure threshold
- [x] `rules[].exclude` — Per-rule path exclusion via glob
- [x] `.goreleaser.yaml` — Multi-platform releases (linux/darwin amd64+arm64)
- [x] Homebrew tap — `brew install arx` via goreleaser
- [x] `.deb`/`.rpm` packages
- [x] Benchmarks — coupling matrix + rule evaluation
- [x] Fuzz test — config YAML parser
- [x] GitHub Action fixes — SARIF URIs, exit codes, upload-sarif@v4

## ✅ v0.11.0 (CI/CD + HTML Reports)

- [x] `.gitlab-ci.yml` — GitLab CI template with JUnit/JSON artifacts
- [x] `.pre-commit-config.yaml` — Standard pre-commit framework hook
- [x] `Dockerfile` — Multi-stage (golang → distroless) for GHCR
- [x] `.github/workflows/docker-publish.yml` — Automatic Docker publishing
- [x] `arx check --format html` — Self-contained HTML5 reports
- [x] Embedded CSS + responsive layout + print-friendly

## ✅ v0.10.0 (Project Maturity + DX)

- [x] `arx diagram` — CLI command with ASCII/DOT/Mermaid output
- [x] Shell completion — bash/zsh/fish/powershell
- [x] `arx config validate` — Standalone config validation
- [x] `arx doctor` — Diagnostics (5 checks: project, config, detectors, git, version)
- [x] C# (.NET) detector — `.csproj`/`.sln` + `using` directives
- [x] JUnit XML output — Jenkins/GitLab/CircleCI compatible
- [x] GitHub Annotations output — PR inline comments
- [x] `Makefile` — build/test/lint/clean targets
- [x] `CHANGELOG.md` — Full release history
- [x] Fix deprecated APIs — `strings.Title()`, `filepath.HasPrefix()`

## ✅ v0.9.0 (Overrides, Rust, GitHub Action)

- [x] `overrides[]` per-rule — Path-based severity downgrade and rule disable
- [x] Rust detector — `Cargo.toml` detection, `use` statement parsing
- [x] `.github/actions/arx-action/` — GitHub Action for CI/CD
- [x] Override-aware exit code — 0 when only overridden violations remain
- [x] JSON `overridden_count` in summary

## ✅ v0.8.0 (Kotlin, Watch, Hooks, Custom Rules)

- [x] Kotlin detector — `.kt` files, `build.gradle.kts` support, import alias
- [x] `arx check --watch` — Continuous file monitoring with fsnotify
- [x] `arx hook install` — Git pre-commit hook
- [x] Custom rule `pattern` field — Regex matching on import paths

## ✅ v0.7.0 (Baseline, Diff, Cache)

- [x] `arx baseline` — Suppress existing violations for incremental adoption
- [x] `arx diff` — Compare architecture between git refs
- [x] Performance cache — Only re-parse changed files
- [x] Baseline-aware CI — Exit 0 if no new violations

## ✅ v0.6.0 (Java Detector + Audit)

- [x] Java detector — Maven/Gradle projects, `package` + `import` parsing
- [x] `arx audit` — Health reports with coupling matrix, debt score, trends
- [x] History persistence — `.arx-history/` with retention policy (max 10)

## ✅ v0.5.0 (Presets + Diagrams)

- [x] `arx init --preset {clean,hexagonal,ddd}`
- [x] `arx diagram` — ASCII + Graphviz DOT
- [x] Violation highlighting in diagrams

## ✅ v0.4.0 (Python Detector)

- [x] Python AST-based detector

## ✅ v0.3.0 (Explain + Circular Detection)

- [x] `arx explain <id>` — Detailed fix guidance
- [x] Circular dependency detection

## ✅ v0.2.0 (SARIF + Markdown)

- [x] SARIF and Markdown output formats

## ✅ v0.1.0 (MVP)

- [x] Go and TypeScript detectors
- [x] Basic `arx check` command

---

## 🔜 Future (v0.14.0+)

### PHP Detector
**Priority:** Medium | **Effort:** S

Support for PHP projects via `use`/`require` statement parsing with Composer detection.

### Swift Detector
**Priority:** Medium | **Effort:** M

Support for Swift projects via `import` statement parsing with SPM detection.

### Custom Rule DSL
**Priority:** Low | **Effort:** XL

Domain-specific language for complex architectural rules with JavaScript/TypeScript
evaluation engine. Access to full dependency graph, custom violation messages.

### Arx Server (Web UI)
**Priority:** Low | **Effort:** XL

Web interface for architecture visualization, violation timeline, team collaboration,
and interactive dependency graphs.

### VSCode Extension
**Priority:** Low | **Effort:** M

VSCode extension showing violations inline in the editor, with quick-fix suggestions.

### Cross-Language Dependency Resolution
**Priority:** Low | **Effort:** XL

Detect cross-language dependencies (e.g., gRPC proto → TypeScript client, REST API contracts).
