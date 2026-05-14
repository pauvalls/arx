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
| `arx diagram [path]` | Generate dependency diagram (ASCII or DOT) |
| `arx explain <id>` | Show detailed guidance for a specific violation |
| `arx --version` | Show version and build info |
| `arx --help` | Show help for any command |

### Flags

```bash
arx check --config custom.yaml    # Use custom config file
arx check --format json           # Explicit JSON output
arx check --verbose               # Show detailed dependency info
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
| Python | Planned | `ast` module | 🔜 v0.3.0 |
| Java | Planned | `package` + `import` | 🔜 v0.3.0 |
| Rust | Planned | `use` statements | 🔜 v0.3.0 |

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

### ✅ v0.5.0 (Current — Presets + Diagrams)

- [x] `arx init --preset {clean,hexagonal,ddd}` — Configuration presets
- [x] `arx diagram` — Dependency graph (ASCII + Graphviz DOT)
- [x] `arx diagram -o file.dot` — Export to Graphviz format
- [x] Violation highlighting in diagrams (red edges, [!] markers)
- [x] 3 preset templates: Clean, Hexagonal, DDD

### ✅ v0.4.0

- [x] Python detector (import extraction)
- [x] TDD — 7 tests, 100% pass

### ✅ v0.3.0

- [x] `arx explain <id>` — Full detailed violation guidance
- [x] `arx explain --list` — List all cached violations
- [x] Circular dependency detection (DFS algorithm)
- [x] TDD — 20+ tests for circular detection

### ✅ v0.2.0

- [x] SARIF 2.1.0 output (GitHub code scanning integration)
- [x] Markdown report output
- [x] Violation cache with 24h TTL
- [x] Warning severity level support

### ✅ v0.1.0 (MVP)

- [x] `arx init` — Project scanning, language detection, config generation
- [x] `arx check` — Rule evaluation with terminal output
- [x] `arx check --ci` — JSON output for CI/CD
- [x] Go detector (AST-based)
- [x] TypeScript detector (regex-based)
- [x] Built-in explanations library (12+ patterns)
- [x] Hexagonal architecture (clean separation)

### 🔜 v0.6.0

- [ ] Java detector
- [ ] GitHub Action wrapper (`arx-action`)
- [ ] `arx audit` — Health report with trend tracking
- [ ] Performance optimization for large codebases (>10k lines)
- [ ] Layer coupling matrix visualization

---

## Why Arx?

**Architecture erodes invisibly.** No CI check catches when a domain module imports a database driver. By the time someone notices, the damage is structural.

| Problem | Traditional Tools | Arx |
|---------|------------------|-----|
| Language lock-in | ArchUnit (Java), Deptrac (PHP) | Cross-language (Go, TS, Python, Java, Rust) |
| Silent violations | Linters only flag style | Fails CI on architectural violations |
| No teaching | "Remove this dependency" | Explains WHY + HOW to fix |
| Static docs | ADRs in wikis, disconnected | Executable architecture rules |
| Enterprise-only | SonarQube (paid for full features) | Free, open-source (MPL-2.0) |

---

## Architecture

Arx itself follows Hexagonal Architecture:

```
arx/
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
