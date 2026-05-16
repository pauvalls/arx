# Arx

[![Go Report Card](https://goreportcard.com/badge/github.com/pauvalls/arx)](https://goreportcard.com/report/github.com/pauvalls/arx)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pauvalls/arx)](go.mod)
[![Release](https://img.shields.io/github/v/release/pauvalls/arx)](https://github.com/pauvalls/arx/releases)

**Architecture audit CLI for cross-language codebases.**

Validates architectural rules against real code. Not a linter — an **architecture guard with a teaching soul**.

## Quickstart

```bash
# Install
go install github.com/pauvalls/arx/cmd/arx@latest

# Initialize config
arx init

# Run audit
arx check

# Create baseline for existing projects
arx baseline
```

## Documentation

| Topic | Description |
|-------|-------------|
| **[Installation](docs/installation.md)** | Install, build from source, verify |
| **[Configuration](docs/configuration.md)** | arx.yaml reference — layers, rules, overrides, excludes |
| **[Baseline](docs/baseline.md)** | Suppress existing violations, incremental adoption |
| **[Diff](docs/diff.md)** | Compare architecture between git refs |
| **[Watch Mode](docs/watch-mode.md)** | Continuous feedback on file changes |
| **[Pre-commit Hook](docs/pre-commit-hook.md)** | Block violations before they're committed |
| **[GitHub Action](docs/github-action.md)** | CI/CD integration with SARIF |
| **[Output Formats](docs/output-formats.md)** | Terminal, JSON, SARIF, Markdown |
| **[Presets](docs/presets/README.md)** | Clean, Hexagonal, DDD templates |
| **[Diagrams](docs/diagrams/README.md)** | Dependency visualization |
| **[Roadmap](docs/roadmap.md)** | Past releases and future plans |

## Commands

| Command | Description |
|---------|-------------|
| `arx init [path]` | Initialize config |
| `arx check [path]` | Run architecture audit |
| `arx audit [path]` | Full health report with metrics |
| `arx baseline [path]` | Suppress existing violations |
| `arx diff [ref-before] [ref-after]` | Compare architecture between refs |
| `arx explain <id>` | Detailed violation guidance |
| `arx hook install\|uninstall` | Pre-commit hook management |

## Supported Languages

| Language | Detector | Method | Since |
|----------|----------|--------|-------|
| Go | `go/ast` | AST | v0.1.0 |
| TypeScript | Regex | Pattern matching | v0.1.0 |
| Python | `ast` module | AST | v0.4.0 |
| Java | Regex | `package` + `import` | v0.6.0 |
| Kotlin | Regex | Package + alias | v0.8.0 |
| Rust | Regex | `use` statements | v0.9.0 |
| C# | Regex | `using` directives | v0.10.0 |
| Ruby | Regex | `require` / `require_relative` | v0.13.0 |

## Quick Config Example

```yaml
layers:
  - name: domain
    paths: ["internal/domain/**"]
  - name: application
    paths: ["internal/application/**"]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]

rules:
  - id: domain-purity
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
    overrides:
      - path: legacy/
        severity: warning
```

## Why Arx?

| Problem | Traditional Tools | Arx |
|---------|------------------|-----|
| Language lock-in | ArchUnit (Java), Deptrac (PHP) | Cross-language (Go, TS, Python, Java, Kotlin, Rust, C#, Ruby) |
| Silent violations | Linters only flag style | Fails CI on architectural violations |
| No teaching | "Remove this dependency" | Explains WHY + HOW to fix |
| Enterprise-only | SonarQube (paid) | Free, open-source (MPL-2.0) |

## Quick Start for Contributors

```bash
git clone https://github.com/pauvalls/arx.git && cd arx
go build ./cmd/arx
go test ./...
./arx check  # Dogfooding — should pass with 0 violations
```

## License

[Mozilla Public License 2.0](LICENSE) — weak copyleft, business-friendly.
