# Design: arx v0.30.0 — filter()/map() for DSL

## Technical Approach

Add `ValueKindList` to the `Value` type system, then implement `filter()` and `map()` as pure builtin functions in the existing `builtins` map. A small **predicate evaluator** tokenizes the filter string by spaces and resolves dependency struct fields by name. No parser/grammar changes — both builtins accept string arguments parsed inline at eval time.

## Architecture Decisions

### Decision: ValueList as a new ValueKind

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Reuse ValueDeps | Correct semantics but deps are not strings | ❌ |
| New ValueKindList | +5 lines, clear semantics, can extend `String()`/`AsBool()`/`AsInt()` | ✅ |
| Reuse []string in a generic Value | No type safety, breaks existing kind-switches | ❌ |

**Rationale**: A distinct kind is the cheapest path. `count(map(...))` works because `AsInt()` returns `len(Values)` for lists.

### Decision: Predicate evaluator as space-split, not tokenizer reuse

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Reuse existing tokenizer + parser | Overkill for `"field op value"` — no parens, no chaining | ❌ |
| Simple `strings.Split` + switch | ~20 lines, handles all cases, clear error messages | ✅ |

**Rationale**: Predicate grammar is intentionally limited to `field op value` (3 tokens, space-separated). The full DSL tokenizer would parse `ResolvedLayer` as an identifier and `==` as TokenEQ, then a string as another identifier, but there's no AST to build — just evaluate once per dep.

### Decision: Field resolution via switch, not reflection

**Choice**: Hard-coded switch on field name strings (`"SourceFile"`, `"SourceLine"`, etc.)
**Alternatives**: `reflect` field lookup — fragile under rename, slower
**Rationale**: 4 fields only. Tests catch rename drift. No overhead, no surprises.

## Data Flow

```
Parse("filter(deps(domain, infra), \"ResolvedLayer == db\")")
  → FuncCallExpr{Name: "filter", Args: [FuncCallExpr{deps...}, StringLiteral{"ResolvedLayer == db"}]}
  → Eval() → builtinFilter()
       ├─ eval args → deps evalúa → ValueDeps
       │                str lit → StringLiteral (sin evaluar)
       ├─ predicate eval: tokenize "ResolvedLayer == db" → {field:"ResolvedLayer", op:"==", value:"db"}
       └─ iterate ValueDeps, match dep.ResolvedLayer == "db"
            → new ValueDeps with matching subset

Parse("map(deps(domain, infra), \"SourceFile\")")
  → builtinMap()
       └─ iterate ValueDeps, collect dep.SourceFile
            → ValueList{Values: ["domain/a.go", "domain/b.go", ...]}
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/expr.go` | Modify | Add `ValueKindList`, `Strings []string` field, update `AsBool()`/`AsInt()`/`String()`, add `builtinFilter`, `builtinMap`, predicate evaluator, update `count()` for lists |
| `internal/domain/expr_test.go` | Modify | Table-driven tests for filter/map/ValueList/predicate edge cases |
| `openspec/changes/arx-v0.30.0-filter-map/specs/custom-dsl/spec.md` | Create | Delta spec with filter/map requirements and scenarios |

## Interfaces / Contracts

### ValueList type

```go
const ValueKindList ValueKind = iota + 3 // after ValueDeps

type Value struct {
    Kind ValueKind
    Int  int
    Bool bool
    Deps []Dependency
    List []string   // new field for ValueKindList
}
```

### filter() — always returns ValueDeps

```go
func builtinFilter(args []Expr, ctx EvalContext) (Value, error)
// Arg 1: must eval to ValueDeps
// Arg 2: must be StringLiteral with predicate
// Returns: Value{Kind: ValueDeps, Deps: filtered}
```

### map() — always returns ValueList

```go
func builtinMap(args []Expr, ctx EvalContext) (Value, error)
// Arg 1: must eval to ValueDeps
// Arg 2: must be StringLiteral with field name
// Returns: Value{Kind: ValueKindList, List: extracted}
```

### Predicate grammar

```
predicate = field " " op " " value
field     = "SourceFile" | "SourceLine" | "ImportPath" | "ResolvedLayer"
op        = "==" | "!=" | ">" | "<" | ">=" | "<="
value     = string (for ==/!=) or number (for numeric ops)
```

- String fields (`SourceFile`, `ImportPath`, `ResolvedLayer`): only `==`/`!=`
- Numeric fields (`SourceLine`): all six operators
- Invalid field name → error at eval, not parse

### ValueList behaviors

| Operation | Behavior |
|-----------|----------|
| `AsBool()` | `len(v.List) > 0` |
| `AsInt()` | `len(v.List)` |
| `compareValues(List, ==, List)` | Compare as int (len == len) |
| `compareValues(List, ==, Int)` | Compare as int |
| `!=`, `>`, `<`, etc. | Pass through int comparison (always behaves as len) |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | ValueList AsBool/AsInt/String | Direct Value construction |
| Unit | `filter(deps, ...)` string equality | Table-driven with Dependency fixtures |
| Unit | `filter(deps, "SourceLine > 10")` numeric | Table-driven with int boundary cases |
| Unit | `map(deps, "ImportPath")` | Produces ValueList with correct strings |
| Unit | `count(map(...))` | Chain count(map(...)) — returns len |
| Unit | Empty filter/map | Returns empty ValueDeps / empty ValueList |
| Unit | Invalid field name | Error returned, not panic |
| Unit | Invalid op for string field (e.g. `SourceFile > 5`) | Error returned |
| Unit | Predicate parse errors (bad format) | Error returned |
| Unit | Type errors (filter(int, ...), map(int, ...)) | Error returned |
| Integration | Full expression with filter/map | Parse + eval actual expressions |

## Migration / Rollout

No migration required. New builtins are additive — existing configs are unaffected. If a user writes a user function named `filter` or `map`, validation rejects it as a builtin shadow (existing `IsBuiltinName` mechanism).

## Open Questions

None.
