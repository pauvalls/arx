# Roadmap

## ✅ v0.13.0 (Current — Ruby Detector + Fuzz Tests)

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
