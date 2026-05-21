# Config Reference

This page documents every field in `arx.yaml`. Each entry includes the type, whether it's required, default value, description, and an example.

## `$schema`

| Field | Value |
|-------|-------|
| **Type** | `string` (URI) |
| **Required** | No |
| **Default** | `""` |
| **Description** | JSON Schema reference for IDE autocompletion |

```yaml
$schema: ./arx-schema.json
```

## `version`

| Field | Value |
|-------|-------|
| **Type** | `string` |
| **Required** | Yes |
| **Default** | — |
| **Description** | Configuration format version |

```yaml
version: "1.0"
```

## `layers`

| Field | Value |
|-------|-------|
| **Type** | `array` of [Layer](#layer) |
| **Required** | Yes (at least 1) |
| **Default** | — |
| **Description** | Architectural layers in the system |

```yaml
layers:
  - name: domain
    paths: ["internal/domain/**"]
    description: Core business logic
    tags: [entity, value-object]
```

### Layer

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | `string` | Yes | — | Unique layer identifier |
| `paths` | `string[]` | Yes | — | Glob patterns matching files in this layer |
| `description` | `string` | No | `""` | Human-readable description |
| `tags` | `string[]` | No | `[]` | Semantic tags for grouping and filtering |

## `rules`

| Field | Value |
|-------|-------|
| **Type** | `array` of [Rule](#rule) |
| **Required** | Yes |
| **Default** | — |
| **Description** | Architectural rules to enforce |

```yaml
rules:
  - id: domain-no-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
```

### Rule

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | `string` | Yes | — | Unique rule identifier |
| `from` | `string` | Conditional | `""` | Source layer name |
| `to` | `string[]` | Conditional | `[]` | Target layer names |
| `type` | `string` | No | `"Cannot"` | Rule type: `Cannot`, `Must`, `Can`, `MustNotCircular` |
| `severity` | `string` | No | `"error"` | Violation severity: `error`, `warning`, `info` |
| `explanation` | `string` | No | `""` | Human-readable explanation of the rule |
| `pattern` | `string` | No | `""` | Regex pattern for import-path matching (pattern-only rule) |
| `template` | `string` | No | `""` | Template name: `max-deps`, `no-leak`, `layer-balance` |
| `params` | `object` | No | `{}` | Parameters for template rules |
| `check` | `string` or `string[]` | No | `""` | Expression DSL check (cannot mix with `from`/`to`) |
| `overrides` | `object[]` | No | `[]` | Per-directory severity/enabled overrides |
| `exclude` | `string[]` | No | `[]` | Per-rule file exclusion patterns (glob) |
| `wasm` | `object` | No | — | WASM policy configuration |

**Required fields depend on rule type:**

| Rule Type | Required Fields |
|-----------|----------------|
| Standard (`from`/`to`) | `id`, `from`, `to` |
| Pattern-only | `id`, `pattern` |
| Template | `id`, `template` |
| Expression (`check`) | `id`, `check` |
| WASM | `id`, `wasm` |

#### Rule Override

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | `string` | Yes | — | Directory path prefix for the override |
| `severity` | `string` | No | — | Override severity: `error`, `warning`, `info` |
| `enabled` | `boolean` | No | `true` | Whether the rule is enabled for this path |

#### WASM Config

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | `string` | Yes | — | Path to the `.wasm` binary |
| `params` | `object` | No | `{}` | Parameters passed to the WASM module |

## `language_overrides`

| Field | Value |
|-------|-------|
| **Type** | `map<string, LanguageOverride>` |
| **Required** | No |
| **Default** | `{}` |
| **Description** | Language-specific import/comment configuration |

```yaml
language_overrides:
  go:
    extensions: [".go"]
    comment: "//"
    import: "import"
  typescript:
    extensions: [".ts", ".tsx"]
    comment: "//"
    import: "import"
```

### LanguageOverride

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `extensions` | `string[]` | No | `[]` | File extensions for this language |
| `comment` | `string` | No | `""` | Comment prefix (e.g. `//`, `#`) |
| `import` | `string` | No | `""` | Import keyword (e.g. `import`, `use`, `require`) |

## `exclude`

| Field | Value |
|-------|-------|
| **Type** | `string[]` |
| **Required** | No |
| **Default** | `[]` |
| **Description** | Global file exclusion patterns (glob) |

```yaml
exclude:
  - vendor/**
  - node_modules/**
  - .git/**
  - dist/**
  - build/**
  - tmp/**
  - testdata/**
```

## `severity_config`

| Field | Value |
|-------|-------|
| **Type** | `map<severity, SeverityConfig>` |
| **Required** | No |
| **Default** | `{}` |
| **Description** | Per-severity behavior configuration |

```yaml
severity_config:
  error:
    fail_build: true
    show_in_ui: true
  warning:
    fail_build: false
    show_in_ui: true
```

### SeverityConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `fail_build` | `boolean` | Yes | — | Whether this severity causes build failure |
| `show_in_ui` | `boolean` | Yes | — | Whether to display in UI output |

## `max_violations`

| Field | Value |
|-------|-------|
| **Type** | `integer` |
| **Required** | No |
| **Default** | `0` (unlimited) |
| **Description** | Maximum violations before failing. `0` = unlimited |

```yaml
max_violations: 10
```

## `severity_mapping`

| Field | Value |
|-------|-------|
| **Type** | `map<string, string>` |
| **Required** | No |
| **Default** | `{}` |
| **Description** | Map custom severity names to standard ones (`error`, `warning`, `info`) |

```yaml
severity_mapping:
  critical: error
  minor: warning
  suggestion: info
```

## `functions`

| Field | Value |
|-------|-------|
| **Type** | `map<string, string>` |
| **Required** | No |
| **Default** | `{}` |
| **Description** | User-defined expression functions |

```yaml
functions:
  is_clean: "count(deps(domain, infrastructure)) == 0"
  has_min_coverage: "ratio(files(test), files(src)) >= 3"
```

## `cross_language`

| Field | Value |
|-------|-------|
| **Type** | `object` |
| **Required** | No |
| **Default** | — |
| **Description** | Cross-language dependency detection configuration |

```yaml
cross_language:
  mappings:
    - source_pattern: "proto/**/*.proto"
      generated_pattern: "**/*.pb.go"
      language: go
      match_strategy: stem
```

### CrossLanguageMapping

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `source_pattern` | `string` | Yes | — | Glob pattern for source files |
| `generated_pattern` | `string` | Yes | — | Glob pattern for generated files |
| `language` | `string` | Yes | — | Target language for generated files |
| `match_strategy` | `string` | No | `"stem"` | Matching strategy: `stem` or `glob` |
| `header_patterns` | `string[]` | No | Built-in defaults | Custom header patterns for generated-file detection |

## `plugins`

| Field | Value |
|-------|-------|
| **Type** | `array` of [PluginConfig](#pluginconfig) |
| **Required** | No |
| **Default** | `[]` |
| **Description** | External plugin detectors for custom language support |

```yaml
plugins:
  - name: dart-detector
    command: dart run bin/detect.dart
    languages: [dart]
    timeout: 30s
    extensions: [.dart]
```

### PluginConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | `string` | Yes | — | Unique plugin name (must not conflict with built-in detectors) |
| `command` | `string` | Yes | — | Plugin command to execute |
| `args` | `string[]` | No | `[]` | Additional arguments to the command |
| `languages` | `string[]` | Yes | — | Languages this plugin supports |
| `timeout` | `string` | No | `"30s"` | Plugin execution timeout (Go duration) |
| `extensions` | `string[]` | No | `[]` | File extensions this plugin handles |

## `workspace`

| Field | Value |
|-------|-------|
| **Type** | `object` |
| **Required** | No |
| **Default** | — |
| **Description** | Workspace configuration (inline alternative to `arx-workspace.yaml`) |

See the [Workspace Mode guide](guides/workspace-mode.md) for full details.

```yaml
workspace:
  projects:
    - path: services/auth
    - path: services/billing
      override:
        max_violations: 5
```

---

## Complete Example

```yaml
$schema: ./arx-schema.json
version: "1.0"

layers:
  - name: domain
    paths: ["internal/domain/**"]
    description: Core business logic
    tags: [entity, value-object]
  - name: application
    paths: ["internal/application/**"]
    description: Use cases
    tags: [use-case]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]
    description: External integrations
    tags: [repository]
  - name: interfaces
    paths: ["internal/interfaces/**"]
    description: HTTP handlers, CLI
    tags: [api, handler]

rules:
  - id: domain-purity
    from: domain
    to: [application, infrastructure, interfaces]
    type: Cannot
    severity: error
    explanation: Domain must be pure
    overrides:
      - path: legacy/
        severity: warning
  - id: app-depends-domain
    from: application
    to: [domain]
    type: Must
    severity: error
  - id: no-circular
    type: MustNotCircular
    severity: error
  - id: infra-deps-limit
    check: count(deps(application, infrastructure)) < 5
    severity: warning
  - id: wasm-policy
    wasm:
      path: policies/layer-balance.wasm
      params: { min: 3, max: 8 }
    severity: warning

language_overrides:
  go:
    extensions: [".go"]
    comment: "//"
    import: "import"
  typescript:
    extensions: [".ts", ".tsx"]
    comment: "//"
    import: "import"

exclude:
  - vendor/**
  - node_modules/**
  - .git/**
  - dist/**
  - build/**
  - testdata/**

severity_config:
  error:
    fail_build: true
    show_in_ui: true
  warning:
    fail_build: true
    show_in_ui: true
  info:
    fail_build: false
    show_in_ui: false

max_violations: 10

severity_mapping:
  critical: error
  minor: warning

functions:
  is_clean: "count(deps(domain, infrastructure)) == 0"
  balanced: "files(infrastructure) >= files(domain) && files(application) >= files(domain)"

cross_language:
  mappings:
    - source_pattern: "proto/**/*.proto"
      generated_pattern: "**/*.pb.go"
      language: go
      match_strategy: stem
    - source_pattern: "openapi/**/*.yaml"
      generated_pattern: "**/*.api.ts"
      language: typescript
      match_strategy: glob
      header_patterns: ["@generated", "auto-generated"]

plugins:
  - name: dart-detector
    command: dart run bin/detect.dart
    languages: [dart]
    timeout: 30s
    extensions: [.dart]
```
