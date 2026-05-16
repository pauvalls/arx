# Design: Arx v0.12.0 Release Automation + Config Quality

## Technical Approach

This release adds production-ready distribution (GoReleaser + Homebrew), CI tolerance controls (`max_violations` threshold), granular rule path exclusion (`exclude` patterns), and quality infrastructure (benchmarks + fuzz tests). Implementation follows existing hexagonal architecture patterns.

## Architecture Decisions

### Decision 1: MaxViolations Threshold Implementation

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Config-level threshold | Simple, centralized control | ✅ Chosen |
| Per-rule threshold | Granular but complex | Rejected |
| CLI flag only | No persistence | Rejected |

**Rationale**: Config-level threshold aligns with existing config-driven architecture. Allows teams to set tolerance once, version with config.

### Decision 2: Per-Rule Exclude Patterns

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Glob patterns on Rule struct | Reuses existing glob utility | ✅ Chosen |
| Separate exclude config section | More flexible but scattered | Rejected |
| Regex patterns | Powerful but error-prone | Rejected |

**Rationale**: Glob patterns match existing layer pattern syntax. Adding `Exclude []string` to Rule struct keeps exclusion logic co-located with rule definition.

### Decision 3: Release Automation Stack

| Option | Tradeoff | Decision |
|--------|----------|----------|
| GoReleaser + GitHub Actions | Industry standard, Homebrew support | ✅ Chosen |
| Manual release scripts | Full control but error-prone | Rejected |
| CI-only builds | No Homebrew tap | Rejected |

**Rationale**: GoReleaser provides cross-platform builds, checksums, signatures, and Homebrew formula automation out-of-box.

### Decision 4: Quality Infrastructure Scope

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Benchmarks on hot paths | Performance baselines | ✅ Chosen |
| Fuzz tests on parsers | Catch edge cases | ✅ Chosen |
| Full coverage first | Time-intensive | Deferred |

**Rationale**: Focus on performance-critical paths (coupling matrix, rule evaluation, parsers) and input validation (config, parsers).

## Data Flow

### MaxViolations Exit Code Flow

```
cmd/arx/check.go          internal/infrastructure/output/terminal.go
     │                              │
     │  EvaluateRules()             │
     ├──────────────────────────────►
     │                              │  Count non-overridden
     │                              │  violations
     │  ExitCode(violations)        │
     ◄──────────────────────────────┤
     │                              │  Return 0 if count <= max_violations
     │                              │  Return 1 otherwise
     │
  os.Exit()
```

### Per-Rule Exclude Flow

```
arx.yaml                  internal/domain/rule.go     internal/domain/audit.go
    │                            │                           │
    │  rules:                    │                           │
    │  - id: "no-legacy"         │  Exclude: ["**/legacy/*"] │
    │    exclude:                │                           │
    │    - "**/legacy/*"         │                           │
    │                            │                           │
    ├────────────────────────────┤                           │
    │                            │                           │
    │                            │  IsExcludedFor(filePath)   │
    │                            ├──────────────────────────►│
    │                            │                           │
    │                            │                           │ Skip if excluded
    │                            │                           │
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/config.go` | Modify | Add `MaxViolations int` field + `ViolationThreshold()` method |
| `internal/domain/rule.go` | Modify | Add `Exclude []string` field + `IsExcludedFor()` method |
| `internal/domain/audit.go` | Modify | Check `IsExcludedFor()` before creating violations |
| `internal/infrastructure/config/yaml.go` | Modify | Validate exclude patterns at config load |
| `cmd/arx/check.go` | Modify | Pass config to exit code logic, include `max_violations` in JSON |
| `internal/infrastructure/output/json.go` | Modify | Add `MaxViolations` field to `JSONOutput` struct |
| `internal/infrastructure/output/terminal.go` | Modify | Modify `ExitCode()` to accept config threshold |
| `internal/infrastructure/detector/java/parser_bench_test.go` | Create | Benchmark Java import extraction |
| `internal/domain/coupling_bench_test.go` | Create | Benchmark coupling matrix calculation |
| `internal/domain/audit_bench_test.go` | Create | Benchmark rule evaluation |
| `internal/infrastructure/config/config_fuzz_test.go` | Create | Fuzz config YAML parsing |
| `internal/infrastructure/detector/java/parser_fuzz_test.go` | Create | Fuzz Java parser input |
| `internal/infrastructure/detector/csharp/parser_fuzz_test.go` | Create | Fuzz C# parser input |
| `.goreleaser.yaml` | Create | GoReleaser configuration |
| `.github/workflows/release.yml` | Create | GitHub Actions release workflow |

## Interfaces / Contracts

### Config Changes

```go
// internal/domain/config.go
type Config struct {
    Version        string
    Layers         []Layer
    Rules          []Rule
    // ... existing fields ...
    MaxViolations  int `yaml:"max_violations,omitempty" json:"max_violations,omitempty"`
}

// ViolationThreshold returns the max_violations threshold (0 = unlimited).
func (c *Config) ViolationThreshold() int {
    return c.MaxViolations
}
```

### Rule Changes

```go
// internal/domain/rule.go
type Rule struct {
    ID              string
    From            string
    To              []string
    Type            RuleType
    Severity        Severity
    // ... existing fields ...
    Exclude         []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
    compiledExclude []*regexp.Regexp `json:"-" yaml:"-"`
}

// CompileExcludePatterns compiles exclude glob patterns to regex.
func (r *Rule) CompileExcludePatterns() error

// IsExcludedFor checks if filePath matches any exclude pattern.
func (r *Rule) IsExcludedFor(filePath string) bool
```

### Exit Code Logic

```go
// internal/infrastructure/output/terminal.go
func ExitCode(violations []domain.Violation, maxViolations int) int {
    if len(violations) == 0 {
        return 0
    }
    
    nonOverridden := 0
    for _, v := range violations {
        if !v.Overridden {
            nonOverridden++
        }
    }
    
    if maxViolations > 0 && nonOverridden <= maxViolations {
        return 0
    }
    
    return 1
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `MaxViolations` threshold logic | Table-driven tests with edge cases (0, 1, threshold, threshold+1) |
| Unit | `IsExcludedFor()` glob matching | Test exact match, prefix, glob patterns (`*`, `**`) |
| Unit | Config validation for invalid patterns | Reject invalid glob patterns at load time |
| Benchmark | Coupling matrix calculation | `BenchmarkCouplingMatrix` with varying layer counts |
| Benchmark | Rule evaluation | `BenchmarkRuleEvaluation` with varying rule/dependency counts |
| Benchmark | Java parser extraction | `BenchmarkJavaExtraction` on sample files |
| Fuzz | Config YAML parsing | `FuzzConfigParse` for malformed YAML |
| Fuzz | Java parser input | `FuzzJavaParse` for malformed source files |
| Fuzz | C# parser input | `FuzzCSharpParse` for malformed source files |
| Integration | End-to-end exit code | Run `arx check` with various violation counts |

## Migration / Rollout

**No migration required.** All changes are backward-compatible:
- `max_violations` defaults to 0 (unlimited, current behavior)
- `exclude` defaults to empty slice (no exclusions, current behavior)
- Release automation is additive (doesn't affect existing workflows)

## Open Questions

- [ ] Should `max_violations` apply per-severity (e.g., separate thresholds for errors vs warnings)?
- [ ] Should exclude patterns support negation (e.g., `!internal/legacy/*` to re-include)?
- [ ] Homebrew tap: use `pauvalls/homebrew-arx` or contribute to `homebrew-core`?
