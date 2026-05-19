# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v50.0] - 2026-05-19

### Added
- Fuzz seed corpora for all 13 fuzz functions (expression, 10 languages, config)
- Flaky test detection: `go test -count=5 -race` CI gate for core packages
- Benchstat hard fail: >5% regression on DetectionPipeline_10k blocks CI
- `docs/roadmap.md` updated for v40-v50 including v50.0 completion

### Changed
- Domain coverage: 88.5% → 89.9% (config.go Validate: 71% → 90%, template.go checkParamType: 76% → 91%, toInt: 50% → 100%)
- Application coverage: Phase 2 complete at 79.6% with diagram, importscan, suggest, diff, doctor tests
- Dogfooding: all 5 C-01 circular dependency violations fixed (1 pre-existing masked remaining)
- .arx-cache directory permissions hardened from 0755 to 0700
- `.gitignore` now covers `.arx-backup/` and `.arx-baseline-history/`
- README.md docs table updated with Rule Testing, Workspace Mode, Dashboard/SSE references
- Makefile bench-compare target now hard-fails (>5% regression on DetectionPipeline_10k)
- GitHub CI: added flaky test detection step with go test -count=5 -race
- Benchstat regression checked with hard fail on >5% for DetectionPipeline_10k

### Fixed
- Security: .arx-cache now created with 0700 permissions instead of 0755
- .gitignore now covers `.arx-backup/` and `.arx-baseline-history/`

### Coverage
- `internal/domain` config.go: Validate 89.8%, compileFunctions 97.7%, validateTemplateLayerRefs 88.9%
- `internal/domain` template.go: checkParamType 90.5%, toInt 100%, toStrSlice 87.5%, resolveSourceLayer 100%
- All existing fuzz tests supplemented with hand-crafted seeds from real fixture code

---

## [v0.32.0] - 2026-05-18

### Added
- Cross-language dependency resolution (MVP) — detects proto→generated code relationships
- `Language` field on Dependency records — all 10 language detectors populate it
- `cross_language.mappings` config section with source_pattern, generated_pattern, and language
- `CrossLanguageDetector` — glob matching, stem-based matching, header verification
- Synthetic `Dependency{Language: "cross"}` linking proto definitions to their generated code
- `GetDetectorsForConfig()` convenience function for config-aware detector lists

## [v0.31.0] - 2026-05-18

### Added
- `arx fmt` command — formats arx.yaml with consistent YAML indentation and key order
- `arx fmt --check` flag — exits with code 1 if file is not formatted (CI use)
- `arx init --detect` flag — dry-run scan showing detected layers without writing config
- Dashboard config reloaded indicator — shows ⚡ flash when arx.yaml changes
- `/api/reload` endpoint triggers the config reloaded indicator
- Pre-commit hook and docs links in dashboard footer

### Changed
- Server state tracks `ConfigReloaded` flag, reset after next successful check

## [v0.30.0] - 2026-05-18

### Added
- `filter(deps(a,b), "field op value")` — filters dependencies by predicate string
- `map(deps(a,b), "field")` — extracts field values into new `ValueList` type
- `ValueList` value type with `count()` support
- Predicate evaluator: `==`/`!=` for string fields (SourceFile, ImportPath, ResolvedLayer)
- Predicate evaluator: all 6 comparison operators for SourceLine (numeric)
- Tokenizer enhancement: quoted string support for predicate arguments
- 42 new tests across filter, map, predicate evaluation, and ValueList

## [v0.29.0] - 2026-05-18

### Added
- `POST /api/reload` endpoint — forces config re-read and full architecture re-check
- `GET /api/config` endpoint — returns current config summary (layers, rules, functions)
- File watcher now logs when `arx.yaml` changes — config hot-reload without server restart
- `isConfigPath()` helper for detecting config file changes in watcher events

## [v0.28.0] - 2026-05-18

### Added
- `CheckExpr` type supporting `string | []string` rule check expressions (multi-line AND-chained)
- `all()`/`any()` built-in functions for dependency aggregation checks
- `functions` config section for user-defined expression functions with DAG validation
- Cycle detection for user-defined functions via Kahn's topological sort
- Cross-function references (function A can call function B)
- Builtin shadowing protection at config load time
- Schema update: `check` field now accepts both string and array of strings; `functions` object property added

## [v0.27.0] - 2026-05-18

### Added
- `arx suggest` command — analyzes violations and shows concrete fix suggestions as unified diffs
- `FixEngine` with template-based fix generation (Go first: domain→infrastructure, application→infrastructure)
- `arx suggest --apply` — auto-applies fixes with `.arx-backup/` safety directory
- `arx suggest --force` — skips confirmation prompt
- `arx suggest --output` — writes diffs to file
- Atomic rollback on error (all-or-nothing file restoration)
- 18 new tests (12 CLI + 6 engine)

## [v0.26.0] - 2026-05

### Added
- Performance metrics: check duration, files scanned, total deps, detectors run, uptime
- `/api/metrics` endpoint + metric cards on dashboard
- `arx config set` supports dotted paths (`severity_mapping.critical`), JSON arrays, numbers
- `arx config get` supports dotted paths for complex values
- Metrics JSON round-trip serialization

### Changed
- Quality pass: `go vet` clean, `go test -race` — 0 data races

## [v0.25.0] - 2026-05

### Added
- Dashboard filtering by severity (checkboxes), layer (dropdown), and search text
- Sortable violation columns (asc/desc/none with visual arrows)
- Filter summary ("Showing X of Y violations") + empty state
- Clear filters button
- Server state persistence (`.arx-cache/server-state.json`) — survives server restart
- `arx check --diff` — shows violations added/removed since last check run
- Color-coded diff output: red (new), green (resolved), dim (unchanged)

## [v0.24.0] - 2026-05

### Added
- `arx server` — Web server with interactive dashboard and REST API
- Live dashboard with violation summary, coupling matrix, and debt score
- REST API endpoints: `/api/health`, `/api/status`, `/api/violations`, `/api/coupling`, `/api/debt`
- Auto-refresh via 30s ticker and fsnotify file watcher (debounced 500ms)
- Responsive CSS with print-friendly styles
- Vanilla JS polling for real-time updates without page reload
- `--port`, `--bind`, and `--path` flags for server configuration
- Graceful shutdown on SIGINT/SIGTERM

## [v0.23.0] - 2026-05

### Added
- End-to-end tests for all language detection fixtures
- Hardened cross-language test files for Go, TypeScript, Python, Java, Kotlin
- Robust Rust, C#, Ruby, PHP, and Swift test fixtures
- Integration test coverage for C# detector with .csproj and .sln fixtures

### Changed
- Restructured test fixtures to avoid cross-detector interference

## [v0.22.0] - 2026-05

### Added
- `arx config set` command with preset-based configuration generation
- `arx config get` command for viewing current configuration values
- `arx check --severity` flag to filter results by severity level
- Schema generation for YAML autocompletion in IDEs

### Changed
- Config package refactored for better separation of concerns

## [v0.21.0] - 2026-05

### Added
- Full HTML audit report with coupling matrix, debt score, and trend analysis
- JSON check output with coupling matrix and detector metadata
- `arx audit` command — comprehensive health report
- Detector status reporting per language

### Changed
- Quality pass: cleanup deprecated API usage, improved error messages

## [v0.20.0] - 2026-05

### Added
- JSON Schema for `arx.yaml` IDE autocompletion (`$schema` support)
- `NO_COLOR` support for CI environments
- Smart `arx init` — auto-detects `.gitignore` and adds arx-specific entries
- Verbose check output with per-detector status
- `--force` flag to overwrite existing configuration

## [v0.19.0] - 2026-05

### Added
- Extended expression engine: `count()`, `deps()`, `layers()`, `has_circular()`, `files()`, `ratio()`, `violations()`, `threshold()` builtins
- Expression-based rules via `check` field in rule definitions
- Mixing prevention: expression rules cannot use `from`/`to`/`template`/`pattern`

### Changed
- Expression engine rewritten with recursive-descent parser and typed AST

## [v0.18.0] - 2026-05

### Added
- Rule template system: `max-deps`, `no-leak`, `layer-balance` templates
- Template parameter validation with schema definitions
- Template-to-expression compilation
- Configurable template parameters via `params` section in rules

### Changed
- Rule evaluation split into three paths: standard from/to, expression check, template

## [v0.17.0] - 2026-05

### Added
- Preset system with Clean Architecture, Hexagonal, DDD, Layered, Onion templates
- `arx init --preset` flag for preset-based initialization
- Preset service for loading and managing architecture templates
- Preset documentation and usage examples

## [v0.16.0] - 2026-05

### Added
- Comprehensive audit command with HTML output
- Coupling matrix percentage calculation
- Debt score with severity breakdown (error/warning/info weighting)
- Trend tracking across multiple check runs
- `arx audit` report with architectural health overview

## [v0.15.0] - 2026-05

### Added
- Man page generation (`arx man`) for Linux distributions
- `.arxignore` file support for excluding files/directories
- `.arx-cache/` directory management with `.gitignore` auto-entries
- Path exclusion patterns for rule evaluation

### Changed
- Ignore and hygiene improvements for production deployments

## [v0.14.0] - 2026-05

### Added
- PHP detector with `use`/`use as`/`use function` import parsing
- Swift detector with `import`/`@_exported import` support
- Multi-language test fixtures for integration testing

## [v0.13.0] - 2026-05

### Added
- Ruby detector for Gemfile-based projects
- Fuzz tests for Java, C#, Rust, and Ruby parsers
- Expanded detector test coverage

## [v0.12.0] - 2026-05

### Added
- Release automation with GoReleaser (multi-platform binaries)
- Homebrew formula generation for macOS/Linux
- RPM and DEB package builds
- Docker image publishing to GHCR
- JSON Schema for arx.yaml

### Changed
- Maturity pass: release infrastructure, CI/CD readiness

## [v0.11.0] - 2026-05

### Added
- CI/CD GitHub Action for automated architecture checking
- HTML report output format
- SARIF output format for GitHub Code Scanning
- Markdown output format
- GitHub Annotations output for workflow integration

### Changed
- Quality pass: enhanced output formats for CI integration

## [v0.10.0] - 2026-05

### Added
- `diagram` command with ASCII, DOT, and Mermaid output formats
- C# detector for .NET projects (`.csproj` and `.sln` files)
- Shell completion command (`arx completion bash|zsh|fish|powershell`)
- `arx config validate` command for config file validation
- `arx doctor` command for project health checks
- JUnit XML output format for CI integration
- GitHub Annotations output format for workflow integration
- Makefile with build, test, lint, clean, and help targets

### Changed
- Replaced deprecated `strings.Title()` with `cases.Title(language.Und)`
- Replaced `filepath.HasPrefix()` with `strings.HasPrefix()`

### Fixed
- Deprecated API warnings resolved across codebase

## [v0.9.0] - 2026-04

### Added
- Per-directory rule overrides with `overrides` field in rules
- Rust detector for Cargo projects
- GitHub Action for automated architecture checking
- `--format` flag to `check` command (text, json, markdown)
- Baseline diff feature for tracking architectural debt changes

### Changed
- Improved violation explanations with actionable guidance
- Enhanced terminal output with color-coded severity

## [v0.8.0] - 2026-03

### Added
- `audit` command for comprehensive architectural analysis
- Circular dependency detection
- Coupling analysis metrics
- Trend analysis for architectural debt
- `--watch` flag for continuous checking during development
- Kotlin detector for Android and JVM projects

### Changed
- Refactored detector architecture for easier language additions
- Improved performance for large codebases

## [v0.7.0] - 2026-02

### Added
- Performance cache: file-hash-keyed detector result caching (`.arx-cache/`)
- `arx baseline` command — generates `.arx-baseline.json` from current violations
- Baseline-aware `arx check` — fails only on NEW violations (existing ones suppressed)
- `arx diff <ref-before> <ref-after>` — compares architecture between git refs
- Python detector for Django and Flask projects
- TypeScript/JavaScript detector for Node.js projects
- Java detector for Maven and Gradle projects
- Go detector for Go modules

### Changed
- Unified detector interface across all languages
- Added language-specific import parsing
- Check exit code changes when baseline exists (only new violations cause failure)

## [v0.6.0] - 2026-01

### Added
- `baseline` command for managing architectural debt baseline
- `--update-baseline` flag to `check` command
- Baseline storage in `.arx/baseline.json`
- Baseline diff reporting
- Java detector (initial version)

### Fixed
- False positives in layer path matching
- Edge cases with glob pattern matching

## [v0.5.0] - 2025-12

### Added
- `hook` command for Git pre-commit integration
- `--install` flag to install Git hook automatically
- `--uninstall` flag to remove Git hook
- Pre-commit hook returns non-zero exit on violations
- Architecture diagram generation (Mermaid)

### Changed
- Improved error messages for configuration issues
- Better handling of missing configuration files

## [v0.4.0] - 2025-11

### Added
- `--explain` flag to `check` command for detailed violation explanations
- Built-in explanation library for common architectural violations
- Fix guidance with actionable steps for each violation type
- Pattern-based explanation matching
- TypeScript/JavaScript detector (initial version)

### Changed
- Enhanced violation output with rule ID and severity
- Improved help text for all commands

## [v0.3.0] - 2025-10

### Added
- `init` command for generating starter `arx.yaml` configuration
- Preset architectures: hexagonal, layered, clean, onion
- `--preset` flag to `init` command
- Interactive preset selection
- Python detector (initial version)

### Changed
- Configuration file format improved with better defaults
- Layer path matching now supports glob patterns

## [v0.2.0] - 2025-09

### Added
- `check` command as default action when no command specified
- `--config` flag to specify custom config file path
- `--verbose` flag for detailed output
- Support for multiple target layers in rules
- Go detector (initial version)

### Fixed
- YAML parsing errors with empty rule lists
- Layer matching edge cases with nested directories

## [v0.1.0] - 2025-08

### Added
- Initial release of Arx — cross-language architecture audit CLI
- Core architecture checking engine with hexagonal validation
- Layer-based dependency validation (Cannot/Must rules)
- YAML configuration format (`arx.yaml`)
- Basic terminal output for violations
- Support for custom architectural rules with severity levels

---

## Version History Summary

| Version | Release Date | Key Feature |
|---------|-------------|-------------|
| v0.32.0 | 2026-05-18  | Cross-language dependency resolution |
| v0.31.0 | 2026-05-18  | arx fmt, init --detect, dashboard QoL |
| v0.30.0 | 2026-05-18  | filter()/map() DSL completion |
| v0.29.0 | 2026-05-18  | Rule hot-reload (/api/reload, /api/config) |
| v0.28.0 | 2026-05-18  | Custom Rule DSL (multi-line, all/any, user functions) |
| v0.27.0 | 2026-05-18  | arx suggest (auto-fix suggestions) |
| v0.26.0 | 2026-05     | Performance metrics, config improvements |
| v0.25.0 | 2026-05     | Dashboard filters, state persistence, check --diff |
| v0.24.0 | 2026-05     | Web server + dashboard |
| v0.23.0 | 2026-05     | E2E testing for all language fixtures |
| v0.22.0 | 2026-05     | Config set/get, severity filtering |
| v0.21.0 | 2026-05     | Audit HTML, JSON metadata, quality pass |
| v0.20.0 | 2026-05     | JSON Schema, NO_COLOR, smart init |
| v0.19.0 | 2026-05     | Extended expression engine (8 builtins) |
| v0.18.0 | 2026-05     | Rule template system |
| v0.17.0 | 2026-05     | Architecture presets (Clean, Hex, DDD) |
| v0.16.0 | 2026-05     | Comprehensive audit with debt score |
| v0.15.0 | 2026-05     | Man pages, .arxignore, path exclusions |
| v0.14.0 | 2026-05     | PHP + Swift detectors |
| v0.13.0 | 2026-05     | Ruby detector + fuzz testing |
| v0.12.0 | 2026-05     | GoReleaser, Homebrew, Docker, packages |
| v0.11.0 | 2026-05     | CI/CD action, SARIF, Markdown output |
| v0.10.0 | 2026-05     | Diagram CLI, C# detector, DX improvements |
| v0.9.0  | 2026-04     | Rule overrides, Rust detector, GitHub Action |
| v0.8.0  | 2026-03     | Audit, circular detection, Kotlin detector |
| v0.7.0  | 2026-02     | Baseline, diff mode, performance cache, Python+Java+TS+Go |
| v0.6.0  | 2026-01     | Baseline command, Java detector |
| v0.5.0  | 2025-12     | Git hook, architecture diagrams |
| v0.4.0  | 2025-11     | Violation explanations, TypeScript detector |
| v0.3.0  | 2025-10     | Init command, presets, Python detector |
| v0.2.0  | 2025-09     | Check command, Go detector |
| v0.1.0  | 2025-08     | Initial release |