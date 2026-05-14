# Arx

[![Go Report Card](https://goreportcard.com/badge/github.com/pauvalls/arx)](https://goreportcard.com/report/github.com/pauvalls/arx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pauvalls/arx)](go.mod)

**Architecture audit CLI for cross-language codebases.**

Arx validates architectural rules against real codebases and explains *why* violations matter and *how* to fix them. It's not a linter — it's an **architecture guard with a teaching soul**.

## Quickstart

### Installation

```bash
# From source (requires Go 1.21+)
go install github.com/pauvalls/arx/cmd/arx@latest

# Or clone and build
git clone https://github.com/pauvalls/arx.git
cd arx
go build ./cmd/arx
```

### Basic Usage

```bash
# Initialize arx config in your project
arx init

# Check architecture against defined rules
arx check

# Run in CI mode (JSON output)
arx check --ci

# Get detailed audit report
arx audit
```

## Features (MVP Scope)

- **Multi-language support**: Go and TypeScript detectors in MVP, extensible plugin system
- **Layer-based rules**: Define architecture layers (domain, application, infrastructure) and enforce dependencies
- **Didactic violations**: Every violation explains the architectural principle and suggests a fix
- **Multiple output formats**: Terminal (colored), JSON (CI), Markdown (documentation)
- **YAML configuration**: Version-control your architecture rules alongside code
- **Hexagonal Architecture**: Clean separation of concerns in Arx itself

## Example Configuration

```yaml
# .arx.yaml
layers:
  - name: domain
    path: src/domain/**
  - name: application
    path: src/application/**
  - name: infrastructure
    path: src/infrastructure/**

rules:
  - domain CANNOT depend on infrastructure
  - domain CANNOT depend on application
  - application MUST depend on domain
```

## Roadmap

- [x] **Phase 1**: Project setup and structure
- [ ] **Phase 2**: Domain entities and port interfaces
- [ ] **Phase 3**: Detector implementations (Go, TypeScript)
- [ ] **Phase 4**: CLI commands (init, check, audit)
- [ ] **Phase 5**: Output formatters and CI integration
- [ ] **Phase 6**: Plugin system and community detectors

## Why Arx?

Architecture erodes invisibly. No CI check catches when a domain module imports a database driver. By the time someone notices, the damage is structural.

Existing tools are language-bound (ArchUnit for Java, Deptrac for PHP) or ignore architecture entirely (ESLint, SonarQube). Arx fills this gap with a unified, didactic approach.

## License

MIT — see [LICENSE](LICENSE) for details.
