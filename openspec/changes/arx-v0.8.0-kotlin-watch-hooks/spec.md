# Arx v0.8.0 — Kotlin Detector, Watch Mode, Pre-commit Hooks, Custom Rules Specification

---

## Domain: kotlin-detector (NEW)

### Requirement: Kotlin Project Detection

The system MUST detect Kotlin projects by recognising `.kt` files and build configuration files at the project root. Detection follows the same pattern as the existing Java detector but handles Kotlin-specific build files.

#### Scenario: Gradle Kotlin DSL project

- GIVEN a project root with `build.gradle.kts` or `settings.gradle.kts`
- WHEN `KotlinDetector.Detect()` is called
- THEN returns `true` indicating a Kotlin project

#### Scenario: Maven project with Kotlin

- GIVEN a project root with `pom.xml` and `.kt` files in `src/main/kotlin/`
- WHEN `KotlinDetector.Detect()` is called
- THEN returns `true` (mixed Java/Kotlin detected)

#### Scenario: Non-Kotlin project

- GIVEN a project root without `.kt` files and without `build.gradle.kts`
- WHEN `KotlinDetector.Detect()` is called
- THEN returns `false`

### Requirement: Kotlin Import Extraction

The system MUST extract `import` and `package` declarations from `.kt` files using regex, supporting Kotlin-specific syntax.

#### Scenario: Standard import

- GIVEN a `.kt` file with `import org.springframework.boot.autoconfigure.SpringBootApplication`
- WHEN `ExtractImports()` is called
- THEN extracts the full import path with line number

#### Scenario: Wildcard import

- GIVEN a `.kt` file with `import com.example.domain.*`
- WHEN `ExtractImports()` is called
- THEN extracts the wildcard import

#### Scenario: Package declaration

- GIVEN a `.kt` file with `package com.example.domain.order`
- WHEN `ExtractImports()` is called
- THEN extracts the package declaration for layer resolution

#### Scenario: Skip external dependencies

- GIVEN imports like `kotlin.collections.List` or `java.util.UUID`
- WHEN `ExtractImports()` resolves to layers
- THEN marks them as external dependencies (no internal rule violations)

#### Scenario: Skip generated and test files

- GIVEN files under `build/`, `target/`, or matching `**/*Test.kt`
- WHEN `ExtractImports()` walks the project
- THEN excludes those files from analysis

---

## Domain: watch-mode (NEW)

### Requirement: Watch Command Interface

The system MUST provide `arx check --watch` to continuously monitor file changes and re-run the check automatically.

#### Scenario: Initial check then watch

- GIVEN a valid project with `arx.yaml`
- WHEN `arx check --watch` is executed
- THEN runs a full `arx check` immediately
- AND starts the file watcher for subsequent changes

#### Scenario: File change triggers re-check

- GIVEN the watcher is running
- WHEN a `.kt`, `.java`, `.go`, or `.ts` file is modified
- THEN re-runs `arx check` on the changed project only (debounced)

#### Scenario: Debounce prevents rapid re-runs

- GIVEN multiple file changes within 500ms
- WHEN the watcher receives events
- THEN waits 500ms after the last event before running the check
- AND runs the check exactly once

#### Scenario: Diff summary after re-check

- GIVEN check completed with 5 violations previously
- WHEN re-check finds 3 new violations and 1 resolved
- THEN prints "3 new violations, 1 resolved"

#### Scenario: JSON diff on each change

- GIVEN `--watch --json` flags
- WHEN a re-check produces a diff
- THEN outputs a JSON object with `added`, `resolved`, and `unchanged` violation arrays

#### Scenario: Graceful shutdown

- GIVEN the watcher is running
- WHEN Ctrl+C or SIGINT is received
- THEN prints summary and exits cleanly with code 0

### Requirement: Watch Configuration

The system SHOULD respect `.gitignore` patterns and support configurable polling.

#### Scenario: Respect .gitignore

- GIVEN a `.gitignore` excluding `vendor/`
- WHEN the watcher detects a change in `vendor/`
- THEN ignores the event and does not re-check

#### Scenario: Configurable polling interval

- GIVEN `--watch --interval 2s`
- WHEN the watcher starts
- THEN uses 2-second polling instead of the default 500ms debounce

---

## Domain: pre-commit-hook (NEW)

### Requirement: Hook Installation

The system MUST provide `arx hook install` to create a git pre-commit hook script.

#### Scenario: Install in git repository

- GIVEN a valid git repository with `.git/hooks/` directory
- WHEN `arx hook install` is executed
- THEN creates `.git/hooks/pre-commit` as an executable POSIX shell script
- AND the script invokes `arx check` on staged files

#### Scenario: Non-git directory error

- GIVEN a directory that is not a git repository
- WHEN `arx hook install` is executed
- THEN returns a clear error: "not a git repository"

#### Scenario: Uninstall removes hook

- GIVEN a previously installed hook at `.git/hooks/pre-commit`
- WHEN `arx hook uninstall` is executed
- THEN removes `.git/hooks/pre-commit`

### Requirement: Hook Execution Behavior

The pre-commit hook MUST run `arx check` on staged files and block commits only for NEW violations, respecting the baseline.

#### Scenario: Block commit on new violations

- GIVEN staged files that introduce new violations not in `.arx-baseline.json`
- WHEN the pre-commit hook runs
- THEN exits with code 1
- AND blocks the commit

#### Scenario: Allow commit with suppressed violations only

- GIVEN staged files whose violations are all present in `.arx-baseline.json`
- WHEN the pre-commit hook runs
- THEN exits with code 0
- AND allows the commit

### Requirement: Hook Bypass

The system MUST support hook bypass via environment variable.

#### Scenario: SKIP environment variable bypass

- GIVEN the environment variable `SKIP=arx`
- WHEN `git commit` triggers the pre-commit hook
- THEN the hook exits immediately with code 0
- AND the commit proceeds

---

## Domain: custom-rules-patterns (NEW)

### Requirement: Pattern Field on Rules

The system MUST support a `pattern` field on rules in `arx.yaml` that matches import paths via glob/regex.

#### Scenario: Block legacy package imports

- GIVEN a rule with `pattern: "com/legacy/**"` configured in `arx.yaml`
- WHEN `arx check` finds an import matching `com.legacy.util.OldClass`
- THEN reports a violation for that import

#### Scenario: Match specific controllers

- GIVEN a rule with `pattern: "com/example/.*Controller"`
- WHEN `arx check` processes `com.example.UserController`
- THEN matches the pattern and applies the rule

#### Scenario: Combined with from/to layer rules

- GIVEN a rule with both `pattern` and `from: "application"`, `to: "domain"`
- WHEN an import matches the pattern AND the from/to layers
- THEN reports a violation only when all conditions match

### Requirement: Pattern Validation

The system MUST validate `pattern` fields at config load time.

#### Scenario: Invalid regex rejected

- GIVEN `pattern: "[invalid"` in `arx.yaml`
- WHEN config is loaded
- THEN returns a clear validation error indicating the invalid regex
- AND `arx check` fails to start

#### Scenario: Valid regex accepted

- GIVEN `pattern: "com/example/.*"` in `arx.yaml`
- WHEN config is loaded
- THEN compiles and caches the regex successfully

### Requirement: Pattern Performance

The system MUST compile `pattern` rules once at startup and cache them.

#### Scenario: Single compilation at startup

- GIVEN a config with 10 pattern rules
- WHEN `arx check` is invoked
- THEN all 10 patterns are compiled once during config load
- AND the compiled `*regexp.Regexp` values are cached for the lifetime of the process

---

## Domain: check-command (MODIFIED)

### ADDED Requirements for v0.8.0

### Requirement: Watch Flag

The `arx check` command MUST accept a `--watch` flag that enables continuous file monitoring.

#### Scenario: --watch flag presence

- GIVEN `arx check --help`
- THEN the output includes `--watch` flag description

#### Scenario: --watch with verbose

- GIVEN `arx check --watch --verbose`
- WHEN the watcher detects a file change
- THEN prints each changed file path to stderr before re-checking

#### Scenario: --watch --interval flag

- GIVEN `arx check --watch --interval 1s`
- WHEN the watcher starts
- THEN uses 1-second polling instead of default 500ms debounce

### Requirement: Hook-Compatible Exit Codes

The `arx check` command MUST return exit code 0 when only suppressed (baseline) violations exist, enabling hook compatibility.

#### Scenario: Exit 0 with suppressed violations only

- GIVEN a baseline with existing violations
- GIVEN no new violations detected
- WHEN `arx check` completes
- THEN exits with code 0

#### Scenario: Exit 1 with new violations

- GIVEN a baseline with existing violations
- GIVEN new violations detected beyond baseline
- WHEN `arx check` completes
- THEN exits with code 1

#### Scenario: --no-baseline overrides hook behavior

- GIVEN `--no-baseline` flag
- WHEN `arx check` is invoked in a hook context
- THEN runs check without baseline filtering
- AND reports all violations

---

## Acceptance Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-01 | KotlinDetector detects Gradle Kotlin DSL (`build.gradle.kts`) | Test fixture → `Detect() = true` |
| AC-02 | KotlinDetector detects Maven + `.kt` files | Test fixture with `pom.xml` + `.kt` → `Detect() = true` |
| AC-03 | Kotlin import extraction precision ≥90% | 100 known Kotlin imports → ≥90 correct |
| AC-04 | `arx check --watch` re-runs within 1s of file change | Integration test with temp file write |
| AC-05 | Watch mode diff summary shows added/resolved counts | Integration test with known violation delta |
| AC-06 | `arx hook install` creates executable pre-commit | Verify `.git/hooks/pre-commit` exists and is executable |
| AC-07 | Hook blocks commit on new violations; allows on suppressed only | Seeded baseline → staged add/remove → assert exit code |
| AC-08 | `SKIP=arx` bypasses hook | Env set → hook exits 0 without running check |
| AC-09 | Pattern rule `com/legacy/**` blocks matching imports | Config with pattern → `arx check` reports violation |
| AC-10 | Invalid regex in pattern fails config load with clear error | Config with `[invalid` → validation error at startup |
| AC-11 | Pattern compiled once, cached | Check runs 2x, regex compilation count = 1 |
| AC-12 | All v0.7.0 tests pass without modification | `go test ./...` on main branch |

| NFC ID | Criterion | Target |
|--------|-----------|--------|
| NFC-01 | Watch mode memory on 10K file project | < 50 MB RSS |
| NFC-02 | Pre-commit hook execution time | < 2s on typical staged diff |
| NFC-03 | Test coverage on new code | ≥80% |
| NFC-04 | Binary size increase vs v0.7.0 | < 1.5 MB |

---

## Artifacts

| Artifact | Path |
|----------|------|
| Spec document | `openspec/changes/arx-v0.8.0-kotlin-watch-hooks/spec.md` |
| Kotlin detector | `internal/infrastructure/detector/kotlin_detector.go` |
| File watcher | `internal/infrastructure/watch/fswatcher.go` |
| Hook command | `cmd/arx/hook.go` |
| Check command (modified) | `cmd/arx/check.go` |
| Rule model (modified) | `internal/domain/rule.go` |
| Config schema (modified) | `arx.yaml` schema |

---

## Next Step

Ready for **design phase** (sdd-design). Design must define:
- KotlinDetector struct and method signatures (matching JavaDetector contract)
- Watcher service — fsnotify init, debounce timer, event filtering
- Pre-commit hook script template
- `Rule.Pattern` field — string + compiled `*regexp.Regexp`
- Check service — diff computation between runs for watch mode output
