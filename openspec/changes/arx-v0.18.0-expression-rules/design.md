# Design: Expression Rules, Concurrent Detectors, and Severity Mapping

## Technical Approach

Three orthogonal features delivered in one change:

1. **Expression Rules**: A recursive-descent parser/evaluator for boolean expressions that operate on the dependency graph. Rules gain an optional `Check` field; when present, the expression evaluator replaces the standard from/to violation logic.
2. **Concurrent Detectors**: Parallelize `RunDetectors` using `errgroup.Group`. One goroutine per applicable detector; first error cancels the rest and fails fast.
3. **Severity Mapping**: A config-level `severity_mapping` dictionary that translates raw severity strings into canonical ones after YAML load.

All changes are additive and backward-compatible.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Parser style | Hand-written recursive descent | ANTLR, yacc, or external parser lib | Zero new runtime dependencies; the grammar is tiny (~10 productions). Go stdlib only. |
| Value representation | Tagged union (`Value{kind, i, b, deps}`) | Generics / separate BoolExpr/IntExpr interfaces | A single `Expr` interface keeps the AST flat. The tag switch is ~5 cases. |
| Check field precedence | `Check` wins over from/to/template when non-empty | Merge logic or AND/OR semantics | Simpler mental model: one rule, one evaluation path. |
| Concurrent collection | `errgroup.Group` + `sync.Mutex` on result slice | Channels per detector | errgroup gives us context cancellation for free; mutex append is race-safe and readable. |
| Severity mapping timing | Post-unmarshal in `Config.Validate()` | Custom `Severity.UnmarshalYAML` | Mapping lives on `Config`, so we need the full config before we can rewrite severities. |

## Data Flow

```
Config Load
    │
    ├──► YAML Unmarshal ──► Config.Validate()
    │                           │
    │                           ├──► Apply SeverityMapping
    │                           └──► Validate Check expressions (parse only)
    │
    ▼
RunDetectors (concurrent via errgroup)
    │
    ├──► Detector A (goroutine) ──┐
    ├──► Detector B (goroutine) ──┼──► mutex-protected deps slice
    └──► Detector C (goroutine) ──┘
    │
    ▼
EvaluateRules
    │
    ├──► Standard from/to rules
    ├──► Template rules
    └──► Expression rules ──► Parse (cached) ──► Eval(deps, layers) ──► bool
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/expr.go` | Create | Tokenizer, AST nodes, recursive-descent parser, and evaluator. |
| `internal/domain/expr_test.go` | Create | Unit tests for tokenizer, parser, and evaluator. |
| `internal/domain/rule.go` | Modify | Add `Check string` field to `Rule`. Add Check syntax validation in `Rule.Validate()`. |
| `internal/domain/config.go` | Modify | Add `SeverityMapping map[string]string`. Apply mapping + validate mapped values in `Config.Validate()`. |
| `internal/domain/audit.go` | Modify | Hook expression-rule evaluation into `EvaluateRules` when `rule.Check != ""`. |
| `internal/application/check.go` | Modify | Replace sequential `RunDetectors` loop with `errgroup`-based concurrency. |
| `internal/application/check_test.go` | Modify | Add concurrency tests (detectors with sleep, verify total wall time drops). |
| `go.mod` | Modify | Add `golang.org/x/sync` dependency. |

## Interfaces / Contracts

```go
// EvalContext is the runtime data available to expressions.
type EvalContext struct {
    Deps   []Dependency
    Layers []Layer
}

// Value is the tagged-union result of an expression.
type Value struct {
    Kind  ValueKind // Int, Bool, Deps
    Int   int
    Bool  bool
    Deps  []Dependency
}

// Expr is implemented by every AST node.
type Expr interface {
    Eval(ctx EvalContext) (Value, error)
}

// Parse parses a Check string into an Expr tree.
func Parse(check string) (Expr, error)
```

**Grammar (informal):**
```
expr        = or_expr
or_expr     = and_expr { "||" and_expr }
and_expr    = not_expr { "&&" not_expr }
not_expr    = "!" not_expr | comparison
comparison  = primary { (">" | "<" | ">=" | "<=" | "==" | "!=") primary }
primary     = func_call | number | "(" expr ")"
func_call   = ident "(" [ arg { "," arg } ] ")"
arg         = expr | string_literal
string_literal = ident   // layer names are bare identifiers in this DSL
```

**Built-in functions:**
- `count(expr)` → int
- `deps(fromLayer, toLayer)` → []Dependency
- `layers()` → int
- `has_circular()` → bool

**Rule.Validate()** change: if `Check != ""`, call `Parse(check)` and surface syntax errors.

**Config.Validate()** change: after unmarshaling, iterate `c.Rules`; if `c.SeverityMapping[rule.Severity]` exists, rewrite the severity and validate the mapped value is one of `error`, `warning`, `info`.

**RunDetectors** change:
```go
g, ctx := errgroup.WithContext(ctx)
var mu sync.Mutex
var allDeps []Dependency

for _, d := range detectors {
    d := d // capture
    g.Go(func() error {
        // detect + extract
        mu.Lock()
        allDeps = append(allDeps, deps...)
        mu.Unlock()
        return nil
    })
}
return allDeps, g.Wait()
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Tokenizer, parser error paths, evaluator with mock context | Table-driven tests in `expr_test.go` |
| Unit | `Rule.Validate()` rejects invalid Check syntax | Add cases to `rule_test.go` |
| Unit | `Config.Validate()` applies severity mapping correctly | Add cases to `config_test.go` |
| Integration | Expression rules produce violations in `EvaluateRules` | Add cases to `audit_test.go` |
| Integration | Concurrent detectors aggregate deps correctly and cancel on error | Sleep-based mocks in `check_test.go`, measure elapsed time |

## Migration / Rollout

No migration required. All changes are additive:
- Existing configs without `check` or `severity_mapping` behave identically.
- Existing `RunDetectors` callers see only performance improvement.

## Open Questions

- [ ] Should parsed `Expr` trees be cached on the `Rule` struct (like `compiledPattern`) to avoid re-parsing on every `EvaluateRules` call? **Recommendation: yes**, add `compiledExpr Expr` unexported field.
- [ ] Should expression rules still carry `Severity` and `Explanation` for violation enrichment? **Yes** — reuse existing fields; the expression only decides *whether* the rule is violated.
