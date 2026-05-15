# Design: Arx v0.9.0 — Action Overrides & Rust Detector

## Technical Approach

Four additive features, each independent. Implementation order per proposal: per-directory overrides → rust detector → github action → check-command modifications. Overrides are foundational domain logic; Rust detector follows the exact Kotlin/Java detector pattern; GitHub Action wraps existing `arx check --ci --format sarif`; check-command modifications integrate override filtering into the existing pipeline.

## Architecture Decisions

### Decision: RuleOverride as embedded slice on Rule (not separate config section)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Separate `overrides` top-level section in config | Parallel structure, harder to correlate with rules at a glance | Rejected |
| `overrides[]` inline on each `Rule` | Co-located with the rule they modify, natural YAML, `omitempty` for backward compat | **Chosen** |

Override data must live with the rule it modifies — scanning a separate section requires cross-referencing by rule ID. Inline is self-documenting.

### Decision: Override evaluation at violation time, not dependency time

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Filter dependencies before evaluating rules | Misses overrides that change severity but don't disable — severity is a violation-level concept | Rejected |
| Evaluate rules fully, then filter/post-process violations | Keeps `EvaluateRules` pure; override logic is a separate concern layered on top | **Chosen** |

Overrides affect *violation reporting*, not dependency detection. A rule still "matches" — the override only changes whether it surfaces and at what severity.

### Decision: Override path matching uses prefix match (no globs)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Full glob/regex path patterns | Powerful but inconsistent with layer path matching complexity | Rejected |
| Simple prefix match (`strings.HasPrefix` normalized) | Predictable, fast, matches existing `Layer.MatchesPath` directory-prefix pattern | **Chosen** |

Users write `path: internal/legacy/` and it applies to everything under that tree. Clear, no surprises. If glob support is needed later, it's additive.

### Decision: RustDetector follows KotlinDetector pattern exactly

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom simplified Rust detector | Diverges from established pattern, different error handling | Rejected |
| Reuse Go import graph analysis | Wrong semantics — Rust module system is path/`use`-based, not package-path-based | Rejected |
| Exact mirror of Kotlin/Java detector structure | Same `Detect()` / `ExtractImports()` / `parseFile()` / `resolveImport()` / `isExternalDependency()` flow | **Chosen** |

Rust's `use` statements map directly to the import-extraction model. The Kotlin detector is the closest template because both have `modulePrefix`-like concepts (Rust crate root) and both distinguish external vs internal imports.

### Decision: GitHub Action as Docker action (not composite or JavaScript)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Composite action (runs arx via `npx` or pre-installed) | Requires Node or pre-installed binary on runner | Rejected |
| JavaScript action | Requires Node runtime, no Go binary | Rejected |
| Docker action | Self-contained, any runner, pins exact arx version | **Chosen** |

Docker action guarantees the exact arx binary version is used. No dependency on runner tooling. Tradeoff: slightly slower first-run (image pull).

### Decision: Overridden violations are separated at the check-result level, not in reporters

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Each reporter decides how to handle overrides | Duplicate logic across 4 reporters | Rejected |
| `runCheckWithService` splits violations into active/overridden, passes to reporters cleanly | Single responsibility, reporters remain unchanged | **Chosen** |

The `checkResult` struct gains `overriddenCount` and a filtered `violations` slice. Reporters see only active violations. JSON reporter adds `overridden_count` to `Summary`. Terminal verbose mode shows the count.

## Data Flow

### Per-directory overrides

```
arx.yaml rule entry:
  - id: domain-cannot-depend-on-infra
    overrides:
      - path: internal/legacy/
        enabled: false
      - path: internal/migration/
        severity: warning

Config.Validate()
  → Rule.Validate() validates override paths + severity values

EvaluateRules(deps, rules, layers)
  → for each violation:
      rule.IsEnabledFor(violation.File)? → skip if disabled
      effectiveSev := rule.GetEffectiveSeverity(violation.File)
      → set violation.Severity = effectiveSev

runCheckWithService():
  → violations = service.Evaluate(...)
  → split: active vs overridden
  → checkResult{ violations: active, overriddenCount: N }

printCheckResult():
  → verbose mode writes overridden count to stderr
  → JSON: includes overridden_count in Summary
  → ExitCode: 0 if only overridden violations remain
```

### Rust detector

```
Registry.GetDetectors()
  → append rust.New()

rust.Detect(projectRoot):
  Cargo.toml exists? → return true
  otherwise → return false

rust.ExtractImports():
  Walk src/ directory for *.rs files
    → skip *_test.rs and tests/ dir
    → parseFile():
        lines → extractImportsFromLine()
          → use path::to::Module          (standard)
          → use crate::path::to::Module   (crate-relative)
          → use self::path::to::Module    (self-relative)
          → use super::path::to::Module   (parent-relative)
          → pub use path::to::Module      (re-export)
          → pub mod module_name           (module declaration)
    → resolveImport():
        skip external: std::, core::, alloc::, test::
        convert :: to /
        match against layers
```

### GitHub Action

```
.github/workflows/arx-ci.yml:
  push / pull_request → checkout → arx-action

.github/actions/arx-action/action.yml:
  inputs: path, config, format, baseline, diagram
  runs: Docker
  entrypoint.sh:
    → arx check --ci --format sarif --config $INPUT_CONFIG $INPUT_PATH
    → if baseline: arx check --ci --format sarif --baseline $INPUT_BASELINE
    → if diagram: arx diagram ...

entrypoint.sh:
  #!/bin/sh
  arx check --ci \
    --format "${INPUT_FORMAT:-sarif}" \
    --config "${INPUT_CONFIG:-arx.yaml}" \
    "${INPUT_PATH:-.}"
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/rule.go` | Modify | Add `RuleOverride` struct, `Overrides []RuleOverride` field, `GetEffectiveSeverity()`, `IsEnabledFor()` |
| `internal/domain/rule_test.go` | Modify | Tests for override methods |
| `internal/domain/audit.go` | Modify | `EvaluateRules` passes file path to override check, sets severity from override |
| `internal/domain/audit_test.go` | Modify | Tests for override-aware evaluation |
| `internal/domain/config.go` | Modify | `Config.Validate()` validates overrides.path and overrides.severity |
| `internal/domain/violation.go` | Modify | (if needed) ensure `Severity` is populated from overrides |
| `internal/infrastructure/detector/rust/detector.go` | Create | `RustDetector` — Detect(), ExtractImports(), FindRustFiles() |
| `internal/infrastructure/detector/rust/parser.go` | Create | Regex patterns for Rust `use`, `pub use`, `pub mod` statements |
| `internal/infrastructure/detector/rust/detector_test.go` | Create | Tests for detector |
| `internal/infrastructure/detector/rust/parser_test.go` | Create | Tests for parser regex |
| `internal/infrastructure/detector/registry.go` | Modify | Import and register `rust.New()` |
| `internal/infrastructure/output/json.go` | Modify | Add `OverriddenCount` to `Summary`, populate in `Report()` |
| `internal/infrastructure/output/terminal.go` | Modify | `ExitCode()` returns 0 if only overridden violations remain; verbose mode shows count |
| `cmd/arx/check.go` | Modify | Split violations after `EvaluateRules`, populate `overriddenCount` in `checkResult` |
| `.github/actions/arx-action/action.yml` | Create | GitHub Action metadata with inputs |
| `.github/actions/arx-action/Dockerfile` | Create | Multi-stage build or pre-built binary copy |
| `.github/actions/arx-action/entrypoint.sh` | Create | Shell script wrapping `arx check --ci --format sarif` |
| `.github/actions/arx-action/Makefile` | Create | (optional) Build helpers |
| `.github/workflows/arx-ci.yml` | Create | CI workflow: push/PR → check → upload SARIF → upload diagram |

## Interfaces / Contracts

```go
// internal/domain/rule.go — new types
type RuleOverride struct {
    Path     string   `yaml:"path" json:"path"`
    Severity Severity `yaml:"severity,omitempty" json:"severity,omitempty"`
    Enabled  *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// Rule — new fields
type Rule struct {
    // ... existing fields
    Overrides []RuleOverride `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

// Rule — new methods
func (r *Rule) GetEffectiveSeverity(filePath string) (Severity, bool)
    // Returns override severity and true if an override matches the path.
    // Checks overrides by longest-prefix match (most specific wins).

func (r *Rule) IsEnabledFor(filePath string) bool
    // Returns false if any override matches filePath with Enabled=false.
    // If no override sets Enabled, returns true (rule is enabled by default).

// internal/infrastructure/output/json.go — modified types
type Summary struct {
    Total           int `json:"total"`
    Errors          int `json:"errors"`
    Warnings        int `json:"warnings"`
    Info            int `json:"info"`
    OverriddenCount int `json:"overridden_count,omitempty"`  // NEW
}

// internal/infrastructure/detector/rust/detector.go
type RustDetector struct {
    modulePrefix string  // crate name from Cargo.toml
    sourceDirs   []string
}

func New() *RustDetector

// Rust regex patterns (parser.go):
//   use\s+(crate::)?([a-zA-Z_][a-zA-Z0-9_:]*);             — use (crate-relative)
//   use\s+self::([a-zA-Z_][a-zA-Z0-9_:]*);                  — use self::
//   use\s+super::([a-zA-Z_][a-zA-Z0-9_:]*);                 — use super::
//   pub\s+use\s+([a-zA-Z_][a-zA-Z0-9_:]*);                 — re-export
//   pub\s+mod\s+([a-zA-Z_][a-zA-Z0-9_]+);                  — module declaration
// External skip prefixes: std::, core::, alloc::, test::

// cmd/arx/check.go — modified checkResult
type checkResult struct {
    violations      []domain.Violation
    overriddenCount int              // NEW
    suppressedCount int
    config          *domain.Config
    configHash      string
    projectRoot     string
    format          ports.OutputFormat
    duration        time.Duration
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `RuleOverride` methods | Table-driven: `GetEffectiveSeverity` with empty, non-matching, single-match, multi-match overrides |
| Unit | `Rule.IsEnabledFor` | Override with `Enabled: false` blocks violation; no override → enabled |
| Unit | `EvaluateRules` with overrides | Rule disabled for path → no violation; severity override → violation has override severity |
| Unit | Rust parser regex | Table-driven: `use std::collections::HashMap`, `use crate::domain::model`, `pub use`, `pub mod`, skip external |
| Unit | RustDetector.Detect | Fake `Cargo.toml` → true; empty dir → false |
| Unit | RustDetector.FindRustFiles | Walk `src/`, skip `*_test.rs` and `tests/` dir |
| Unit | RustDetector.resolveImport | Resolve `crate::domain::Model` to layer; skip `std::sync::Mutex` |
| Unit | JSON `Summary.OverriddenCount` | Serialize/deserialize with and without overridden count |
| Unit | `ExitCode` with overrides | Only overridden violations → exit 0; mixed → exit 1 |
| Integration | Rust detector on real Cargo project | ArxFakeProject with `Cargo.toml` + `src/lib.rs` → verify dependencies extracted |
| Integration | Override evaluation end-to-end | Config with overrides → check → verify violations filtered and severity adjusted |

Rust detector tests mirror Kotlin detector test structure. Override tests are table-driven unit tests on the domain model — no integration needed for core logic.

## Migration / Rollout

No migration required. All features are additive:
- `Overrides []RuleOverride` is `omitempty` — existing configs load unchanged
- Rust detector is a new registry entry — won't activate unless `Cargo.toml` exists
- `.github/actions/arx-action/` is opt-in — no existing CI pipeline affected
- `checkResult.overriddenCount` defaults to 0 — all existing callers unchanged
- `ExitCode` change (0 for only overridden) is backward-compatible: overrides are new, so this path never hits on existing configs

## Open Questions

- [x] Should override path be relative to project root or config path? **Project root** — consistent with layer paths. Resolved at config load time.
- [x] Multiple overrides match the same file? **Longest prefix wins** for severity. If any override sets `Enabled: false`, the rule is disabled regardless of other matches (disable is the strongest signal).
- [x] Rust module declarations (`pub mod foo;`) should create local dependencies? **Yes** — they register the file as part of the module tree but don't create inter-layer violations. The `pub mod` pattern identifies local file relationships, not import violations. Only `use` statements generate import-based violations.
