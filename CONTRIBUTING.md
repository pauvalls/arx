# Contributing to Arx

Thank you for considering contributing to Arx! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to the [Contributor Covenant](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* **Use a clear and descriptive title**
* **Describe the exact steps to reproduce the problem**
* **Provide specific examples to demonstrate the steps**
* **Describe the behavior you observed and what behavior you expected**
* **Include Arx version and output from `arx check --ci`**

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* **Use a clear and descriptive title**
* **Provide a detailed description of the suggested enhancement**
* **Explain why this enhancement would be useful**
* **List some examples of how this enhancement would be used**

### Pull Requests

* Fill in the required template
* Follow the Go style guide
* Include tests that cover all new code paths
* Ensure all existing tests pass
* Update documentation if needed
* Squash commits into logical units

## Development Setup

### Prerequisites

* Go 1.21 or later
* Git
* A code editor (we recommend VS Code or Neovim)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/pauvalls/arx.git
cd arx

# Build the binary
go build ./cmd/arx

# Run tests
go test ./...

# Run Arx on itself (dogfooding)
./arx check
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/domain/...
go test ./internal/application/...

# Run integration tests
go test ./test/integration/...
```

## Architecture

Arx follows Hexagonal Architecture. Understanding this is crucial for contributions:

```
Domain Layer (internal/domain/)
    ↓ Pure business logic, no I/O
    ↓ Entities: Layer, Rule, Violation, Dependency
    ↓ Audit service: rule evaluation logic

Ports Layer (internal/ports/)
    ↓ Interfaces: Detector, ConfigReader, Reporter, FileWriter
    ↓ No implementations, only contracts

Application Layer (internal/application/)
    ↓ Use cases: Init, Check
    ↓ Orchestrates domain + ports
    ↓ Built-in explanations library

Infrastructure Layer (internal/infrastructure/)
    ↓ Concrete implementations of ports
    ↓ Detectors: Go (AST), TypeScript (regex)
    ↓ Reporters: Terminal (lipgloss), JSON
    ↓ Config: YAML parser (viper)

CLI Layer (cmd/arx/)
    ↓ Cobra commands: init, check, explain
    ↓ Wires up application use cases
```

### Key Design Principles

1. **Domain Purity**: Domain layer must not import application, infrastructure, or ports
2. **Dependency Injection**: All I/O goes through port interfaces
3. **Educational Output**: Every violation explains "why" and "how to fix"
4. **Cross-Language**: Detectors are pluggable; MVP supports Go + TypeScript

## Writing a Detector

Detectors are the primary extension point for Arx. Here's how to add support for a new language:

### Step 1: Create Detector Structure

```go
// internal/infrastructure/detector/python/detector.go
package python

import (
    "context"
    "github.com/pauvalls/arx/internal/domain"
    "github.com/pauvalls/arx/internal/ports"
)

type Detector struct{}

func New() *Detector {
    return &Detector{}
}

func (d *Detector) Name() string {
    return "python"
}

func (d *Detector) Detect(ctx context.Context, projectRoot string) (bool, error) {
    // Check for Python project markers (requirements.txt, setup.py, pyproject.toml)
    // Return true if this detector applies
}

func (d *Detector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
    // Parse Python files, extract import statements
    // Resolve imports to layers
    // Return []domain.Dependency
}
```

### Step 2: Implement Import Extraction

For Python, you can use the `ast` module or regex:

```go
// Simple regex approach (MVP style)
importPatterns := []string{
    `^import\s+(\w+)`,
    `^from\s+(\w+)\s+import`,
    `^from\s+([\w.]+)\s+import`,
}

// For each .py file:
// 1. Match import patterns
// 2. Resolve module path to layer
// 3. Create domain.Dependency
```

### Step 3: Register Detector

```go
// internal/infrastructure/detector/registry.go
func GetDetectors() []ports.Detector {
    return []ports.Detector{
        go_detector.New(),
        ts_detector.New(),
        python_detector.New(), // Add your detector
    }
}
```

### Step 4: Write Tests

```go
// test/fixtures/python-project/
// Create a sample Python project with:
// - Clean architecture (no violations)
// - Intentional violations (domain importing infrastructure)

// test/integration/python_detector_test.go
func TestPythonDetector_ExtractImports(t *testing.T) {
    // Test import extraction from sample Python files
}

func TestPythonDetector_DetectViolations(t *testing.T) {
    // Test that violations are correctly detected
}
```

### Step 5: Add Documentation

Update README.md supported languages table:

```markdown
| Python | `ast` module | AST parsing | ✅ v0.3.0 |
```

## Adding Explanation Patterns

Arx includes built-in explanations for common architectural rules. To add a new pattern:

### Step 1: Add to Explanations Library

```go
// internal/application/explanations.go
var builtinExplanations = map[string]Explanation{
    "python-*": {
        Why: "Python-specific architectural principle...",
        Fix: []string{
            "Step 1...",
            "Step 2...",
            "Step 3...",
        },
    },
}
```

### Step 2: Add Tests

```go
// internal/application/explanations_test.go
func TestGetExplanation_PythonPatterns(t *testing.T) {
    // Test that Python rule patterns return correct explanations
}
```

## Code Style

### Go Conventions

* Use `gofmt` or `goimports` for formatting
* Follow [Effective Go](https://golang.org/doc/effective_go.html)
* Use meaningful variable names
* Keep functions small and focused
* Comment exported functions and types
* Write table-driven tests

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add Python detector
fix: resolve circular import in rule evaluation
docs: update README with SARIF output example
test: add integration tests for TypeScript detector
refactor: extract violation formatting to separate function
```

### Testing Guidelines

* **Unit tests**: Test domain logic in isolation
* **Integration tests**: Test detectors with fixture projects
* **Golden files**: Snapshot expected output for comparison
* **Coverage**: Aim for 80%+ on domain and application layers

```bash
# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Dogfooding

Arx must pass its own architecture rules:

```bash
# This MUST pass with 0 violations
./arx check

# If you add new code, verify it doesn't violate architecture
git add .
./arx check
```

## Release Process

Releases follow semantic versioning (MAJOR.MINOR.PATCH):

1. Update version in `cmd/arx/version.go`
2. Update CHANGELOG.md
3. Update README roadmap
4. Create tag: `git tag -a v0.2.0 -m "Release v0.2.0"`
5. Push tag: `git push origin v0.2.0`
6. Create GitHub release with changelog

## Questions?

* Open an issue for general questions
* Join discussions in existing issues
* Check the [FAQ](docs/faq.md) (coming soon)

---

Thank you for contributing to Arx! 🎉
