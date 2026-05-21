# Arx

[![Go Report Card](https://goreportcard.com/badge/github.com/pauvalls/arx)](https://goreportcard.com/report/github.com/pauvalls/arx)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pauvalls/arx)](go.mod)
[![Release](https://img.shields.io/github/v/release/pauvalls/arx)](https://github.com/pauvalls/arx/releases)
![Build Status](https://img.shields.io/github/actions/workflow/status/pauvalls/arx/ci.yml?branch=master)
![Test Coverage](https://img.shields.io/badge/coverage-92%25-green)

**Architecture audit CLI for cross-language codebases.**  
Validates architectural rules against real code across 10 languages.  
Not a linter — an **architecture guard with a teaching soul**.

---

## Install

```bash
# Via curl (recommended)
curl -sfL https://raw.githubusercontent.com/pauvalls/arx/master/install.sh | sh

# Via Homebrew
brew install pauvalls/tap/arx

# Via Go
go install github.com/pauvalls/arx/cmd/arx@latest
```

## Quick Example

```bash
cd my-project
arx init          # Scans project, generates arx.yaml
arx check         # Detects dependencies, evaluates rules, reports violations
arx explain D-01  # Explains WHY a violation matters and HOW to fix it
```

New to arx? Start with the [Quickstart — 5 minutes](docs/quickstart.md).

---

## Features

| Feature | Description |
|---------|-------------|
| **10 Languages** | Go, TypeScript, Python, Java, Kotlin, Rust, C#, Ruby, PHP, Swift |
| **Layers & Rules** | Define architectural layers and dependency rules (Cannot/Must/Can/MustNotCircular) |
| **Expression DSL** | Custom rules with logic: `count(deps(domain, infra)) == 0` |
| **WASM Policies** | Write rules in any language compiled to WebAssembly |
| **Plugin System** | External detectors for any language via JSON protocol |
| **Cross-Language** | Proto→Go, OpenAPI→TypeScript, and custom generated-code mappings |
| **Workspace Mode** | Audit monorepos with shared config and per-project overrides |
| **LSP Server** | Real-time diagnostics in VS Code, Neovim, Helix, Zed |
| **PR Checks** | Auto-comment on PRs with new violations (GitHub App) |
| **Web Dashboard** | Real-time SSE dashboard with coupling matrix, debt, trends |
| **HTML Reports** | Full HTML audit reports with coupling matrix and debt score |
| **Baseline** | Suppress existing violations for incremental CI adoption |
| **Auto-Fix Suggestions** | `arx suggest --apply` to fix common violations |
| **Shell Completion** | Bash, Zsh, Fish, PowerShell |
| **CI/CD Ready** | JSON, SARIF, JUnit, GitHub Annotations output formats |
| **Zero CGO** | Pure Go — no platform-specific dependencies |

## Documentation

| Topic | Description |
|-------|-------------|
| **[Quickstart](docs/quickstart.md)** | 5-minute "zero to arx check" guide |
| **[CLI Reference](docs/reference/cli.md)** | Every command, flag, and exit code |
| **[Config Reference](docs/reference/config.md)** | Every field in arx.yaml |
| **[API Reference](docs/reference/api.md)** | REST, SSE, LSP endpoints |
| **Conceptual Guides** | [Layers & Rules](docs/guides/layers-and-rules.md), [Detectors](docs/guides/detectors.md), [Expression DSL](docs/guides/expression-dsl.md), [WASM Policies](docs/guides/wasm-policies.md), [Workspace Mode](docs/guides/workspace-mode.md) |
| **Tutorials** | [CI/CD](docs/tutorials/ci-cd.md), [Workspace Monorepo](docs/tutorials/workspace-monorepo.md), [Custom Plugin](docs/tutorials/custom-plugin.md), [GitHub App](docs/tutorials/github-app.md) |
| **Editor Setup** | [VS Code](docs/editors/vscode.md), [Neovim](docs/editors/neovim.md), [Helix](docs/editors/helix.md), [Zed](docs/editors/zed.md) |
| **[FAQ](docs/faq.md)** | Common questions and troubleshooting |
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
| **[AI Integration](docs/ai-integration.md)** | Install arx-setup skill to AI coding assistants |
| **[Expression Rules](docs/expression-rules.md)** | Expression DSL — builtins, filter/map, user functions |
| **[Cross-Language](docs/cross-language.md)** | Detect proto→generated code, OpenAPI→client |
| **[Suggest & Explain](docs/suggest.md)** | Auto-fix suggestions and violation explanations |
| **[Plugin System](docs/plugins.md)** | External plugin protocol and authoring guide |
| **[Roadmap](docs/roadmap.md)** | Full release history v0.1.0 → v50.0 |

## Commands

| Command | Description |
|---------|-------------|
| `arx init [path]` | Initialize arx.yaml config (supports `--preset`, `--detect`) |
| `arx check [path]` | Run architecture audit (supports `--watch`, `--format`, `--no-cache`, `--profile`) |
| `arx audit [path]` | Full health report with coupling matrix, debt, trends |
| `arx baseline [path]` | Suppress existing violations for incremental CI adoption |
| `arx diff [ref-before] [ref-after]` | Compare architecture between git refs |
| `arx diagram [path]` | Render architecture diagrams (ASCII, DOT, Mermaid) |
| `arx explain <id>` | Detailed violation guidance with fix examples |
| `arx suggest <id>` | Show and apply fix suggestions |
| `arx config validate [path]` | Validate arx.yaml independently |
| `arx config get <key>` | Read a config value (dotted paths) |
| `arx config set <key> <value>` | Set a config value |
| `arx doctor [path]` | Diagnostics: project health, detectors, config, git |
| `arx fmt [path]` | Format arx.yaml |
| `arx test [path]` | Run architecture rule tests |
| `arx workspace [path]` | Multi-project architecture audit |
| `arx server` | Start web dashboard + REST API |
| `arx lsp` | Start LSP server for real-time diagnostics |
| `arx pr-check` | Check PR changes for new violations |
| `arx hook install\|uninstall` | Install/remove git pre-commit hook |
| `arx rollback [file]` | Restore files from backup |
| `arx schema generate` | Generate JSON Schema |
| `arx completion <shell>` | Generate shell completion (bash/zsh/fish/powershell) |
| `arx man` | Generate man pages |
| `arx skill install [tool]` | Install arx-setup skill to AI coding assistants |

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
