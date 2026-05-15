# Arx

[![Go Report Card](https://goreportcard.com/badge/github.com/pauvalls/arx)](https://goreportcard.com/report/github.com/pauvalls/arx)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pauvalls/arx)](go.mod)
[![Release](https://img.shields.io/github/v/release/pauvalls/arx)](https://github.com/pauvalls/arx/releases)

**Architecture audit CLI for cross-language codebases.**

Arx validates architectural rules against real codebases and explains *why* violations matter and *how* to fix them. It's not a linter — it's an **architecture guard with a teaching soul**.

```
❌ [D-01] domain/order.go:14 → infrastructure/postgres.go
   ───────────────────────────────────────────────────────
   Rule: "domain" MUST NOT depend on "infrastructure"

   Why this matters:
   The domain layer is the heart of your business logic. It should
   not know HOW data is persisted — only THAT it is persisted...

   How to fix:
   1. Define an interface in domain (e.g., OrderRepository)
   2. Move the PostgreSQL implementation to infrastructure
   3. Inject the implementation via constructor (Dependency Inversion)
```

---

## Quickstart

### Installation

```bash
# Recommended: Install latest release binary
go install github.com/pauvalls/arx/cmd/arx@latest

# Verify installation
arx --version

# Or build from source (requires Go 1.21+)
git clone https://github.com/pauvalls/arx.git
cd arx
go build ./cmd/arx
./arx --help
```

### First Use

```bash
# 1. Initialize Arx in your project
arx init

# This generates arx.yaml with:
#   - Detected layers (domain, application, infrastructure, etc.)
#   - Sensible default rules for Clean/Hexagonal Architecture
#   - Language-specific settings for Go/TypeScript

# 2. Review and customize arx.yaml
# Edit the generated file to match your architecture

# 3. Run your first audit
arx check

# 4. Run in CI mode (JSON output for pipelines)
arx check --ci
```

---

## 🆕 v0.5.0 New Features

### Configuration Presets

Start with battle-tested architecture templates:

```bash
arx init --preset clean        # Clean Architecture
arx init --preset hexagonal    # Ports & Adapters
arx init --preset ddd          # Domain-Driven Design
```

📚 **[Full Presets Guide →](docs/presets/README.md)**

### Dependency Diagrams

Visualize your architecture:

```bash
arx diagram                    # ASCII in terminal
arx diagram --format dot       # Graphviz DOT
arx diagram -o deps.dot        # Export to file
```

📚 **[Full Diagrams Guide →](docs/diagrams/README.md)**

---

## Documentation

| Topic | Description |
|-------|-------------|
| **[Presets](docs/presets/README.md)** | Configuration presets for Clean, Hexagonal, and DDD architectures |
| **[Diagrams](docs/diagrams/README.md)** | Generate and render dependency diagrams |
| **[Commands](#commands)** | CLI reference |
| **[Configuration](#configuration)** | arx.yaml format and examples |

---

## Commands

| Command | Description |
|---------|-------------|
| `arx init [path]` | Initialize arx.yaml configuration for a project |
| `arx init --preset {clean,hexagonal,ddd}` | Initialize with a preset template |
| `arx check [path]` | Run architecture audit against defined rules |
| `arx check --ci` | JSON output for CI/CD pipelines (exit code 1 on violations) |
| `arx check --watch` | Watch mode for continuous feedback on file changes |
| `arx check --interval 500ms` | Watch polling/debounce interval |
| `arx check --no-cache` | Bypass performance cache |
| `arx check --no-baseline` | Ignore baseline, report all violations |
| `arx baseline [path]` | Create baseline to suppress existing violations |
| `arx baseline --reset` | Regenerate baseline from current state |
| `arx diff [ref-before] [ref-after]` | Compare architecture between git refs |
| `arx diff --format json` | Machine-readable diff output |
| `arx audit [path]` | Run comprehensive architecture audit with health metrics |
| `arx audit --trend` | Show trend comparison with previous audit |
| `arx audit --since 2026-04-01` | Show audits since a specific date |
| `arx explain <id>` | Show detailed guidance for a specific violation |
| `arx hook install` | Install git pre-commit hook (blocks new violations) |
| `arx hook uninstall` | Remove pre-commit hook |
| `arx --version` | Show version and build info |
| `arx --help` | Show help for any command |

### Flags

```bash
arx check --config custom.yaml    # Use custom config file
arx check --format json           # Explicit JSON output
arx check --verbose               # Show detailed dependency info
arx check --watch                 # Watch mode (re-runs on file changes)
arx check --interval 1s           # Watch debounce interval (default 500ms)
arx check --no-cache              # Bypass performance cache
arx check --no-baseline           # Ignore baseline, report all violations
arx init --output config/arx.yaml # Write config to custom path
arx init --force                  # Overwrite existing config
arx init --preset clean           # Use Clean Architecture preset
arx diagram --format dot          # Output Graphviz DOT format
arx diagram -o deps.dot           # Write diagram to file
```

---

## Presets

Arx includes three architecture presets based on established patterns. See **[Presets Guide](docs/presets/README.md)** for complete documentation.

### Quick Reference

| Preset | Best For | Layers |
|--------|----------|--------|
| **clean** | Web apps, services | domain, application, infrastructure, interface |
| **hexagonal** | Testability, adapter swapping | domain, ports, adapters, infrastructure |
| **ddd** | Complex business domains | domain, application, infrastructure, interfaces |

### Usage

```bash
arx init --preset clean
arx init --preset hexagonal
arx init --preset ddd
```

Presets are starting points — review and customize the generated `arx.yaml` to match your architecture.

---

## Configuration

### Example arx.yaml

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

### Rule Types

| Type | Semantics | Example |
|------|-----------|---------|
| `cannot` | Source layer MUST NOT import target layer | domain CANNOT depend on infrastructure |
| `must` | Source layer MUST import target layer | application MUST depend on domain |
| `can` | Source layer is ALLOWED to import (informational) | cmd CAN depend on all |
| `must_not_circular` | No circular dependencies between layers | domain MUST NOT be circular with infrastructure |

### Severity Levels

| Level | Exit Code | Use Case |
|-------|-----------|----------|
| `error` | 1 | Critical violations that break architecture |
| `warning` | 0 | Architectural concerns, not blocking |
| `info` | 0 | Informational, never fails |

### Overrides

Per-directory overrides let you customize rule behavior for specific paths:

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

Overrides use **prefix matching**: `internal/legacy/` applies to all files under that tree.
If `enabled: false` matches, the rule is disabled regardless of severity overrides.
Overrides are `omitempty` — existing configs without them work unchanged.

**Exit code behavior**: When all violations are overridden, `arx check` exits with code 0
(only non-overridden violations trigger a failure).

---

## Output Formats

### Terminal (Default)

Colored, educational output with explanations:

```
╔═══════════════════════════════════════════════════════════╗
║         ARCHITECTURE VIOLATIONS DETECTED                  ║
╚═══════════════════════════════════════════════════════════╝

❌ [D-01] domain/order.go:14
   Rule: "domain" → "infrastructure"
   
   Why this matters:
   The domain layer is the heart of your business logic...
   
   How to fix:
   1. Define an interface in the domain layer
   2. Move the concrete implementation to infrastructure
   3. Inject via constructor (Dependency Inversion)

═══════════════════════════════════════════════════════════
Found 1 violation (1 errors, 0 warnings, 0 info)
Across 1 file
```

### JSON (CI/CD)

```bash
arx check --ci > results.json
```

```json
{
  "version": "1.0",
  "tool": "arx",
  "violations": [
    {
      "id": "D-01",
      "rule_id": "domain-cannot-depend-on-infrastructure",
      "severity": "error",
      "file": "internal/domain/order.go",
      "line": 14,
      "source_layer": "domain",
      "target_layer": "infrastructure",
      "import": "github.com/example/app/internal/infrastructure/postgres",
      "message": "The domain layer is the heart of your business logic..."
    }
  ],
  "summary": {
    "total": 1,
    "errors": 1,
    "warnings": 0,
    "info": 0
  }
}
```

---

## Supported Languages

| Language | Detector | Method | Status |
|----------|----------|--------|--------|
| Go | `go/ast` | AST parsing | ✅ MVP |
| TypeScript | Regex + path aliases | Pattern matching | ✅ MVP |
| Python | `ast` module | AST parsing | ✅ v0.4.0 |
| Java | `package` + `import` | Pattern matching | ✅ v0.6.0 |
| Rust | `use` statements | Regex + Cargo.toml | ✅ v0.9.0 |

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Architecture Audit
on: [push, pull_request]

jobs:
  arx:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Arx
        run: go install github.com/pauvalls/arx/cmd/arx@latest
      
      - name: Run Architecture Audit
        run: arx check --ci > arx-results.json
      
      - name: Upload Results
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: arx-results
          path: arx-results.json
```

### GitLab CI

```yaml
architecture-audit:
  image: golang:1.21
  script:
    - go install github.com/pauvalls/arx/cmd/arx@latest
    - arx check --ci > arx-results.json
  artifacts:
    reports:
      architecture: arx-results.json
```

---

## Roadmap

### ✅ v0.9.0 (Current — Overrides, Rust, GitHub Action)

- [x] `overrides[]` per-rule — Path-based severity downgrade and rule disable
- [x] Rust detector — `Cargo.toml` detection, `use` statement parsing
- [x] `.github/actions/arx-action/` — Docker-based GitHub Action for CI/CD
- [x] Override-aware exit code — 0 when only overridden violations remain
- [x] JSON `overridden_count` in summary

### ✅ v0.5.0 (Presets + Diagrams)

- [x] `arx init --preset {clean,hexagonal,ddd}` — Configuration presets
- [x] `arx diagram` — Dependency graph (ASCII + Graphviz DOT)
- [x] `arx diagram -o file.dot` — Export to Graphviz format
- [x] Violation highlighting in diagrams (red edges, [!] markers)
- [x] 3 preset templates: Clean, Hexagonal, DDD

### ✅ v0.6.0 (Java Detector)

**Target:** Q3 2026

#### Java Detector
**Status:** ✅ v0.6.0 | **Priority:** High | **Effort:** Medium

Support for Java projects via package and import statement parsing.

```bash
# Works automatically on Java projects
arx init --preset clean
arx check
```

**Scope:**
- Parse `package` and `import` statements
- Support Maven (`pom.xml`) and Gradle (`build.gradle`) module detection
- Handle static imports and wildcard imports
- Skip generated sources (`target/`, `build/`)

**Not included:**
- Annotation processing
- Bytecode analysis (source-only)

**Track Progress:** [#42](https://github.com/pauvalls/arx/issues/42)

---

#### Arx Audit (Health Reports)
**Status:** ⚪ Proposed | **Priority:** Medium | **Effort:** Medium

Generate architecture health reports with trend tracking.

```bash
arx audit --output report.md
arx audit --trend --since 2026-01-01
```

**Features:**
- Violation trends over time (improving vs degrading)
- Layer coupling matrix (which layers depend on which)
- Technical debt estimation (hours to fix violations)
- Comparison with industry benchmarks

**Output Example:**
```
Architecture Health Score: 78/100 (+5 from last month)

Trends:
  ✓ Violations decreased by 12% (23 → 20)
  ✓ Domain purity improved (3 → 1 violations)
  ⚠ Application coupling increased (2 → 4 violations)

Top Issues:
  1. domain → infrastructure (1 violation, 2h to fix)
  2. application → infrastructure (4 violations, 8h to fix)
```

**Track Progress:** [#48](https://github.com/pauvalls/arx/issues/48)

---

#### Performance Optimization
**Status:** ⚪ Proposed | **Priority:** Medium | **Effort:** High

Optimize for large codebases (>10k LOC, >100 files).

**Goals:**
- `arx check` completes in <5s for 10k LOC
- Parallel detector execution
- Incremental analysis (only changed files)
- Memory-efficient AST parsing

**Benchmarks:**
```
Current:  10k LOC → 15s
Target:   10k LOC → 5s
Current:  50k LOC → 60s
Target:   50k LOC → 20s
```

**Track Progress:** [#51](https://github.com/pauvalls/arx/issues/51)

---

### 🔜 Future (v0.10.0+)

**Target:** Q4 2026 - Q1 2027

#### C# Detector
**Status:** ⚪ Backlog | **Priority:** Low

Support for C# projects via `using` statement parsing.

**Scope:**
- Parse `using` directives
- Handle `.csproj` project files
- Skip `bin/` and `obj/` directories

**Track Progress:** [#55](https://github.com/pauvalls/arx/issues/55)

---

#### Arx Watch (Continuous Monitoring)
**Status:** ⚪ Backlog | **Priority:** Low

File watcher for real-time architecture validation.

```bash
arx watch
# Runs arx check on every file save
# Notifications via terminal, desktop, or Slack
```

**Features:**
- File system watcher (fsnotify)
- Incremental re-check (only affected files)
- Desktop notifications
- Slack/Discord webhooks

**Track Progress:** [#58](https://github.com/pauvalls/arx/issues/58)

---

#### Custom Rule DSL
**Status:** ⚪ Backlog | **Priority:** Low

Domain-specific language for complex architectural rules.

```yaml
# Example: Layer coupling limit
rule: layer-coupling-limit
  description: "No layer can depend on more than 3 other layers"
  check: |
    for layer in layers:
      if count(layer.dependencies) > 3:
        violation(layer, "excessive coupling")
```

**Features:**
- JavaScript/TypeScript-based rule engine
- Access to full dependency graph
- Custom violation messages
- Rule testing framework

**Track Progress:** [#62](https://github.com/pauvalls/arx/issues/62)

---

#### Arx Server (Web UI)
**Status:** ⚪ Backlog | **Priority:** Low

Web interface for architecture visualization and tracking.

```bash
arx server --port 8080
# Opens web UI at http://localhost:8080
```

**Features:**
- Interactive dependency graph (D3.js)
- Violation timeline and trends
- Team collaboration (comments, assignments)
- Integration with Jira, Linear, GitHub Issues

**Track Progress:** [#65](https://github.com/pauvalls/arx/issues/65)

---

## How to Contribute

### Vote on Features

Add 👍 reactions to issues you want prioritized:
- [Java Detector](https://github.com/pauvalls/arx/issues/42)
- [GitHub Action](https://github.com/pauvalls/arx/issues/45)
- [Audit Reports](https://github.com/pauvalls/arx/issues/48)

### Implement Features

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- How to write a new detector
- Adding explanation patterns
- Testing guidelines
- Code style and conventions

### Request Features

Open an issue with:
- **Use case:** What problem are you solving?
- **Current workaround:** How do you handle this today?
- **Expected behavior:** What should Arx do?
- **Example:** CLI syntax or config example

---

## Release History

| Version | Date | Features | Breaking Changes |
|---------|------|----------|------------------|
| v0.9.0 | July 2026 | Per-directory overrides, Rust detector, GitHub Action | No |
| v0.5.0 | May 2026 | Presets, Diagrams | No |
| v0.4.0 | Apr 2026 | Python detector | No |
| v0.3.0 | Mar 2026 | Explain, Circular detection | No |
| v0.2.0 | Feb 2026 | SARIF, Markdown, Cache | No |
| v0.1.0 | Jan 2026 | MVP (Go, TS detectors) | N/A |

**Release Notes:** See [Releases](https://github.com/pauvalls/arx/releases) for detailed changelogs.

---
- [ ] Layer coupling matrix visualization

---

## Why Arx?

**Architecture erodes invisibly.** No CI check catches when a domain module imports a database driver. By the time someone notices, the damage is structural.

| Problem | Traditional Tools | Arx |
|---------|------------------|-----|
| Language lock-in | ArchUnit (Java), Deptrac (PHP) | Cross-language (Go, TS, Python, Java, Kotlin, Rust) |
| Silent violations | Linters only flag style | Fails CI on architectural violations |
| No teaching | "Remove this dependency" | Explains WHY + HOW to fix |
| Static docs | ADRs in wikis, disconnected | Executable architecture rules |
| Enterprise-only | SonarQube (paid for full features) | Free, open-source (MPL-2.0) |

---

## Architecture

Arx itself follows Hexagonal Architecture:

```
arx/
├── .github/actions/arx-action/  # Docker-based GitHub Action
├── cmd/arx/                 # CLI adapter (Cobra commands)
├── internal/
│   ├── domain/              # Pure business logic
│   │   ├── layer.go         # Layer entity
│   │   ├── rule.go          # Rule entity
│   │   ├── violation.go     # Violation entity
│   │   └── audit.go         # Audit orchestration
│   ├── application/         # Use cases
│   │   ├── init.go          # Init command handler
│   │   ├── check.go         # Check command handler
│   │   └── explanations.go  # Built-in explanation library
│   ├── infrastructure/      # I/O implementations
│   │   ├── detector/go/     # Go AST detector
│   │   ├── detector/ts/     # TypeScript regex detector
│   │   ├── detector/rust/   # Rust regex detector
│   │   ├── config/yaml.go   # YAML config parser
│   │   ├── output/terminal.go # Colored terminal output
│   │   └── output/json.go   # JSON CI output
│   └── ports/               # Interfaces
│       ├── detector.go      # Detector interface
│       ├── config.go        # Config reader interface
│       └── reporter.go      # Output reporter interface
└── test/
    ├── fixtures/            # Sample projects with violations
    └── integration/         # End-to-end tests
```

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for:

- How to write a new detector
- Adding explanation patterns
- Testing guidelines
- Code style and conventions

### Quick Start for Contributors

```bash
# Clone and build
git clone https://github.com/pauvalls/arx.git
cd arx
go build ./cmd/arx

# Run tests
go test ./...

# Run Arx on itself (dogfooding)
./arx check  # Should pass with 0 violations
```

---

## License

[Mozilla Public License 2.0](LICENSE) — weak copyleft, business-friendly.

- ✅ Can be used in proprietary projects (CLI is separate work)
- ✅ Modifications to Arx source must be shared
- ✅ Not viral like GPL (doesn't infect audited projects)

---

## Acknowledgments

- **Cobra** — CLI framework (used by Hugo, Kubernetes, GitHub CLI)
- **Viper** — Configuration management
- **Lip Gloss** — Terminal styling
- **Clean Architecture** — Robert C. Martin
- **Hexagonal Architecture** — Alistair Cockburn
- **Domain-Driven Design** — Eric Evans

---

**Built with ❤️ by the open source community.**

[Report an issue](https://github.com/pauvalls/arx/issues) · [Request a feature](https://github.com/pauvalls/arx/issues/new) · [View releases](https://github.com/pauvalls/arx/releases)
