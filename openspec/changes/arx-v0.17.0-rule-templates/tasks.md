# Tasks: Arx v0.17.0 — Rule Templates

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350-450 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR — all domain-layer, well-contained |
| Delivery strategy | single-PR |
| Chain strategy | none |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: none
400-line budget risk: Low
```

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Template engine + config integration + evaluation | PR 1 | All domain-layer, zero external deps |

## Phase 1: Template Engine Core

- [x] 1.1 Create `internal/domain/template.go` with `TemplateFunc` type: `func(params map[string]any, deps []Dependency, layers []Layer) []Violation`, `TemplateRegistry` map with keys `"max-deps"`, `"no-leak"`, `"layer-balance"`, and `ValidateTemplateParams(templateName string, params map[string]any) error` that checks required params exist and have correct types per the schema defined in design.md
- [x] 1.2 Implement `TemplateMaxDeps` — params: `from` (string), `to` ([]string), `max` (int). Count deps where source layer == `from` AND target layer in `to`. Return one violation if count > max, with message `"<from> has <n> dependencies to <to> (max: <max>)"`. When max=0, any dep is a violation. Multiple target layers → count all together
- [x] 1.3 Implement `TemplateNoLeak` — params: `layer` (string), `forbidden` ([]string). For each dep where source layer == `layer` AND target layer in `forbidden`, return a violation with message `"<sourceFile> imports <targetFile> from forbidden layer"`. One violation per forbidden import
- [x] 1.4 Implement `TemplateLayerBalance` — params: `min` (int), `max` (int). Count total outgoing deps per layer. For each layer where count < min, return violation `"<layer> has <n> dependencies (min: <min>)"`. For each layer where count > max, return violation `"<layer> has <n> dependencies (max: <max>)"`
- [x] 1.5 Create `internal/domain/template_test.go` with table-driven tests for all 3 templates: `TemplateMaxDeps` (under threshold, over threshold, max=0, multiple targets), `TemplateNoLeak` (no forbidden imports, single forbidden import, multiple forbidden layers), `TemplateLayerBalance` (within range, below min, above max), `ValidateTemplateParams` (missing required param, wrong type, valid params for each template)

## Phase 2: Config Integration

- [x] 2.1 Modify `internal/domain/rule.go` — add `Template string` and `Params map[string]any` fields to `Rule` struct with `yaml:"template,omitempty"` and `yaml:"params,omitempty"` tags (json tags too). Extend `Rule.Validate()` to check: if `Template != ""`, verify it exists in `TemplateRegistry`, then call `ValidateTemplateParams` and wrap errors with rule ID
- [x] 2.2 Modify `internal/domain/config.go` — extend `Config.Validate()` layer validation to also check template rule params reference valid layer names (e.g., `from`, `to`, `layer`, `forbidden` params that are layer names must exist in `layerNames` map). Skip this for non-layer-name params like `max`, `min`
- [x] 2.3 Extend `internal/domain/rule_test.go` — add table-driven test cases for `Rule.Validate()` with template field: valid template + valid params, unknown template name, missing required param, wrong param type, template + from/to coexisting, traditional rule without template (unchanged behavior)
- [x] 2.4 Extend `internal/domain/config_test.go` — add test cases for config validation with template rules: valid template rule passes, template referencing unknown layer fails, missing params fails at config level

## Phase 3: Evaluation Integration

- [x] 3.1 Modify `internal/domain/audit.go` — in `EvaluateRules`, after the standard rule loop (line ~69) and before circular dependency check, add template rule evaluation: iterate rules where `rule.Template != ""`, look up `fn := TemplateRegistry[rule.Template]`, call `fn(rule.Params, dependencies, layers)`, append returned violations to the slice (incrementing `violationIndex` for each). Template violations use the same `Violation` struct with `RuleID` set to the rule's ID and `Severity` from the rule
- [x] 3.2 Extend `internal/domain/audit_test.go` — add integration test: `EvaluateRules` with mixed standard + template rules. Verify: standard rule violations still produced, template rule violations produced, all IDs sequential with no gaps, violation count matches expected. Also test: rule with both template AND from/to fields evaluates both paths (AND logic per design)

## Phase 4: Polish

- [x] 4.1 Add integration test: parse a full `arx.yaml` containing template rules (max-deps, no-leak, layer-balance examples from design.md), run `Config.Validate()` then `EvaluateRules` with synthetic deps, assert violations match spec scenarios
- [ ] 4.2 Update `README.md` — add "Rule Templates" section documenting the 3 built-in templates, YAML syntax, and param schemas. Include the max-deps and no-leak examples from design.md
- [ ] 4.3 Update roadmap — mark v0.17.0 rule templates as in-progress/completed
