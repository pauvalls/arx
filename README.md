# Arx

[![Go Report Card](https://goreportcard.com/badge/github.com/pauvalls/arx)](https://goreportcard.com/report/github.com/pauvalls/arx)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pauvalls/arx)](go.mod)
[![Release](https://img.shields.io/github/v/release/pauvalls/arx)](https://github.com/pauvalls/arx/releases)

**Architecture audit CLI for cross-language codebases.**  
Validates architectural rules against real code across 10 languages.  
Not a linter — an **architecture guard with a teaching soul**.

## Quickstart

```bash
# Install (Go)
go install github.com/pauvalls/arx/cmd/arx@latest

# Or via Homebrew
brew install pauvalls/arx/arx

# Or use Docker
docker pull ghcr.io/pauvalls/arx:latest

# Initialize config
arx init

# Run audit
arx check

# Create baseline for existing projects (suppress known violations)
arx baseline
```

## Documentation

| Topic | Description |
|-------|-------------|
| **[Configuration](docs/configuration.md)** | arx.yaml reference — layers, rules, overrides, excludes, threshold |
| **[Baseline](docs/baseline.md)** | Suppress existing violations for incremental adoption |
| **[Diff](docs/diff.md)** | Compare architecture between git refs |
| **[Watch Mode](docs/watch-mode.md)** | Continuous feedback on file changes (`arx check --watch`) |
| **[Pre-commit Hook](docs/pre-commit-hook.md)** | Block violations before they're committed |
| **[GitHub Action](docs/github-action.md)** | CI/CD integration with SARIF + Code Scanning |
| **[Output Formats](docs/output-formats.md)** | Terminal, JSON, SARIF, HTML, JUnit, Markdown, GitHub Annotations |
| **[Presets](docs/presets/README.md)** | Clean, Hexagonal, DDD architecture templates |
| **[Diagrams](docs/diagrams/README.md)** | Architecture diagrams (ASCII, DOT, Mermaid) |
| **[Installation](docs/installation.md)** | Install from source, brew, or Docker |
| **[Roadmap](docs/roadmap.md)** | Full release history v0.1.0 → v0.32.0 |

## Commands

| Command | Description |
|---------|-------------|
| `arx init [path]` | Initialize arx.yaml config (supports `--preset`) |
| `arx check [path]` | Run architecture audit (supports `--watch`, `--format`, `--no-cache`) |
| `arx audit [path]` | Full health report with coupling matrix, debt, trends |
| `arx baseline [path]` | Suppress existing violations for incremental CI adoption |
| `arx diff [ref-before] [ref-after]` | Compare architecture between git refs |
| `arx diagram [path]` | Render architecture diagrams (ASCII, DOT, Mermaid) |
| `arx explain <id>` | Detailed violation guidance with fix examples |
| `arx config validate [path]` | Validate arx.yaml independently |
| `arx doctor [path]` | Diagnostics: project health, detectors, config, git |
| `arx completion <shell>` | Generate shell completion (bash/zsh/fish/powershell) |
| `arx hook install\|uninstall` | Install/remove git pre-commit hook |
| `arx man` | Generate man pages for Linux distributions |

## Supported Languages

| Language | Detection | Method | Since |
|----------|-----------|--------|-------|
| Go | `go.mod` | AST parsing | v0.1.0 |
| TypeScript | `tsconfig.json` | Regex pattern matching | v0.1.0 |
| Python | `requirements.txt` / `setup.py` | AST parsing | v0.4.0 |
| Java | `pom.xml` / `build.gradle` | Regex (`package` + `import`) | v0.6.0 |
| Kotlin | `build.gradle.kts` | Regex (package + alias) | v0.8.0 |
| Rust | `Cargo.toml` | Regex (`use` statements) | v0.9.0 |
| C# | `.csproj` / `.sln` | Regex (`using` directives) | v0.10.0 |
| Ruby | `Gemfile` | Regex (`require` / `require_relative`) | v0.13.0 |
| PHP | `composer.json` | Regex (`use` / `use as` / `use function`) | v0.14.0 |
| Swift | `Package.swift` | Regex (`import` / `@_exported import`) | v0.14.0 |

## Quick Config

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
    exclude: ["generated/**", "vendor/**"]    # Per-rule excludes
    overrides:
      - path: legacy/
        severity: warning                     # Per-path overrides

max_violations: 10                             # CI failure threshold
```

## Why Arx?

| Problem | Traditional Tools | Arx |
|---------|------------------|-----|
| Language lock-in | ArchUnit (Java), Deptrac (PHP) | **10 languages** (Go, TS, Python, Java, Kotlin, Rust, C#, Ruby, PHP, Swift) |
| Silent violations | Linters only flag style | **Fails CI** on architectural violations |
| No teaching | "Remove this dependency" | Explains **WHY** + **HOW** to fix |
| Legacy code | Manual exception lists | **Baseline** + **overrides** + **excludes** |
| Enterprise-only | SonarQube (paid) | **Free**, open-source (MPL-2.0) |

## Quick Start for Contributors

```bash
git clone https://github.com/pauvalls/arx.git && cd arx
make build
make test
./arx check  # Dogfooding — should pass with 0 violations
```

## License

[Mozilla Public License 2.0](LICENSE) — weak copyleft, business-friendly.
