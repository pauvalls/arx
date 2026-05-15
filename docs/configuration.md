# Configuration

Arx uses a single `arx.yaml` file to define your architecture layers, rules, exclusions, and overrides.

## Example

```yaml
version: "1.0"

layers:
  - name: domain
    description: "Core business logic — no external dependencies"
    paths:
      - "internal/domain/**"
  - name: application
    description: "Use cases and orchestration"
    paths:
      - "internal/application/**"
  - name: infrastructure
    description: "External implementations (DB, APIs, frameworks)"
    paths:
      - "internal/infrastructure/**"
  - name: ports
    description: "Interfaces and contracts"
    paths:
      - "internal/ports/**"

rules:
  - id: domain-purity
    from: domain
    to: [infrastructure, ports]
    type: cannot
    severity: error
    explanation: |
      The domain layer must not depend on infrastructure or ports.
      Business rules should be expressible without knowing about
      databases, web frameworks, or external systems.

  - id: application-depends-on-domain
    from: application
    to: [domain]
    type: must
    severity: error
    explanation: |
      Application layer exists to orchestrate domain operations.
      If it doesn't depend on domain, business logic has leaked
      into application or the use case is empty.

  - id: infrastructure-implements-ports
    from: infrastructure
    to: [ports]
    type: can
    severity: info

exclude:
  - "**/*_test.go"
  - "**/*.spec.ts"
  - "vendor/**"
  - "node_modules/**"
```

## Layer Paths

Layers are mapped via `paths` using glob patterns:

| Pattern | Meaning | Example Match |
|---------|---------|---------------|
| `**/domain/**` | Any path containing `domain/` | `src/main/java/com/myapp/domain/Order.java` |
| `internal/domain/**` | Under `internal/domain/` | `internal/domain/user.go` |
| `com/example/domain/**` | Java package paths | `com/example/domain/order/Order.java` |
| `**/com/wedding/domain/**` | Full package path anywhere | `src/main/java/com/wedding/domain/guest/Guest.java` |

> **Tip**: For Java/Kotlin projects, use patterns like `**/com/example/domain/**` to match the package structure regardless of source directory nesting.

## Rule Types

| Type | Semantics | Example |
|------|-----------|---------|
| `cannot` | Source layer MUST NOT import target layer | domain CANNOT depend on infrastructure |
| `must` | Source layer MUST import target layer | application MUST depend on domain |
| `can` | Source layer is ALLOWED to import (informational) | cmd CAN depend on all |
| `must_not_circular` | No circular dependencies between layers | domain MUST NOT be circular with infrastructure |

## Severity Levels

| Level | Exit Code | Use Case |
|-------|-----------|----------|
| `error` | 1 | Critical violations that break architecture |
| `warning` | 0 | Architectural concerns, not blocking |
| `info` | 0 | Informational, never fails |

## Custom Rule Patterns

Rules can use a `pattern` field for regex matching on import paths:

```yaml
rules:
  - id: no-legacy
    pattern: "com/legacy/**"
    type: Cannot
    severity: error
    explanation: Legacy packages should not be imported
```

Pattern-only rules (no `from`/`to`) match **any** import that matches the regex. Combine with `from`/`to` for AND logic.

## Per-Directory Overrides

Override rule behavior for specific paths:

```yaml
rules:
  - id: domain-cannot-depend-on-infrastructure
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    overrides:
      - path: internal/legacy/
        severity: warning       # Downgrade to warning for legacy code
      - path: internal/migration/
        enabled: false          # Disable rule entirely for migration code
```

| Field | Description |
|-------|-------------|
| `path` | Directory prefix — longest match wins for severity |
| `severity` | Override severity (`error`, `warning`, `info`) |
| `enabled` | Set `false` to disable the rule for matching paths |

- Overrides use **prefix matching**: `internal/legacy/` applies to all files under that tree.
- If `enabled: false` matches, the rule is disabled regardless of severity overrides.
- When all violations are overridden, `arx check` exits with code 0.

## Excludes

The `exclude` list uses glob patterns to skip files and directories:

```yaml
exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/*_test.go"
  - "**/*.spec.ts"
```
