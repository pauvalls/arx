# Rule Templates Specification

## Purpose

Parameterized rule templates allow users to express architectural constraints (max dependencies, layer isolation, layer balance) via YAML config without writing code. Templates are resolved at config load time and evaluated alongside standard rules during audit.

## Requirements

### Requirement: Template Registry

The system MUST maintain a registry of named rule templates. Each template is a function that receives params and coupling data, and returns zero or more violations. Built-in templates (`max-deps`, `no-leak`, `layer-balance`) MUST be registered at startup. Unknown template names MUST produce a validation error at config load time.

#### Scenario: Built-in templates are registered

- GIVEN the system starts
- WHEN the template registry is initialized
- THEN `max-deps`, `no-leak`, and `layer-balance` are available

#### Scenario: Unknown template name rejected

- GIVEN a rule config with `template: custom-thing`
- WHEN the config is validated
- THEN validation fails with error "unknown template: custom-thing"

### Requirement: max-deps Template

The system MUST create a violation when the number of dependencies from a source layer to target layer(s) exceeds the configured `max` threshold. The `from` param specifies the source layer. The `to` param specifies one or more target layers. The `max` param specifies the maximum allowed dependency count. When `max` is 0, any dependency from `from` to `to` is a violation. The violation message MUST include the actual count and the configured max.

#### Scenario: Dependencies under threshold pass

- GIVEN domain has 2 dependencies to infrastructure
- AND a rule with `template: max-deps`, `params: {from: domain, to: [infrastructure], max: 3}`
- WHEN the rule is evaluated
- THEN no violation is produced

#### Scenario: Dependencies over threshold produce violation

- GIVEN domain has 5 dependencies to infrastructure
- AND a rule with `template: max-deps`, `params: {from: domain, to: [infrastructure], max: 3}`
- WHEN the rule is evaluated
- THEN a violation is produced with message "domain has 5 dependencies to infrastructure (max: 3)"

#### Scenario: max=0 forbids all dependencies

- GIVEN domain has 1 dependency to infrastructure
- AND a rule with `template: max-deps`, `params: {from: domain, to: [infrastructure], max: 0}`
- WHEN the rule is evaluated
- THEN a violation is produced

#### Scenario: Multiple target layers

- GIVEN domain has 2 deps to infrastructure and 3 deps to application
- AND a rule with `template: max-deps`, `params: {from: domain, to: [infrastructure, application], max: 3}`
- WHEN the rule is evaluated
- THEN a violation is produced (total 5 > max 3) with actual count in message

### Requirement: no-leak Template

The system MUST create a violation when any file in the specified `layer` imports from a `forbidden` layer. The `layer` param specifies the protected layer. The `forbidden` param specifies one or more layers that must not be imported. The violation message MUST identify the importing file and the forbidden layer it imports from.

#### Scenario: No forbidden imports pass

- GIVEN infrastructure/order_repo.go imports only types from infrastructure
- AND a rule with `template: no-leak`, `params: {layer: infrastructure, forbidden: [domain]}`
- WHEN the rule is evaluated
- THEN no violation is produced

#### Scenario: Forbidden import produces violation

- GIVEN infrastructure/order_repo.go imports domain/entity.go
- AND a rule with `template: no-leak`, `params: {layer: infrastructure, forbidden: [domain]}`
- WHEN the rule is evaluated
- THEN a violation is produced with message "infrastructure/order_repo.go imports domain/entity.go from forbidden layer"

#### Scenario: Multiple forbidden layers

- GIVEN infrastructure/handler.go imports application/service.go
- AND a rule with `template: no-leak`, `params: {layer: infrastructure, forbidden: [domain, application]}`
- WHEN the rule is evaluated
- THEN a violation is produced identifying the import from application

### Requirement: layer-balance Template

The system MUST create a violation when any layer's total dependency count falls outside the configured `[min, max]` range. The `min` param specifies the minimum allowed dependencies per layer. The `max` param specifies the maximum allowed dependencies per layer. The violation message MUST identify the layer, its actual count, and which bound was violated.

#### Scenario: Layer within range passes

- GIVEN domain has 3 dependencies and application has 4 dependencies
- AND a rule with `template: layer-balance`, `params: {min: 2, max: 5}`
- WHEN the rule is evaluated
- THEN no violation is produced

#### Scenario: Layer below minimum produces violation

- GIVEN domain has 1 dependency
- AND a rule with `template: layer-balance`, `params: {min: 2, max: 5}`
- WHEN the rule is evaluated
- THEN a violation is produced with message "domain has 1 dependencies (min: 2)"

#### Scenario: Layer above maximum produces violation

- GIVEN infrastructure has 7 dependencies
- AND a rule with `template: layer-balance`, `params: {min: 2, max: 5}`
- WHEN the rule is evaluated
- THEN a violation is produced with message "infrastructure has 7 dependencies (max: 5)"

### Requirement: Config Integration

The system MUST support `template` (string) and `params` (map) fields on rule YAML definitions. Rules with `template` MAY also include traditional `from`/`to` fields — both are evaluated with AND logic. Missing required params for a template MUST produce a validation error at config load time. Rules without `template` MUST work identically to prior versions.

#### Scenario: Template rule parsed from YAML

- GIVEN YAML with `template: max-deps` and `params: {from: domain, to: [infrastructure], max: 3}`
- WHEN the config is parsed
- THEN the rule has `Template: "max-deps"` and `Params` populated

#### Scenario: Template and traditional fields coexist (AND logic)

- GIVEN a rule with `template: max-deps`, `params: {...}`, `from: domain`, `to: infrastructure`
- WHEN the rule is evaluated
- THEN both the template constraint AND the from/to constraint are checked

#### Scenario: Missing required param rejected

- GIVEN a rule with `template: max-deps` and `params: {from: domain}` (missing `to` and `max`)
- WHEN the config is validated
- THEN validation fails with error listing missing required params

#### Scenario: Traditional rule unaffected

- GIVEN a rule with `from: domain`, `to: infrastructure`, no `template` field
- WHEN the config is parsed and evaluated
- THEN the rule behaves identically to pre-template versions
