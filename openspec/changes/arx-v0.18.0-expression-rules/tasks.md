# Tasks: Expression Rules, Concurrent Detectors, and Severity Mapping

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Severity + Concurrency) → PR 2 (Expression Engine) → PR 3 (Integration + Polish) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Severity mapping + concurrent detectors | PR 1 | Independent; includes tests; merges to main |
| 2 | Expression tokenizer, parser, evaluator | PR 2 | Depends on PR 1; adds `internal/domain/expr.go` + tests |
| 3 | Integration (Rule.Check, EvaluateRules hook) + polish | PR 3 | Depends on PR 2; wires expression engine into audit flow |

## Phase 1: Severity Mapping

- [ ] 1.1 Add `SeverityMapping map[string]string` field to `internal/domain/config.go` `Config` struct
- [ ] 1.2 In `Config.Validate()`, iterate `c.Rules`; rewrite `rule.Severity` via mapping and validate mapped value is `error`/`warning`/`info`
- [ ] 1.3 Add table-driven cases to `internal/domain/config_test.go` covering valid mapping, unknown key passthrough, and invalid mapped value rejection

## Phase 2: Concurrent Detectors

- [ ] 2.1 Add `golang.org/x/sync` to `go.mod` (`go get golang.org/x/sync/errgroup`)
- [ ] 2.2 Rewrite `RunDetectors` in `internal/application/check.go` to use `errgroup.Group`: one goroutine per applicable detector, `sync.Mutex` on `allDependencies`, first error cancels context and returns
- [ ] 2.3 Add concurrency test in `internal/application/check_test.go`: mock detectors with `time.Sleep`, assert wall time < sequential sum and deps are aggregated correctly

## Phase 3: Expression Engine

- [ ] 3.1 Create `internal/domain/expr.go` with tokenizer: scan expression string into tokens (`ident`, `number`, `operator`, `paren`, `comma`); export `tokenize(check string) ([]token, error)`
- [ ] 3.2 In `expr.go`, implement recursive-descent parser: `parseOr() → parseAnd() → parseNot() → parseComparison() → parsePrimary()`; return `Expr` AST; export `Parse(check string) (Expr, error)`
- [ ] 3.3 In `expr.go`, define `EvalContext{Deps, Layers}` and `Value{Kind, Int, Bool, Deps}`; implement `Eval(ctx EvalContext) (Value, error)` on all AST node types
- [ ] 3.4 In `expr.go`, implement built-in functions: `count(expr)→int`, `deps(fromLayer,toLayer)→[]Dependency`, `layers()→int`, `has_circular()→bool`
- [ ] 3.5 Add `Check string` + unexported `compiledExpr Expr` to `Rule` struct in `internal/domain/rule.go`; validate syntax by calling `Parse(check)` in `Rule.Validate()` when `Check != ""`
- [ ] 3.6 In `internal/domain/audit.go` `EvaluateRules()`, when `rule.Check != ""`, evaluate `compiledExpr` against `EvalContext{dependencies, layers}` and emit a violation on `true`; skip standard from/to logic for that rule
- [ ] 3.7 In `Config.Validate()`, when `rule.Check != ""`, ensure the rule's `From`/`To` are empty (expression rules are standalone); reject mixing Check with from/to/template
- [ ] 3.8 Create `internal/domain/expr_test.go` with table-driven tests: tokenizer edge cases, parser error paths, evaluator with mock `EvalContext`, built-in function results, and full expression round-trips

## Phase 4: Polish

- [ ] 4.1 Update `ROADMAP.md` (or equivalent docs) documenting `severity_mapping`, `check` field syntax, and built-in functions
- [ ] 4.2 Run full test suite (`go test ./...`) and fix any regressions from Phase 1–3 changes
