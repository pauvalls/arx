# Proposal: arx v0.30.0 — filter()/map() for DSL

## Intent

`deps()` returns an opaque `ValueDeps` — only countable or checkable via `all()`/`any()`. Rule authors need to narrow down deps by field values and extract specific data. `filter()` keeps only matching deps; `map()` extracts field values. Enables targeted violation analysis without leaving the expression language.

## Scope

**In**: `filter(deps, "field op value")`, `map(deps, "field")`, `ValueList` type, predicate evaluator (string-based, reuses comparison ops).
**Out**: Lambdas (deferred), filter/map on non-deps, nested chains, multi-field predicates, dot-notation access.

## Capabilities

- **New**: `dsl-filter-map` — filter/map builtins with string predicates and field accessors.
- **Modified**: `custom-dsl` — delta spec adding filter/map requirements and ValueList scenarios.

## Approach

Approach 3 from v0.28.0 exploration (String-based Predicates — already recommended for filter/map there).

1. **`filter(deps, "field op value")`**: Internal predicate evaluator tokenizes `{field, op, value}`, applies op per-Dependency. Returns filtered `ValueDeps`.
2. **`map(deps, "field")`**: Iterates deps, extracts field by name into `[]string`. Returns `ValueList`.
3. **`ValueList`** (`ValueKind`): wraps `[]string`. `AsBool()` = len>0, `AsInt()` = len, `==`/`!=` against string or ValueList.
4. **No parser changes**: Pure builtins in `builtins` map. Predicate parsed inline — no grammar/token/AST changes.

## Affected Areas

| Area | Impact |
|------|--------|
| `expr.go` | Add ValueList kind/field, filter/map builtins, predicate evaluator |
| `expr_test.go` | Tests for filter/map, ValueList, predicate edge cases |
| `specs/custom-dsl/spec.md` | Delta for filter/map requirements + scenarios |

## Risks

| Risk | Mitigation |
|------|------------|
| String predicates — no compile-time validation | Fail at parse with clear error; validate field name + op |
| Go struct field rename drift | Tests catch; 4 fields only |
| ValueList affects comparison logic | `==`/`!=` only; numeric ops error at eval |
| Predicate grammar diverges from DSL | Share same op tokens, token-style parsing |

## Rollback Plan

Remove 2 builtins, delete predicate evaluator (~40 lines), drop ValueList (~15 lines). ~55 lines total, single-commit revert on v0.30.0 tag.

## Dependencies

None. Requires `custom-dsl` active (v0.28.0+). No ordering constraints.

## Success Criteria

- [ ] `filter(deps(a,b), "ResolvedLayer == b")` returns matching deps only
- [ ] `filter(deps(a,b), "SourceLine > 10")` numeric field comparison
- [ ] `map(deps(a,b), "ImportPath")` returns ValueList of import paths
- [ ] `count(map(...))` returns list length via AsInt()
- [ ] Empty filter returns falsy ValueDeps (len 0)
- [ ] Invalid field name → eval error
- [ ] ValueList `==` against string works (single element)
- [ ] All existing v0.28.x tests pass
