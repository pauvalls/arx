# Expression Rules & Custom DSL

Arx supports **expression-based rules** using a built-in expression language. Expression rules give you fine-grained control over architectural constraints beyond the standard `from`/`to` format.

## Quick Start

```yaml
rules:
  - id: domain-purity
    check: "count(deps(domain, infra)) == 0"
    severity: error
```

This rule triggers when the domain layer has ANY dependency on infrastructure.

## Built-in Functions

```yaml
# count() — number of dependencies between layers
check: "count(deps(domain, infra)) > 5"

# all() — true when all queried deps exist
check: "all(deps(domain, infra))"

# any() — true when at least one dep exists
check: "any(deps(application, infra))"

# has_circular() — whether there are circular dependencies
check: "!has_circular()"

# files() — count of files in a layer
check: "files(domain) > 50"

# layers() — total layer count
check: "layers() >= 3"

# violations() — count of violations for a specific rule
check: "violations(domain-no-infra) == 0"

# threshold() — check a value against min/max
check: "threshold(files(infrastructure), 10, 100)"
```

## Multi-line Checks

Use YAML arrays to AND multiple conditions:

```yaml
check:
  - "count(deps(domain, infra)) > 0"
  - "!has_circular()"
```

Both expressions must be true for the rule to trigger.

## filter() / map() — Collection Operations

```yaml
# filter deps by criteria
check: "count(filter(deps(domain, infra), \"ResolvedLayer == infra\")) > 3"

# extract field values
check: "count(map(deps(domain, infra), \"SourceFile\")) > 0"
```

Supported predicate fields:
- `SourceFile` — file path (string, == and != only)
- `SourceLine` — line number (all 6 comparison operators)
- `ImportPath` — imported package path (== and !=)
- `ResolvedLayer` — resolved layer name (== and !=)

## User-Defined Functions

Define reusable expressions in a `functions` section:

```yaml
functions:
  has_leaks: "violations(domain-no-infra) > 0"
  high_coupling: "count(deps(domain, infra)) > 10"
  is_stable: "!has_leaks && !high_coupling && !has_circular()"

rules:
  - id: architecture-health
    check: "is_stable()"
    severity: warning
```

Functions can call other functions. Circular references are detected and rejected at config load time.

## Combining with from/to Rules

Expression rules are evaluated **separately** from standard rules. You can mix both types:

```yaml
rules:
  # Standard from/to rule
  - id: domain-no-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error

  # Expression rule — fails CI if total debt is too high
  - id: debt-limit
    check: "violations(domain-no-infra) + violations(app-no-infra) < 20"
    severity: warning
```

## Complete Example

```yaml
functions:
  infra_leaks: "count(deps(domain, infra)) + count(deps(application, infra))"

rules:
  - id: strict-domain
    check: "count(deps(domain, infra)) == 0 && files(domain) > 0"
    severity: error

  - id: coupling-warning
    check: "infra_leaks() > 5"
    severity: warning

  - id: no-circles
    check: "!has_circular()"
    severity: error
```
