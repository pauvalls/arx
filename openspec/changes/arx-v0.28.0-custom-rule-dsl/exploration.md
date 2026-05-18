# Exploration: Custom Rule DSL (Extended)

## Current State

### Expression Engine (`internal/domain/expr.go`)

The engine is a hand-written recursive-descent parser with three layers:

**Tokenizer** (lines 62–162): Scans expression strings into tokens. Supports identifiers (layer names), integers, comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`), logical operators (`&&`, `||`, `!`), parentheses, and commas. Strings are bare identifiers (no quotes). Newlines are treated as whitespace and skipped.

**AST** (lines 166–356): Single `Expr` interface with `Eval(ctx EvalContext) (Value, error)`. `Value` is a tagged union of `ValueInt`, `ValueBool`, `ValueDeps`. AST nodes: `ComparisonExpr`, `BinaryExpr` (AND/OR with short-circuit), `UnaryExpr` (NOT), `FuncCallExpr`, `NumberLiteral`, `StringLiteral`. No collection/array types beyond opaque `ValueDeps`.

**Parser** (lines 543–711): Recursive-descent, precedence: `||` → `&&` → `!` → comparison → primary. No support for lambdas, list literals, or multi-statement expressions.

**Built-in functions** (lines 360–510) — 8 hardcoded in the `builtins` map:
- `count(expr)` → int
- `deps(fromLayer, toLayer)` → []Dependency
- `layers()` → int
- `has_circular()` → bool
- `files(layer)` → int
- `ratio(count, total)` → int
- `violations(ruleID)` → int
- `threshold(value, min, max)` → bool

**No mechanism for user-defined functions.** The `builtins` map is a compile-time constant. No `functions` field exists on `Config`.

**Key limitation — value types only go one way**: `ValueDeps` can be created by `deps()` and consumed by `count()`, but there is no way to inspect individual dependencies, filter them, or map over them. The dependency type itself (`Dependency` with `SourceFile`, `SourceLine`, `ImportPath`, `ResolvedLayer`) exists but is opaque to the expression language.

### How Rules Use Expressions (`internal/domain/rule.go`, `internal/domain/audit.go`)

- `Rule` has a `Check string` field + cached `compiledExpr Expr`
- Validation (**rule.go** lines 176–308): When `Check != ""`, it's parsed and cached. Check rules are **standalone only** — cannot mix with `from`/`to`/`template`/`pattern`.
- Evaluation (**audit.go** lines 89–121): After standard rules, expression rules are evaluated separately against `EvalContext{Deps, Layers, Violations, LayerFiles}`. If `ruleCheckMatches()` returns true, a violation is emitted.
- The expression is evaluated in a **rule-level context** (not per-dependency like traditional rules). This means expression rule violations have empty `File`, `Line`, `SourceLayer`, `TargetLayer`, and `Import` fields.
- **`violations()` builtin**: Already enables rudimentary cross-rule composition (e.g., `violations("domain-no-import-infra") == 0`).

### Templates (`internal/domain/template.go`)

Three built-in templates: `max-deps`, `no-leak`, `layer-balance`. Templates are Go functions registered in `TemplateRegistry`. They have their own param schema validation. Templates are separate from the expression engine — they don't use `Check` at all.

### Rule Evaluation Flow (`internal/domain/audit.go`, `internal/application/audit.go`)

```
Config.Load → Config.Validate() → RunDetectors() → EvaluateRules() → GenerateReport()
                                                    │
                                        ┌───────────┼───────────┐
                                        ▼           ▼           ▼
                                   Standard    Expression   Template
                                   from/to     (Check)      (Template)
                                   rules       rules        rules
```

### Test Patterns

- **`expr_test.go`** (1085 lines): Table-driven tests organized by layer — tokenizer, parser, evaluator. Tests for all builtins, error paths, full expression round-trips, and integration with `EvaluateRules`.
- **`rule_test.go`** (1064 lines): Tests for `Rule.Validate()` including Check expression validation and mixing rejection.
- **`audit_test.go`** (934 lines): Integration tests for `EvaluateRules()` including mixed traditional + expression rule scenarios.
- Tests use standard `go test`, table-driven, `testing.T` only (no external test frameworks).

## Affected Areas

- `internal/domain/expr.go` — Core expression engine. Add new builtins (`all`, `any`, `filter`, `map`), support user-defined function registration, extend value types if needed (e.g., `ValueList` for filter/map results). Add lambda/closure support if filter/map take predicates.
- `internal/domain/expr_test.go` — Tests for all new builtins, lambda parsing, user function resolution, edge cases.
- `internal/domain/rule.go` — Possibly extend `Rule` struct for rule composition (e.g., `Compose` field). No changes needed for user-defined functions (they live on Config).
- `internal/domain/config.go` — Add `Functions map[string]string` field to `Config` for user-definable functions. Validate function expressions at config load time. Register user functions in `EvalContext` or a shared registry.
- `internal/domain/audit.go` — Pass user functions into `EvalContext`. Handle composed rules if using a `compose` field. Build `EvalContext` with user function map.
- `internal/domain/audit_test.go` — Integration tests for user-defined functions and composed rules.
- `internal/domain/template.go` — Potentially extend template system to support user-defined expressions-as-templates.
- `internal/domain/layer.go` — No changes needed.
- `internal/domain/dependency.go` — No changes needed (Dependency struct is already stable).
- `internal/application/audit.go` — No changes (evaluation logic is in domain layer).
- `internal/application/check.go` — No changes needed.
- `arx-schema.json` — Add `functions` JSON schema definition and rule composition field.
- `arx.yaml` — No changes (example only).

## Approaches

### 1. Minimal Builtins Extension — `all()`, `any()` only, no `filter()`/`map()`

Add `all(deps)` and `any(deps)` as simple boolean aggregators over `ValueDeps`. `all(deps(domain, infra))` returns true when EVERY resolved dependency from domain targets infra. `any(...)` returns true when AT LEAST ONE matches. No new types needed. Then `filter()`/`map()` are deferred.

- Pros:
  - Zero new types — works with existing `ValueDeps` and `EvalContext`
  - ~50 lines of code, trivial test additions
  - No grammar changes needed
  - Fully backward compatible
- Cons:
  - `filter()`/`map()` are explicitly requested and this scope excludes them
  - Limited expressive power — no way to filter deps by specific criteria
- Effort: **Low** (~1–2 hours)

### 2. Full Builtins + Lambda Support — `all()`, `any()`, `filter()`, `map()` with anonymous functions

Extend the grammar to support lambda expressions: `fn(ident) expr` or `|ident| expr`. Add `ValueList` type (collection of `Dependency`). Then:
- `all(deps(domain, infra), |d| d.ResolvedLayer == "infra")` → bool
- `any(deps, |d| d.SourceFile matches "*.go")` → bool  
- `filter(deps(domain, infra), |d| d.ImportPath contains "db")` → []Dependency
- `map(deps(domain, infra), |d| d.ResolvedLayer)` → []string

- Pros:
  - Delivers all four requested functions
  - Genuinely powerful — enables arbitrary dependency inspection
  - Extensible for future DSL additions
- Cons:
  - **Major grammar expansion** — lambdas require new token types, AST nodes, and scope management
  - `ValueList` type ripples through the evaluator — comparison, boolean coercion, and `count()` all need updates
  - New parser complexity (lambda binding, closure capture)
  - `EvalContext` needs to carry variable bindings
  - Effort is disproportionate to the value for an audit tool
- Effort: **High** (~3–5 days)

### 3. String-based Predicates for `filter()`/`map()` — Simpler than lambdas

Instead of lambdas, pass string arguments that describe field access: `filter(deps(domain, infra), "ResolvedLayer == infra")`. A mini-evaluator interprets predicates against individual `Dependency` structs. `map(deps(domain, infra), "ResolvedLayer")` extracts field values.

- Pros:
  - No grammar changes for lambda syntax
  - ~200 lines for predicate evaluator + field accessor
  - Still delivers `filter()` and `map()` with practical utility
  - Predicate syntax reuses existing comparison operators
- Cons:
  - String-based predicates are stringly-typed — no compile-time validation
  - Field access is fragile (depends on Go struct field names)
  - Nested expressions not supported in predicates
  - Two separate expression evaluators (main + predicate) increase maintenance
- Effort: **Medium** (~1–2 days)

### 4. User-Defined Functions — Expression-based function registry

Add a `functions` section to `arx.yaml`:
```yaml
functions:
  heavy_to_infra: "count(deps(domain, infra)) > 5"
  is_leaking: "violations(domain-no-import-infra) > 0"
  balanced: "threshold(files(domain), 10, 100)"
```

At config load time, parse and register these in a `map[string]Expr` on `EvalContext`. In `check` expressions, reference them as function calls: `heavy_to_infra() && is_leaking()`.

- Pros:
  - Reuses existing parser and evaluator — no new grammar
  - Solves **both** "user-definable functions" AND "rule composition" (functions can call `violations()`)
  - Functions are composable — one function can call another (if ordered topologically)
  - Very natural extension of the existing pattern
- Cons:
  - Functions are static strings — no parameterization (parameterized functions would need a `params:` array)
  - Ordering matters if functions reference each other (DAG validation required)
  - No recursion guard needed for expression-based functions (expressions can't recurse)
- Effort: **Medium** (~1–2 days)

### 5. Rule Composition via Cross-References — Compose existing rules

Add optional `depends_on: []string` or `compose_and: []string` / `compose_or: []string` to the `Rule` struct. A composed rule evaluates its dependencies and emits a violation when the composition condition is met. A new `CompositionExpr` AST node wraps references to other rules.

```yaml
- id: all-clean
  compose: { and: [R1, R2] }
  severity: error
```

- Pros:
  - Clean YAML-based composition — no expression syntax needed
  - Evaluated after standard rules (like templates)
- Cons:
  - Adds a third evaluation path (standard → expression → composed)
  - Limited expressiveness compared to `violations()` + expression
  - Overlaps with what `violations() && violations()` already does
- Effort: **Medium** (~1 day)

### 6. Multi-line Rule Checks — Parser and YAML support

The tokenizer already skips whitespace (including newlines), so `count(\ndeps(a,b)\n) > 3` already parses. The real need is:
1. YAML multi-line strings: using `|` or `>` in arx.yaml (already works with YAML)
2. When multi-line is used to express multiple conditions, they'd be ANDed automatically — or the expression needs a newline-aware grammar extension

```yaml
check: |
  count(deps(domain, infra)) > 0
  && !has_circular()
```

This already works because the tokenizer skips newlines. However, a **list of expressions** might be cleaner:

```yaml
check:
  - "count(deps(domain, infra)) > 3"
  - "!has_circular()"
```

When `check` is a list, expressions are ANDed together.

- Pros:
  - Minimal code change in `Rule.Validate()` / `compileCheckExpression()` — detect list vs string
  - List-of-expressions is more readable than chained `&&`
  - YAML multi-line strings already work
- Cons:
  - Need to change `Check` field type from `string` to `interface{}` (string or []string) — breaks YAML unmarshaling unless using a custom unmarshaler
  - Need to change `compiledExpr` from `Expr` to `[]Expr` (or an AND-wrapping node)
- Effort: **Low** (~4 hours)

## Recommendation

**Primary: Approach 1 (minimal `all()`/`any()` builtins) + Approach 4 (user-defined functions) + Approach 6 (multi-line rule checks)**

This combination delivers the most value with the least risk:

1. **`all()`/`any()`** as simple boolean aggregators over `ValueDeps`: immediate practical value, trivial implementation, no type system changes. Defer `filter()`/`map()` to a future release — they require either lambda support or string-based predicates, both of which are significant scope expansions.

2. **User-defined functions in `arx.yaml`**: Solves TWO requested features at once — "user-definable functions" AND "rule composition". Functions defined as expression strings are parsed at config load time and injected into `EvalContext`. Cross-rule composition is already possible via `violations()`, but user functions make it ergonomic:
   ```yaml
   functions:
     has_issues: "violations(domain-no-import-infra) > 0"
   rules:
     - id: R1
       check: has_issues()
       severity: warning
   ```

3. **Multi-line rule checks (list form)**: Change `Check` field to accept `string | []string`, compile list as `BinaryExpr` tree with `TokenAnd`, `TokenOr` links. The single-line string form remains fully backward compatible.

**Effort**: Medium (~3-4 days total)

**Implementation order**:
1. Multi-line `check` list support (simplest, enables better UX for everything else)
2. `all()`/`any()` builtins (isolated, no ripple effects)
3. User-defined functions (depends on 1 for multi-line function definitions)

Defer `filter()`/`map()` to a future release with a proper lambda design that doesn't overcomplicate the parser.

## Risks

- **User function DAG**: If functions reference other functions, a topological order is needed during compilation. Circular references must be rejected. This is a new validation concern in `Config.Validate()`.
- **String → `interface{}` change for `Check`**: Changing `Check string` to `Check interface{}` affects YAML unmarshaling. A custom `UnmarshalYAML` on a new type (e.g., `CheckExpr`) is safer than changing the field type directly.
- **Expression evaluation scope creep**: Adding `all()`/`any()` is safe, but if the team later wants `filter()`/`map()`, the lambda design should be intentional from the start rather than retrofitted. Recommend deferring with a design doc note.
- **User functions shadowing builtins**: A user-defined function named `count` would shadow the built-in. Need a validation rule against this at config load time.
- **No parameterized user functions**: The expression-based approach doesn't support parameters. If users later want `my_check(from, to)`, the design needs rework. Document this as a known limitation.

## Ready for Proposal

Yes
