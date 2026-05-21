# arx Quickstart — 5 Minutes

Get from zero to `arx check` in under five minutes. No assumptions.

## Install

Choose one:

```bash
# Via curl (recommended)
curl -sfL https://raw.githubusercontent.com/pauvalls/arx/master/install.sh | sh

# Via Homebrew
brew install pauvalls/tap/arx

# Via Go
go install github.com/pauvalls/arx/cmd/arx@latest
```

Verify it worked:

```bash
arx --version
```

## Initialize

```bash
cd my-project
arx init
```

This scans your project structure, detects languages, and generates an `arx.yaml` configuration file with sensible defaults:

- **Layers** — detected from your directory structure (domain, application, infrastructure, interfaces)
- **Rules** — default Clean Architecture rules (domain cannot depend on infrastructure, etc.)
- **Excludes** — vendor, node_modules, .git, and common build directories
- **Language overrides** — Go and TypeScript import detection

## Check

```bash
arx check
```

You'll see:
- ✓ **No violations** — your architecture is clean
- ✗ **Violations found** — a list of files breaking the rules
- Each violation shows: file, line, rule ID, severity, and import path

## Understand violations

```bash
arx explain D-01
```

This gives you:
- **Code context** — lines around the violation
- **Why it matters** — architectural explanation
- **How to fix** — step-by-step guidance
- **Code example** — before/after for common patterns
- **Auto-fix suggestion** — if available

## Baseline existing projects

If you're adding arx to an existing codebase with known violations:

```bash
arx baseline
```

This captures current violations into `.arx-baseline.json`. Future `arx check` runs only report **new** violations — letting you adopt arx incrementally.

## Next steps

| Resource | Description |
|----------|-------------|
| [Conceptual guides](guides/layers-and-rules.md) | Understand layers, rules, detectors, and the DSL |
| [CLI reference](reference/cli.md) | Every command, flag, and exit code |
| [Config reference](reference/config.md) | Every field in arx.yaml documented |
| [Tutorials](tutorials/ci-cd.md) | CI/CD, monorepos, custom plugins, GitHub App |
| [FAQ](faq.md) | Common questions and troubleshooting |
