---
name: arx-setup
description: >
  Set up and configure arx architecture audit in any project.
  Trigger: When the user says "setup arx", "configure arx", "arx init",
  "arx setup", "analizar arquitectura", or similar.
license: Apache-2.0
metadata:
  author: arx
  version: "1.0"
---

## When to Use

- Set up arx in a new project
- Generate/regenerate arx.yaml from existing code
- Analyze project architecture with arx
- Add cross-language dependency detection to an existing setup

## Workflow

### 1. Scan the project

```bash
# Detect languages
ls go.mod tsconfig.json package.json Cargo.toml build.gradle pom.xml 2>/dev/null
# Check directory structure for architectural patterns
ls -d internal/*/ src/*/ cmd/ proto/ specs/ 2>/dev/null
```

### 2. Detect with arx

```bash
arx init --detect
```

### 3. Generate configuration

Based on detected patterns, write or enhance `arx.yaml`:

- **Domain/Application/Infrastructure** layers for Clean/Hexagonal projects
- **Cross-language mappings** when proto/OpenAPI specs exist
- **Expression rules** for specific constraints (thresholds, circular deps)
- **Functions** for reusable architectural checks

### 4. Validate and run

```bash
arx config validate
arx check
arx baseline   # For existing codebases (suppress known violations)
```

### 5. Iterate

If violations are unexpected, adjust layer paths or add exclusions.
Use `arx explain <violation-id>` for detailed guidance.

## Key Config Patterns

### Clean Architecture (Go)

```yaml
layers:
  - name: domain
    paths: ["internal/domain/**"]
  - name: application
    paths: ["internal/application/**"]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]
  - name: presentation
    paths: ["cmd/**", "internal/handler/**"]

rules:
  - id: domain-purity
    check: "count(deps(domain, infrastructure)) == 0"
    severity: error
```

### Cross-Language (proto → generated code)

```yaml
cross_language:
  mappings:
    - source_pattern: "proto/**/*.proto"
      generated_pattern: "**/*.pb.go"
      language: "go"
```

### NestJS / TypeScript

```yaml
layers:
  - name: domain
    paths: ["src/**/domain/**", "src/**/entities/**"]
  - name: application
    paths: ["src/**/application/**", "src/**/use-cases/**"]
  - name: infrastructure
    paths: ["src/**/infrastructure/**", "src/**/repositories/**"]

rules:
  - id: domain-no-infra
    check: "count(deps(domain, infrastructure)) == 0"
    severity: error
```

## Commands

```bash
# Generate config
arx init

# Dry-run detection
arx init --detect

# With preset
arx init --preset hexagonal

# Format config
arx fmt

# Run audit
arx check

# Create baseline
arx baseline

# Explain violations
arx explain D-01

# Auto-fix suggestions
arx suggest
```

## Resources

- Config docs: `docs/configuration.md`
- Expression rules: `docs/expression-rules.md`
- Cross-language: `docs/cross-language.md`
- Suggest & Explain: `docs/suggest.md`
