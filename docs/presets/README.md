# Arx Configuration Presets

Presets provide battle-tested starting points for common architectural patterns.

## Quick Start

```bash
# Clean Architecture
arx init --preset clean

# Hexagonal Architecture (Ports & Adapters)
arx init --preset hexagonal

# Domain-Driven Design
arx init --preset ddd
```

## What's Included

Each preset provides:

- ✅ **Pre-configured layers** with conventional directory paths
- ✅ **Architectural rules** based on established best practices
- ✅ **Language-specific overrides** for Go and TypeScript
- ✅ **Common exclude patterns** (vendor, node_modules, etc.)
- ✅ **Built-in explanations** for each rule

## Available Presets

### Clean Architecture

**Best for:** Web applications, services with clear separation of concerns

**Layers:**
| Layer | Description | Typical Paths |
|-------|-------------|---------------|
| `domain` | Business logic, entities, domain services | `internal/domain/**`, `domain/**` |
| `application` | Use cases, application services | `internal/application/**`, `application/**` |
| `infrastructure` | Database, external APIs, frameworks | `internal/infrastructure/**`, `infra/**` |
| `interface` | HTTP handlers, CLI, presenters | `internal/interface/**`, `handlers/**` |

**Key Rules:**
1. **Domain Purity** — Domain CANNOT depend on application, infrastructure, or interface
2. **Application Isolation** — Application CANNOT depend on infrastructure or interface
3. **Dependency Flow** — Dependencies flow inward: interface → application → domain

**Example Output:**
```
$ arx init --preset clean
✓ Written to arx.yaml (preset: clean)
  Loaded 4 layer(s): domain, application, infrastructure, interface
  Generated 5 rule(s)

Review and adjust the configuration before running 'arx check'.
```

### Hexagonal Architecture (Ports & Adapters)

**Best for:** Systems requiring high testability and adapter swapping

**Layers:**
| Layer | Description | Typical Paths |
|-------|-------------|---------------|
| `domain` | Core business model (center) | `internal/domain/**`, `model/**` |
| `ports` | Interfaces defining contracts | `internal/ports/**`, `interfaces/**` |
| `adapters` | Implementations (driving/driven) | `internal/adapters/**`, `driving/**` |
| `infrastructure` | Shared utilities, configuration | `internal/infrastructure/**`, `config/**` |

**Key Rules:**
1. **Domain Isolation** — Domain is completely isolated (no dependencies outward)
2. **Port Purity** — Ports depend only on domain
3. **Adapter Contracts** — Adapters MUST implement ports
4. **Adapter Dependencies** — Adapters MAY depend on domain

### Domain-Driven Design (DDD)

**Best for:** Complex business domains with rich business logic

**Layers:**
| Layer | Description | Typical Paths |
|-------|-------------|---------------|
| `domain` | Entities, value objects, aggregates | `internal/domain/**`, `model/**` |
| `application` | Application services, CQRS handlers | `internal/application/**`, `usecases/**` |
| `infrastructure` | Repositories, event bus, external services | `internal/infrastructure/**`, `persistence/**` |
| `interfaces` | APIs, CLI, event handlers | `internal/interfaces/**`, `api/**` |

**Key Rules:**
1. **Domain Isolation** — Domain is isolated from external concerns
2. **Application Dependency** — Application MUST depend on domain
3. **Application Isolation** — Application CANNOT depend on infrastructure
4. **Interface Flow** — Interfaces MUST go through application services

## Customizing Presets

Presets are starting points, not final solutions. After initialization:

1. **Review generated `arx.yaml`**
2. **Adjust layer paths** to match your project structure
3. **Add/remove rules** based on your architectural decisions
4. **Configure language overrides** if needed

### Example: Customizing Clean Preset

```yaml
# arx.yaml
version: "1.0"

# Add custom layer
layers:
  - name: domain
    paths:
      - "internal/domain/**"
      - "pkg/domain/**"  # Add your custom path
  
  - name: events  # New layer
    description: "Domain events and event handlers"
    paths:
      - "internal/events/**"

# Add custom rule
rules:
  - id: events-isolation
    from: events
    to: [infrastructure]
    type: Cannot
    severity: error
    explanation: |
      Domain events must not depend on infrastructure.
      Events are pure domain concepts.
```

## Preset Comparison

| Feature | Clean | Hexagonal | DDD |
|---------|-------|-----------|-----|
| **Layer Count** | 4 | 4 | 4 |
| **Focus** | Separation of concerns | Testability & swapping | Business logic richness |
| **Best For** | Web apps, services | Plugin architectures | Complex domains |
| **Learning Curve** | Low | Medium | High |
| **Flexibility** | High | Very High | Medium |

## When to Use Each

### Choose Clean Architecture if:
- Building a web application or API
- Team is familiar with layered architectures
- You want clear separation between business logic and technical concerns
- Quick onboarding is important

### Choose Hexagonal if:
- You need to swap adapters frequently (e.g., different databases)
- Testability is a primary concern
- Building a library or framework
- You want maximum flexibility at boundaries

### Choose DDD if:
- Business logic is complex and evolving
- Team understands DDD concepts (entities, aggregates, value objects)
- You're building a domain-focused system (not CRUD)
- Long-term maintainability is critical

## Migration Guide

### From Auto-Detect to Preset

If you already have an `arx.yaml` from `arx init` (auto-detect):

1. **Backup existing config:**
   ```bash
   cp arx.yaml arx.yaml.backup
   ```

2. **Generate preset config:**
   ```bash
   arx init --preset clean --output arx.new.yaml
   ```

3. **Compare and merge:**
   ```bash
   diff -u arx.yaml arx.new.yaml
   ```

4. **Manually merge** custom rules from old to new config

### From Preset to Custom

1. **Start with preset** that's closest to your needs
2. **Run `arx check`** to identify violations
3. **Adjust rules** based on actual codebase
4. **Document deviations** in comments

## Troubleshooting

### "Preset not found" Error

```bash
$ arx init --preset microservices
Error: failed to initialize with preset "microservices": preset "microservices" not found. Available presets: clean, hexagonal, ddd
```

**Solution:** Use one of the built-in presets: `clean`, `hexagonal`, or `ddd`.

### "File already exists" Error

```bash
$ arx init --preset clean
Error: configuration file already exists: arx.yaml
Use --force to overwrite
```

**Solution:** Either:
- Use `--force` to overwrite: `arx init --preset clean --force`
- Or specify different output: `arx init --preset clean --output arx.clean.yaml`

### Preset Rules Too Strict

If the preset generates many violations:

1. **Review violations** — some may be legitimate issues to fix
2. **Adjust severity** — change `error` to `warning` for transitional period
3. **Add exclusions** — temporarily exclude problematic files
4. **Iterate** — fix violations incrementally

## Contributing New Presets

To propose a new preset:

1. **Create YAML file** in `internal/infrastructure/preset/{name}.yaml`
2. **Follow schema** — match existing preset structure
3. **Add tests** — ensure preset loads and validates
4. **Document** — add to this guide

See existing presets for reference:
- [`clean.yaml`](../internal/infrastructure/preset/clean.yaml)
- [`hexagonal.yaml`](../internal/infrastructure/preset/hexagonal.yaml)
- [`ddd.yaml`](../internal/infrastructure/preset/ddd.yaml)

## Resources

- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture by Alistair Cockburn](https://alistair.cockburn.us/hexagonal-architecture/)
- [Domain-Driven Design by Eric Evans](https://domainlanguage.com/ddd/)
