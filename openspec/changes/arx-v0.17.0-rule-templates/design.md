# Design: Arx v0.17.0 — Rule Templates

## Technical Approach

Add a `TemplateRegistry` in `internal/domain/template.go` with a `TemplateFunc` type signature. Each template is a pure function `(params, deps, layers) → []Violation`. Rules gain optional `template` and `params` YAML fields. `EvaluateRules` in `audit.go` runs template-based rules alongside existing from/to rules (AND logic, violations merged). Config validation checks template names and required params at load time. Zero external dependencies.

## Architecture Decisions

| Decision | Option | Tradeoff | Decision |
|----------|--------|----------|----------|
| Template resolution | Pre-registered Go functions | No DSL flexibility, but zero deps and type-safe | **Chosen**: registry map |
| Evaluation timing | During `EvaluateRules` alongside standard rules | Simpler than pre-resolving into Rule structs | **Chosen**: inline evaluation |
| Backward compat | New fields are `omitempty`, zero-value skips template path | Existing configs work unchanged | **Chosen**: additive fields |
| Param validation | At config load (`Config.Validate`) | Fails fast with clear errors before audit runs | **Chosen**: early validation |
| Violation ID strategy | Shared counter across standard + template rules | Sequential IDs, no collision risk | **Chosen**: single counter in `EvaluateRules` |

## Data Flow

```
arx.yaml
  │
  ▼
Config.Validate() ──→ check template exists in registry
  │                   validate required params present
  ▼
EvaluateRules(deps, rules, layers)
  │
  ├── Standard rules: for each dep, rule.Violates() → violations
  │
  └── Template rules: for each rule with Template field:
        fn := TemplateRegistry[rule.Template]
        violations = append(violations, fn(rule.Params, deps, layers)...)
  │
  ▼
[]Violation (merged, sequential IDs)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/template.go` | Create | `TemplateFunc` type, `TemplateRegistry` var, 3 template functions, `ValidateTemplateParams` helper |
| `internal/domain/rule.go` | Modify | Add `Template string` and `Params map[string]any` fields; extend `Validate()` to check template refs |
| `internal/domain/config.go` | Modify | Extend `Validate()` to validate template params against known schemas |
| `internal/domain/audit.go` | Modify | In `EvaluateRules`, after standard rule loop, iterate template rules and merge violations |
| `internal/domain/template_test.go` | Create | Unit tests for each template function |
| `internal/domain/rule_test.go` | Modify | Add test cases for template rule validation |
| `internal/domain/config_test.go` | Modify | Add test cases for config with template rules |
| `internal/domain/audit_test.go` | Modify | Add integration test: `EvaluateRules` with template rules producing violations alongside standard rules |

## Interfaces / Contracts

### TemplateFunc signature

```go
// TemplateFunc evaluates a rule template and returns violations.
// params: validated YAML params map for this template
// deps: all detected dependencies
// layers: all configured layers
type TemplateFunc func(params map[string]any, deps []Dependency, layers []Layer) []Violation
```

### TemplateRegistry

```go
var TemplateRegistry = map[string]TemplateFunc{
    "max-deps":      TemplateMaxDeps,
    "no-leak":       TemplateNoLeak,
    "layer-balance": TemplateLayerBalance,
}
```

### Template param schemas (validated at config load)

```go
// max-deps:   from (string), to ([]string), max (int)
// no-leak:    layer (string), forbidden ([]string)
// layer-balance: min (int), max (int)
```

### Rule struct additions

```go
type Rule struct {
    // ... existing fields ...
    Template string         `yaml:"template,omitempty" json:"template,omitempty"`
    Params   map[string]any `yaml:"params,omitempty" json:"params,omitempty"`
}
```

### Validate extension

```go
func (r *Rule) Validate() error {
    // ... existing validation ...
    if r.Template != "" {
        if _, ok := TemplateRegistry[r.Template]; !ok {
            return fmt.Errorf("rule %q: unknown template %q", r.ID, r.Template)
        }
        if err := ValidateTemplateParams(r.Template, r.Params); err != nil {
            return fmt.Errorf("rule %q: %w", r.ID, err)
        }
    }
    // ... rest ...
}
```

### YAML example

```yaml
rules:
  - id: domain-max-deps
    template: max-deps
    severity: error
    params:
      from: domain
      to: [infrastructure]
      max: 3
    explanation: "Domain should not have more than 3 infrastructure dependencies"

  - id: no-domain-leak
    template: no-leak
    severity: error
    params:
      layer: domain
      forbidden: [infrastructure, application]
    explanation: "Domain types must not leak into other layers"
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `TemplateMaxDeps` — counts deps, returns violation when > max | Table-driven tests with synthetic deps/layers |
| Unit | `TemplateNoLeak` — detects forbidden layer imports | Table-driven tests with path-prefix matching |
| Unit | `TemplateLayerBalance` — counts in/out deps per layer | Table-driven tests with boundary min/max |
| Unit | `ValidateTemplateParams` — missing/extra/wrong-type params | Error message assertions |
| Unit | `Rule.Validate()` with template field — valid + invalid cases | Extend existing rule_test.go table |
| Integration | `EvaluateRules` with mixed standard + template rules | Verify both violation types merge, IDs sequential |
| Integration | Config round-trip: YAML parse → Validate → Evaluate | Full pipeline with arx.yaml containing templates |

## Migration / Rollout

No migration required. Template fields are additive with `omitempty` — existing configs parse and validate identically. Configs using `template` will fail on older versions (safe failure: explicit validation error, not silent misbehavior).

## Open Questions

- [ ] Should template violations use a distinct ID prefix (e.g., `T-01`) vs standard `D-01`? Currently sharing the sequential counter — keeps IDs simple but loses origin info.
- [ ] Should `Explanation` on template rules be interpolated with param values (e.g., "max 3 deps, found 5") or kept static? Proposal shows static explanation.
