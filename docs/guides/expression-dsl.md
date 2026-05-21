# Expression DSL

## What Is the Expression DSL?

The Expression DSL is a small, embedded language for writing **custom architecture rules with logic**. While basic `Cannot`/`Must` rules cover common patterns, the DSL lets you express complex constraints like:

- "No more than 5 dependencies from domain to infrastructure"
- "The infrastructure layer must have at least 10 files"
- "Circular dependencies must be below 20% of total deps"
- "Any layer with more than 50 files must have at least one test file"

Expression rules are defined using the `check` field on a rule:

```yaml
rules:
  - id: infra-deps-limit
    check: count(deps(application, infrastructure)) < 5
    severity: warning
    explanation: Application should have minimal infrastructure dependencies
```

## Syntax

### Values

| Type | Example | Description |
|------|---------|-------------|
| Number | `42`, `0`, `-1` | Integer values |
| String | `domain`, `infrastructure` | Layer names and identifiers |
| Boolean | Result of `>`, `<`, `==`, `!=` | Truth values |

### Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `>` | Greater than | `count(...) > 5` |
| `<` | Less than | `count(...) < 10` |
| `>=` | Greater or equal | `count(...) >= 1` |
| `<=` | Less or equal | `count(...) <= 20` |
| `==` | Equal | `count(...) == 0` |
| `!=` | Not equal | `count(...) != 0` |
| `&&` | Logical AND | `cond1 && cond2` |
| `\|\|` | Logical OR | `cond1 \|\| cond2` |
| `!` | Logical NOT | `!cond` |

### Parentheses

Use parentheses to group expressions:

```
(count(deps(domain, infra)) < 5) && (layers() >= 3)
```

## Built-in Functions

### `count(expr)`

Returns the integer count of items in an expression result.

```yaml
check: count(deps(domain, infrastructure)) > 0
# True when domain has any dependency on infrastructure
```

### `deps(from, to)`

Returns the list of dependencies from one layer to another.

```yaml
check: count(deps(application, domain)) > 0
# Application must depend on domain (enforces Must-type rule)
```

Both arguments are layer names as defined in your `arx.yaml`.

### `layers()`

Returns the number of configured layers.

```yaml
check: layers() >= 3
# Must have at least 3 layers (domain, application, infrastructure)
```

### `has_circular()`

Returns `true` if circular dependencies exist between layers.

```yaml
check: !has_circular()
# No circular dependencies allowed
```

### `files(layer)`

Returns the count of files in a given layer.

```yaml
check: files(infrastructure) >= files(domain)
# Infrastructure must have at least as many files as domain
```

### `ratio(numerator, denominator)`

Returns integer division of two values. Useful for percentages and thresholds.

```yaml
check: ratio(count(deps(infrastructure, domain)), count(deps(domain, infrastructure))) < 2
# Infrastructure → domain deps should be less than 2x domain → infrastructure
```

### `violations(rule_id)`

Returns the count of violations for a specific rule ID.

```yaml
check: violations("domain-no-infra") == 0
# No violations for a specific rule
```

### `threshold(rule_id)`

Returns the `max_violations` threshold for a specific rule (or global if rule has none).

```yaml
check: count(deps(domain, infra)) < threshold("max-deps")
# Dynamic threshold from config
```

### `all(expr, predicate)`

Returns `true` if ALL items in a collection match a predicate.

```yaml
# Not yet implemented in v0.57 — reserved for future use
```

### `any(expr, predicate)`

Returns `true` if ANY item in a collection matches a predicate.

```yaml
# Not yet implemented in v0.57 — reserved for future use
```

### `filter(deps(from, to), "field op value")`

Filters a dependency list by a predicate string. Returns only the matching dependencies.

```yaml
check: count(filter(deps(application, infrastructure), "severity == error")) == 0
# No error-severity deps from application to infrastructure
```

Supported predicates: `==`, `!=`, `>`, `<`, `>=`, `<=`

### `map(deps(from, to), "field")`

Extracts a field value from each dependency in a list, returning a ValueList.

```yaml
check: count(map(deps(application, domain), "source_layer")) > 0
# Count unique source layers among app→domain deps
```

## User-Defined Functions

You can define reusable expressions in the `functions` section:

```yaml
functions:
  is_clean: "count(deps(domain, infrastructure)) == 0 && count(deps(application, infrastructure)) == 0"
  has_min_coverage: "ratio(files(test), files(src)) >= 3"

rules:
  - id: arch-clean
    check: is_clean
    severity: error

  - id: test-coverage
    check: has_min_coverage
    severity: warning
```

### Rules for User Functions

- Names must match `[a-zA-Z_][a-zA-Z0-9_]*`
- Cannot shadow built-in function names (`count`, `deps`, `layers`, etc.)
- Can call other user functions (direct recursion is **not** allowed — cycle detection prevents it)
- Functions are compiled and validated at config load time

## Full Reference Table

| Function | Args | Returns | Description |
|----------|------|---------|-------------|
| `count` | 1 (collection) | Integer | Counts items in a collection |
| `deps` | 2 (from, to) | DepList | Dependencies between two layers |
| `layers` | 0 | Integer | Number of configured layers |
| `has_circular` | 0 | Boolean | Whether circular deps exist |
| `files` | 1 (layer) | Integer | File count in a layer |
| `ratio` | 2 (num, den) | Integer | Integer division of two values |
| `violations` | 1 (rule_id) | Integer | Violation count for a rule |
| `threshold` | 1 (rule_id) | Integer | Max violations threshold |
| `all` | 2 (expr, pred) | Boolean | All items match predicate |
| `any` | 2 (expr, pred) | Boolean | Any item matches predicate |
| `filter` | 2 (deps, pred) | DepList | Filter deps by predicate |
| `map` | 2 (deps, field) | ValueList | Extract field from each dep |

## Real-World Examples

### Enforce layer balance

```yaml
rules:
  - id: layer-balance
    check: files(domain) >= 5 && files(application) >= 5 && files(infrastructure) >= 5
    severity: warning
    explanation: Each layer should have at least 5 files
```

### Prevent excessive coupling

```yaml
rules:
  - id: no-excessive-coupling
    check: ratio(count(deps(domain, infrastructure)), files(domain)) < 3
    severity: warning
    explanation: Average deps per domain file should be less than 3
```

### Multi-condition rule

```yaml
rules:
  - id: clean-architecture-gate
    check: >
      count(deps(domain, infrastructure)) == 0
      && count(deps(domain, interfaces)) == 0
      && count(deps(application, interfaces)) == 0
    severity: error
    explanation: Clean Architecture gate — no layer-skipping dependencies
```
