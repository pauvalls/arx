# Design: arx v0.28.0 — Custom Rule DSL (Extended)

## Technical Approach

Three incremental layers: (1) `CheckExpr` type wrapping `string | []string` with custom YAML unmarshal → AND tree, (2) `all(deps)`/`any(deps)` boolean aggregators as new builtins, (3) `Config.Functions map[string]string` → parse → DAG validation → inject into `EvalContext`. All reuse existing parser/AST — zero grammar changes.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Multi-check type | `CheckExpr` struct with custom `UnmarshalYAML` | `interface{}` on `Rule.Check` | Type safety; `interface{}` leaks YAML marshaling concerns into the model. `CheckExpr` encapsulates the union cleanly. |
| AND-tree construction | Recursive `BinaryExpr{Op: TokenAnd, Left, Right}` | `[]Expr` with loop evaluator | Reuses existing `BinaryExpr` short-circuit eval. No new AST node needed. |
| `all()`/`any()` impl | Pure builtins in `builtins` map | Lambda/predicate approach | Zero grammar changes. Works with existing `ValueDeps`. ~50 lines. Filter/map deferred. |
| User func resolution | `FuncCallExpr.Eval` checks `EvalContext.userFunctions` before `builtins` | Separate registry | Minimal eval path change. Shadowing is a feature with validation guard. |
| DAG validation | Kahn's algorithm at `Config.Validate()` | Runtime cycle detection | Fail fast at load. No runtime overhead. |

## Data Flow

```
arx.yaml load → Config.Validate():
  ├── Parse Functions values (map[string]string → map[string]Expr)
  ├── Build adjacency list from func bodies (tokenize → extract ident-then-lparen calls)
  ├── Kahn topological sort → detect cycles
  ├── Check no func name shadows builtins
  └── Store compiled Exprs in EvalContext.userFunctions

Rule evaluation at runtime:
  FuncCallExpr.Eval(ctx):
    1. Check ctx.userFunctions[name] — if found, eval that Expr
    2. Fall back to builtins[name]
    3. Error if neither exists
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/expr.go` | Modify | Add `CheckExpr` type, `all()`/`any()` builtins, user func resolution in `FuncCallExpr.Eval`, `userFunctions map[string]Expr` in `EvalContext` |
| `internal/domain/rule.go` | Modify | Replace `Check string` → `Check CheckExpr`; update `compileCheckExpression`, `CheckExpressionIsStandalone` |
| `internal/domain/config.go` | Modify | Add `Functions map[string]string` field; DAG validation + shadow check in `Validate()` |
| `internal/domain/audit.go` | Modify | Build `EvalContext` with user func map when evaluating expression rules |
| `arx-schema.json` | Modify | `functions` as `map[string]string`; `check` as `{"oneOf": [string, array]}` |

## Interfaces / Contracts

```go
// CheckExpr wraps a check expression that can be a single string or a list.
type CheckExpr struct {
    raw   string
    exprs []Expr // parsed expressions (multiple → AND-tree)
}

// UnmarshalYAML accepts string (single) or []interface{} (list → ANDed).
func (c *CheckExpr) UnmarshalYAML(v func(interface{}) error) error

// Config.Functions store user-defined expression functions.
type Config struct {
    // ... existing fields ...
    Functions map[string]string `yaml:"functions,omitempty"`
}

// EvalContext gains userFunctions for user func resolution.
type EvalContext struct {
    Deps          []Dependency
    Layers        []Layer
    Violations    []Violation
    LayerFiles    map[string][]string
    userFunctions map[string]Expr // NEW
}

// FuncCallExpr.Eval updated to resolve user funcs first:
func (e *FuncCallExpr) Eval(ctx EvalContext) (Value, error) {
    if fn, ok := ctx.userFunctions[e.Name]; ok {
        return fn.Eval(ctx)
    }
    if fn, ok := builtins[e.Name]; ok {
        return fn(e.Args, ctx)
    }
    return Value{}, fmt.Errorf("unknown function %q", e.Name)
}
```

## Implementation Plan

| Phase | What | Depends on | Effort |
|-------|------|------------|--------|
| 1 | `CheckExpr` type + multi-line support (list → AND tree) | Nothing | ~4h |
| 2 | `all()`/`any()` builtins | Nothing (parallelizable) | ~2h |
| 3 | User-defined functions + DAG validation | Phase 1 (for multi-line func bodies) | ~1-2d |
| 4 | Schema update + docs | Phases 1-3 | ~2h |

Phase 2 is independent and can be done in parallel with Phase 1.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `CheckExpr.UnmarshalYAML` with string, `[]string`, invalid types | Table-driven, assert compiled AND tree |
| Unit | `all()`/`any()` with `ValueDeps`, empty deps, wrong arg count | Table-driven, assert bool result |
| Unit | User func parsing, circular rejection, builtin shadowing | Assert `Config.Validate()` errors |
| Unit | User func evaluation in `FuncCallExpr.Eval` | Assert correct value returned |
| Integration | User func used in rule `check` expression | Full `EvaluateRules` with `EvalContext.userFunctions` set |
| Regression | All existing tests pass unmodified | `go test ./...` |

## Effort Estimate

~3-4 days total. ~600-800 lines added. Delivery: exception-ok (single PR under 400 lines per phase in practice; if combined exceeds, chain by phase).

## Rollout

No migration. Per-feature atomic revert: (1) revert `CheckExpr→string`, (2) remove 4 lines from builtins map, (3) revert `Config.Functions` + `EvalContext.userFunctions`.
