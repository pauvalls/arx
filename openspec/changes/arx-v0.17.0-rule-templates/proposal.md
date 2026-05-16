# Proposal: Arx v0.17.0 — Rule Templates

## Intent

Current rules (Cannot/Must/Can/MustNotCircular) are too rigid for real-world architecture patterns. Users need to express constraints like "max 3 dependencies between layers" or "no domain types leaked into infrastructure" without writing code. Rule templates provide a zero-code, parameterized way to define complex architectural rules.

## Scope

### In Scope
- Template evaluation engine (`internal/domain/template.go`)
- 3 initial templates: `max-deps`, `no-leak`, `layer-balance`
- Config integration: parse `template` + `params` fields in rule YAML
- Backward compatibility: rules without `template` work exactly as before
- Unit + integration tests for templates and config parsing
- Documentation: "Rule Templates" section with examples

### Out of Scope
- Custom/user-defined template functions (DSL)
- Template composition or chaining
- More than 3 initial templates
- Performance optimization beyond O(N) layer scanning

## Capabilities

### New Capabilities
- `rule-templates`: Parameterized rule templates evaluated at config load time. Each template is a function `func(params, deps, layers) []Violation`. Users reference templates via `template` + `params` fields in rule YAML.

### Modified Capabilities
- None — existing rule evaluation is untouched. Template rules are resolved into standard `Rule` structs or evaluated alongside them.

## Approach

**Approach C — Predefined Rule Templates** (from exploration). Each template is a Go function registered in a map. At config load, rules with a `template` field are resolved: params are validated, and the template function is stored. During evaluation, template rules run alongside standard rules, producing `[]Violation`. No external dependencies. ~200-400 lines of Go.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/domain/template.go` | New | Template registry, evaluation engine, 3 template functions |
| `internal/domain/rule.go` | Modified | Add `Template` and `Params` fields to `Rule` struct |
| `internal/domain/config.go` | Modified | Validate template references and params during config validation |
| `internal/application/service.go` | Modified | Evaluate template rules during audit |
| `docs/` | Modified | Add "Rule Templates" section with examples |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Template params schema drift (users pass wrong types) | Medium | Strict param validation at config load with clear error messages |
| Template evaluation order affects results | Low | Document evaluation order; templates run after standard rules |
| Backward compat break if Rule struct changes | Low | New fields are optional with zero-value defaults; existing rules unaffected |

## Rollback Plan

Revert the change — template rules are purely additive. Any config using `template` fields will fail validation on the previous version, which is a safe failure mode (explicit error, not silent misbehavior). No data migration or state cleanup needed.

## Dependencies

- None — pure Go implementation, no external libraries.

## Success Criteria

- [ ] Users can define rules with `template` + `params` in YAML config
- [ ] 3 templates (`max-deps`, `no-leak`, `layer-balance`) produce correct violations
- [ ] Existing rules without `template` work identically (backward compat verified by tests)
- [ ] Invalid template names or params produce clear validation errors at config load
- [ ] Documentation shows working examples for each template
