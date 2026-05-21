# Layers & Rules

## What Are Layers?

Layers are the fundamental building block of arx. They group files by architectural role — domain logic, application services, infrastructure, interfaces, or any custom grouping your project needs.

Every file in your project belongs to one layer. Layers define the **boundaries** that rules enforce.

```yaml
layers:
  - name: domain
    paths: ["internal/domain/**"]
    description: Enterprise business rules and domain entities
    tags: [entity, value-object, aggregate]

  - name: application
    paths: ["internal/application/**"]
    description: Use cases and application services
    tags: [use-case, command, query]

  - name: infrastructure
    paths: ["internal/infrastructure/**"]
    description: Database, HTTP clients, external integrations
    tags: [repository, persistence]

  - name: interfaces
    paths: ["internal/interfaces/**"]
    description: HTTP handlers, CLI, DTOs
    tags: [api, controller, handler]
```

### Layer Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique layer identifier |
| `paths` | Yes | Glob patterns matching files in this layer |
| `description` | No | Human-readable description |
| `tags` | No | Semantic tags for grouping and filtering |

### How Path Globs Work

Patterns use standard glob syntax:

| Pattern | Matches | Example |
|---------|---------|---------|
| `*` | Single path segment (no `/`) | `internal/*` → `internal/foo` ✓, `internal/foo/bar` ✗ |
| `**` | Zero or more path segments | `internal/**` → `internal/foo` ✓, `internal/foo/bar.go` ✓ |
| `internal/domain/**` | Everything under domain | `internal/domain/entity/user.go` ✓ |
| `legacy/` | Trailing slash = directory prefix | `legacy/foo.go` ✓ |
| `exact/path.go` | Exact file match | `exact/path.go` ✓ |

Layers are checked in order. A file matches the **first** layer whose path pattern matches. Files that don't match any layer are ignored.

## What Are Rules?

Rules define allowed and forbidden dependencies between layers. They are the "laws" of your architecture.

```yaml
rules:
  - id: domain-no-import-infrastructure
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure implementations
```

### Rule Types

| Type | Meaning | When It Violates |
|------|---------|-----------------|
| `Cannot` | Source layer MUST NOT depend on target | Dependency exists from `from` to `to` |
| `Must` | Source layer MUST depend on target | No dependency exists from `from` to `to` |
| `Can` | Informational — allowed but noted | Never violates (used for documentation) |
| `MustNotCircular` | No circular dependencies allowed | A → B → A cycle detected between layers |

### Severity Levels

| Level | CI Fail | Display | Use Case |
|-------|---------|---------|----------|
| `error` | Yes | Always | Architecture violations that MUST be fixed |
| `warning` | Configurable | Always | Should be fixed but not blocking |
| `info` | No | Optional | Informational, good practices |

You can configure per-severity behavior:

```yaml
severity_config:
  error:
    fail_build: true
    show_in_ui: true
  warning:
    fail_build: false
    show_in_ui: true
  info:
    fail_build: false
    show_in_ui: false
```

### Thresholds

Set a global violation cap. If exceeded, arx exits with code 1 regardless of severity:

```yaml
max_violations: 10
```

Set `max_violations: 0` (or omit it) for unlimited violations (backward-compatible).

### Severity Mapping

Map custom severity names to standard ones for multi-project conventions:

```yaml
severity_mapping:
  critical: error
  minor: warning
  suggestion: info
```

## Rule Overrides per Path

Override rule severity or disable it for specific directories:

```yaml
rules:
  - id: domain-no-infrastructure
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    overrides:
      - path: legacy/
        severity: warning     # Legacy code gets a pass
      - path: migration/
        enabled: false        # Migration code exempt from this rule
```

Override matching uses **longest-prefix wins**: the most specific matching override takes effect.

## File Exclusion Patterns

Exclude files globally or per-rule:

```yaml
# Global excludes — applied to ALL rules
exclude:
  - vendor/**
  - node_modules/**
  - .git/**
  - dist/**

rules:
  - id: some-rule
    from: domain
    to: [infrastructure]
    type: Cannot
    # Per-rule excludes — in addition to global
    exclude:
      - generated/**
      - **/*.pb.go
```

Patterns support the same glob syntax as layer paths.

## Complete Example

```yaml
version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain/**"]
  - name: application
    paths: ["internal/application/**"]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]
  - name: interfaces
    paths: ["internal/interfaces/**"]

rules:
  - id: domain-purity
    from: domain
    to: [application, infrastructure, interfaces]
    type: Cannot
    severity: error
    explanation: Domain must be pure — no external dependencies

  - id: app-depends-on-domain
    from: application
    to: [domain]
    type: Must
    severity: error
    explanation: Application must depend on domain abstractions

  - id: no-circular
    type: MustNotCircular
    severity: error

exclude:
  - vendor/**
  - node_modules/**

max_violations: 10
```
