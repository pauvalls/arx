# Tasks: arx v0.30.0 — filter()/map() for DSL

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 280–360 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

---

## Phase 1: ValueList type

- [x] 1.1 Add `ValueKindList ValueKind = iota + 3` constant after `ValueDeps` in `internal/domain/expr.go`
- [x] 1.2 Add `List []string` field to `Value` struct
- [x] 1.3 Update `AsBool()` — return `len(v.List) > 0` for `ValueKindList`
- [x] 1.4 Update `AsInt()` — return `len(v.List)` for `ValueKindList`
- [x] 1.5 Update `String()` — return `list[N]` format for `ValueKindList`
- [x] 1.6 Update `compareValues()` — `ValueKindList == ValueKindList` compares `len`, `!=` same; `==`/`!=` against `ValueInt` compares len; other ops pass through via `AsInt()`
- [x] 1.7 Write tests: create ValueList, AsBool on empty/non-empty, AsInt returns len, String format, count() on ValueList returns len

## Phase 2: Predicate evaluator

- [x] 2.1 Implement `evalPredicate(dep Dependency, predicate string) (bool, error)` — tokenizes by space into `[field, op, value]`, validates len == 3, delegates to field switch
- [x] 2.2 Field resolution switch: `"SourceFile"` → dep.SourceFile (==/!= only), `"ImportPath"` → dep.ImportPath (==/!= only), `"ResolvedLayer"` → dep.ResolvedLayer (==/!= only), `"SourceLine"` → dep.SourceLine (all 6 ops), default → error
- [x] 2.3 String op guard: return error if op is not `==` or `!=` for string fields
- [x] 2.4 Write tests: all field+op combos (6 ops × 4 fields), error on unknown field, error on invalid predicate format (< 3 tokens, > 3 tokens), error on string field with numeric op

## Phase 3: filter() builtin

- [ ] 3.1 Implement `builtinFilter` — exactly 2 args, first evals to `ValueDeps`, second is `StringLiteral` with predicate
- [ ] 3.2 Iterate `v.Deps`, call `evalPredicate` per dep, collect matches into `[]Dependency`
- [ ] 3.3 Return `Value{Kind: ValueDeps, Deps: matched}`
- [ ] 3.4 Write tests: filter by SourceFile (==, !=), by ResolvedLayer, by SourceLine (>, <, >=, <=, ==, !=), empty result, invalid predicate error, wrong arg types (first not deps, wrong arity)

## Phase 4: map() builtin

- [ ] 4.1 Implement `builtinMap` — exactly 2 args, first evals to `ValueDeps`, second is `StringLiteral` with field name
- [ ] 4.2 Iterate `v.Deps`, extract field by name via switch (same fields as predicate), collect strings into `[]string`
- [ ] 4.3 Return `Value{Kind: ValueKindList, List: extracted}`
- [ ] 4.4 Write tests: map SourceFile, map ResolvedLayer, map ImportPath, map SourceLine (strconv.Itoa), invalid field name error, wrong arg types (first not deps, wrong arity)

## Phase 5: Cleanup

- [ ] 5.1 Register `filter` and `map` in the `builtins` map
- [ ] 5.2 Run `go test ./internal/domain/...` — verify all new tests pass and no existing tests broken
- [ ] 5.3 Run `go build ./cmd/arx` — verify compilation
