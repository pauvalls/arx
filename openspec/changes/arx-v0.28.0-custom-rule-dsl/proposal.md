# Proposal: Custom Rule DSL (Extended)

## Intent

arx has 8 builtins but no composability or multi-condition checks. This change gives arx a practical mini-DSL — multi-line checks, `all()`/`any()` aggregators, and user-defined functions — with no type system changes.

## Scope

**In Scope**: (1) `check` accepts `string | []string` — list ANDs expressions. (2) `all(deps)` / `any(deps)` boolean aggregators. (3) `functions:` in arx.yaml — expression-based user functions.

**Out of Scope**: `filter()`/`map()` (needs lambdas), parameterized user funcs, rule cross-refs (`violations()` + user funcs cover this).

## Capabilities

**New**: `custom-dsl` — multiline checks, all/any builtins, user-defined functions.

**Modified**: None (no existing specs).

## Approach

1. **Check list**: New `CheckExpr` type with custom `UnmarshalYAML(string | []string)`. List → `BinaryExpr` AND tree. Single string backward compat.

2. **all()/any()**: Two new builtins in `builtins` map. Take `ValueDeps`, iterate `[]Dependency`, return `ValueBool`. Zero new types or grammar.

3. **User functions**: `Config.Functions map[string]string`. At load: parse each to `Expr`, validate (parse, no shadowing, topological sort + cycle detection). Register in `EvalContext`. `resolveFunction()` checks user funcs before builtins.

## Affected Areas

| Area | Impact |
|------|--------|
| `expr.go` | all/any builtins, user func registry, CheckExpr |
| `config.go` | Functions field, DAG+shadow validation |
| `rule.go` | CheckExpr replaces string |
| `audit.go` | Inject user funcs into EvalContext |
| `arx-schema.json` | functions schema, check as union |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| String→CheckExpr breaks YAML | Medium | Custom UnmarshalYAML, backward compat |
| Circular func refs | Low | Topological sort at load |
| User func shadows builtin | Low | Validate against builtins map |
| AND semantics surprise | Low | Document in schema+changelog |

## Rollback Plan

Per-feature atomic revert: (1) revert CheckExpr→string, (2) remove 4 lines from builtins, (3) revert Config.Functions + resolveFunction + EvalContext. Can roll back independently. Recommend single v0.28.0 tag revert if critical.

## Dependencies

None external. Internal order: Feature 1 then 3 (multiline ergonomics). Feature 2 independent.

## Success Criteria

- [ ] `check: [e1, e2]` evaluates both ANDed
- [ ] `all(deps(a,b))` true only when ALL deps match
- [ ] `any(deps(a,b))` true when ANY dep matches
- [ ] Valid user func registers and evaluates
- [ ] Circular user func refs rejected at config load
- [ ] Builtin-shadowing user func rejected at config load
- [ ] Single-string `check:` fully backward compat
- [ ] All existing tests pass unmodified
