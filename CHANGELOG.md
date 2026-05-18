# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.28.0] - 2026-05

### Added
- `CheckExpr` type supporting `string | []string` rule check expressions (multi-line AND-chained)
- `all()`/`any()` built-in functions for dependency aggregation checks
- `functions` config section for user-defined expression functions with DAG validation
- Cycle detection for user-defined functions via Kahn's topological sort
- Schema update: `check` field now accepts both string and array of strings; `functions` object property added

## [v0.24.0] - 2026-05

### Added
- `arx server` — Web server with interactive dashboard and REST API
- Live dashboard with violation summary, coupling matrix, and debt score
- REST API endpoints: `/api/health`, `/api/status`, `/api/violations`, `/api/coupling`, `/api/debt`
- Auto-refresh via 30s ticker and fsnotify file watcher (debounced 500ms)
- Responsive CSS with print-friendly styles and dark/light theming
- Vanilla JS polling for real-time updates without page reload
- `--port`, `--bind`, and `--path` flags for server configuration
- Graceful shutdown on SIGINT/SIGTERM

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

### Changed
- Refactored detector architecture for easier language additions
- Improved performance for large codebases

## [v0.7.0] - 2026-02

### Added
- Kotlin detector for Android and JVM projects
- Python detector for Django and Flask projects
- TypeScript/JavaScript detector for Node.js projects
- Java detector for Maven and Gradle projects
- Go detector for Go modules

### Changed
- Unified detector interface across all languages
- Added language-specific import parsing

## [v0.6.0] - 2026-01

### Added
- `baseline` command for managing architectural debt baseline
- `--update-baseline` flag to `check` command
- Baseline storage in `.arx/baseline.json`
- Baseline diff reporting

### Fixed
- False positives in layer path matching
- Edge cases with glob pattern matching

## [v0.5.0] - 2025-12

### Added
- `hook` command for Git pre-commit integration
- `--install` flag to install Git hook automatically
- `--uninstall` flag to remove Git hook
- Pre-commit hook returns non-zero exit on violations

### Changed
- Improved error messages for configuration issues
- Better handling of missing configuration files

## [v0.4.0] - 2025-11

### Added
- `--explain` flag to `check` command for detailed violation explanations
- Built-in explanation library for common architectural violations
- Fix guidance with actionable steps for each violation type
- Pattern-based explanation matching

### Changed
- Enhanced violation output with rule ID and severity
- Improved help text for all commands

## [v0.3.0] - 2025-10

### Added
- `init` command for generating starter `arx.yaml` configuration
- Preset architectures: hexagonal, layered, clean, onion
- `--preset` flag to `init` command
- Interactive preset selection

### Changed
- Configuration file format improved with better defaults
- Layer path matching now supports glob patterns

## [v0.2.0] - 2025-09

### Added
- `check` command as default action when no command specified
- `--config` flag to specify custom config file path
- `--verbose` flag for detailed output
- Support for multiple target layers in rules

### Fixed
- YAML parsing errors with empty rule lists
- Layer matching edge cases with nested directories

## [v0.1.0] - 2025-08

### Added
- Initial release of Arx architectural linter
- Core architecture checking engine
- Layer-based dependency validation
- YAML configuration format
- Basic terminal output for violations
- Support for custom architectural rules

### Changed
- Foundation for hexagonal architecture enforcement

---

## Version History Summary

| Version | Release Date | Key Feature |
|---------|-------------|-------------|
| v0.1.0  | 2025-08     | Initial release |
| v0.2.0  | 2025-09     | Default check command |
| v0.3.0  | 2025-10     | Init command with presets |
| v0.4.0  | 2025-11     | Violation explanations |
| v0.5.0  | 2025-12     | Git hook integration |
| v0.6.0  | 2026-01     | Baseline management |
| v0.7.0  | 2026-02     | Multi-language detectors |
| v0.8.0  | 2026-03     | Audit and circular detection |
| v0.9.0  | 2026-04     | Rule overrides |
| v0.10.0 | 2026-05     | Diagram CLI & DX improvements |
| v0.28.0 | 2026-05     | Custom Rule DSL (expressions, functions, multi-line checks) |
| v0.24.0 | 2026-05     | Web server + dashboard |
