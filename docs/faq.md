# Frequently Asked Questions

## General

### What architectures does arx support?

Arx is architecture-agnostic. You define your architecture through layers in `arx.yaml`:

- **Clean Architecture** — `domain`, `application`, `infrastructure`, `interfaces`
- **Hexagonal (Ports & Adapters)** — `core`, `ports`, `adapters`
- **DDD (Domain-Driven Design)** — `aggregate`, `entity`, `valueobject`, `repository`
- **Layered** — `presentation`, `business`, `data`, `cross-cutting`
- **Any custom** — name your layers whatever you want

Arx doesn't enforce any specific architecture — it enforces **your** rules.

### How does arx compare to arch-lint / dependency-cruiser / jq?

Arx is **not** trying to be better than those tools — it works at a different level:

- **arch-lint** — Go-only, less flexible rule system
- **dependency-cruiser** — TypeScript/JavaScript only (no Go, Python, Rust, etc.)
- **jq** — Not an architecture tool at all

Arx is unique because it's **cross-language** (10 languages) and has a **teaching soul** — every violation explains WHY it matters and HOW to fix it. For context: use the right tool for your stack. If you're Go-only, arch-lint might suffice. If you're multi-language, arx is the only option.

---

## Configuration

### How do I add a custom rule?

Add a rule to your `arx.yaml`:

```yaml
rules:
  - id: my-custom-rule
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
```

For complex logic, use the [Expression DSL](guides/expression-dsl.md):

```yaml
rules:
  - id: infra-deps-limit
    check: count(deps(domain, infrastructure)) < 5
    severity: warning
```

### How do I exclude files/directories?

**Globally** (all rules):

```yaml
exclude:
  - vendor/**
  - node_modules/**
  - generated/**
```

**Per-rule**:

```yaml
rules:
  - id: my-rule
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    exclude:
      - legacy/**
      - **/*.pb.go
```

### How do I change a rule's severity for specific directories?

Use overrides:

```yaml
rules:
  - id: domain-purity
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    overrides:
      - path: legacy/
        severity: warning    # Downgrade for legacy code
      - path: migration/
        enabled: false       # Exempt migration code entirely
```

---

## Detection

### Why isn't my language detected?

1. **Run `arx doctor`** to check if arx can find your project:

```bash
arx doctor
```

2. **Check the marker file** — each language needs a specific file:

| Language | Marker File |
|----------|-------------|
| Go | `go.mod` |
| TypeScript | `tsconfig.json` |
| Python | `requirements.txt`, `setup.py`, `setup.cfg`, `pyproject.toml` |
| Java | `pom.xml`, `build.gradle` |
| Kotlin | `build.gradle.kts` |
| Rust | `Cargo.toml` |
| C# | `.csproj`, `.sln` |
| Ruby | `Gemfile` |
| PHP | `composer.json` |
| Swift | `Package.swift` |

3. **Missing language?** Write a [custom plugin detector](tutorials/custom-plugin.md).

### What does "circular dependency" mean?

A circular dependency occurs when layer A depends on layer B, and layer B depends (directly or transitively) back on layer A. For example:

```
domain → infrastructure → application → domain
```

This creates a cycle that makes the code:
- **Hard to understand** — no clear direction of flow
- **Brittle** — changes in one layer can break any other layer
- **Hard to test** — components can't be tested in isolation

Use `MustNotCircular` rules to detect and prevent these:

```yaml
rules:
  - id: no-circular
    type: MustNotCircular
    severity: error
```

### Can I use arx with a monorepo?

Yes! Use [Workspace Mode](guides/workspace-mode.md):

```bash
arx workspace
```

Configure projects in `arx-workspace.yaml` with shared rules and per-project overrides.

---

## Baselines & Suppression

### What's the difference between suppress and baseline?

- **Baseline** — Captures ALL current violations to `.arx-baseline.json`. Future checks only report NEW violations. Use it when adopting arx on existing codebases.

```bash
arx baseline
```

- **Suppress** — (Not yet implemented) Would allow suppressing individual violations by ID.

The baseline is the recommended approach for incremental adoption.

### How do I update the baseline?

```bash
arx baseline --reset
```

Or let it auto-refresh: after 3 consecutive clean checks (configurable via `--refresh-threshold`), arx automatically updates the baseline.

---

## CI/CD

### How do I use arx in CI?

See the [CI/CD tutorial](tutorials/ci-cd.md) for GitHub Actions and GitLab CI examples.

Quick start for GitHub Actions:

```yaml
- name: Run architecture audit
  run: |
    go install github.com/pauvalls/arx/cmd/arx@latest
    arx check --ci
```

### How do I make arx fail CI on violations?

By default, `arx check` exits with code 1 when violations are found. CI pipelines treat non-zero exits as failures. No special configuration needed.

If you have existing violations and want incremental enforcement:

```bash
arx baseline     # Capture existing violations (saved to .arx-baseline.json)
arx check        # Only NEW violations fail CI
```

### How do I check only PR-introduced violations?

Use `arx pr-check`:

```bash
arx pr-check --base origin/main --head feature/branch
```

This only reports violations on lines changed by the PR.

---

## Maintenance

### How do I update arx?

```bash
# Via Go
go install github.com/pauvalls/arx/cmd/arx@latest

# Via Homebrew
brew upgrade pauvalls/tap/arx
```

Check your current version:

```bash
arx --version
```

See the [CHANGELOG](../CHANGELOG.md) for what's new.

### How do I format my arx.yaml?

```bash
arx fmt
```

Use `--check` in CI to ensure consistent formatting:

```bash
arx fmt --check
```
