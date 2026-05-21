# CLI Reference

## arx

Root command for the arx architecture audit CLI.

```
Usage: arx [command]
```

### Available Commands

| Command | Description |
|---------|-------------|
| [check](#arx-check) | Run architecture audit on a project |
| [audit](#arx-audit) | Full health report with coupling matrix, debt, trends |
| [explain](#arx-explain) | Show detailed explanation for a violation |
| [suggest](#arx-suggest) | Show and apply fix suggestions |
| [init](#arx-init) | Initialize arx configuration for a project |
| [config](#arx-config) | Manage arx configuration |
| [baseline](#arx-baseline) | Create a baseline of current violations |
| [workspace](#arx-workspace) | Run architecture audit across workspace projects |
| [diff](#arx-diff) | Compare architecture between git refs |
| [server](#arx-server) | Start web server with interactive dashboard |
| [lsp](#arx-lsp) | Start an LSP server for real-time diagnostics |
| [pr-check](#arx-pr-check) | Run architecture check on PR changes |
| [diagram](#arx-diagram) | Generate architecture dependency diagram |
| [doctor](#arx-doctor) | Run diagnostics on arx project |
| [test](#arx-test) | Run architecture rule tests |
| [fmt](#arx-fmt) | Format arx.yaml configuration |
| [schema](#arx-schema) | Manage JSON Schema for arx configuration |
| [rollback](#arx-rollback) | Restore files from backup |
| [hook](#arx-hook) | Manage git pre-commit hooks |
| [skill](#arx-skill) | Manage AI coding assistant integrations |
| [completion](#arx-completion) | Generate shell completion scripts |
| [man](#arx-man) | Generate man pages |
| [help](#arx-help) | Help about any command |

### Global Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--version` | | | Print version information |
| `--help` | `-h` | | Print help message |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Violations found or error occurred |

---

## arx check

Run architecture audit on a project.

```
Usage: arx check [path]
```

### Description

Loads the configuration, detects dependencies, evaluates rules, and reports violations. If no path is provided, the current directory is used.

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--config` | `-c` | `arx.yaml` | Config file path |
| `--ci` | | `false` | Machine-readable JSON output for CI/CD |
| `--format` | `-f` | `terminal` | Output format: `terminal`, `json`, `sarif`, `md`, `junit`, `annotations`, `html` |
| `--verbose` | `-v` | `false` | Show detailed dependency information |
| `--no-cache` | | `false` | Disable the performance cache |
| `--no-baseline` | | `false` | Ignore baseline file and show all violations |
| `--watch` | | `false` | Watch mode: re-run check on file changes |
| `--interval` | | `500ms` | Debounce interval for watch mode |
| `--severity` | | `""` | Filter by severity: `error`, `warning`, `info` |
| `--diff` | | `false` | Show violations added/removed since last check |
| `--profile` | | `false` | Show per-detector performance profile |

### Examples

```bash
# Basic check
arx check

# Check specific directory
arx check ./my-project

# CI-friendly JSON output
arx check --ci

# HTML report
arx check --format html

# Watch mode (re-runs on file changes)
arx check --watch

# Filter by severity
arx check --severity error

# Show performance profile
arx check --profile

# Ignore baseline, show all violations
arx check --no-baseline

# Show diff since last baseline snapshot
arx check --diff
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No violations found (or only info/warnings with `--ci`) |
| 1 | Violations found or error occurred |

When a baseline exists (`.arx-baseline.json`):
- 0: No NEW violations (baseline violations suppressed)
- 1: New violations found

---

## arx audit

Run a comprehensive architecture audit with health metrics and trends.

```
Usage: arx audit [path]
```

### Description

Includes violations, coupling matrix, technical debt score, and trend comparison with previous audits.

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `-o` | `""` | Output file path (default: stdout) |
| `--format` | `-f` | `terminal` | Output format: `terminal`, `json`, `html` |
| `--trend` | | `false` | Show only trend comparison with previous audit |
| `--since` | | `""` | Show trends since date (format: YYYY-MM-DD) |

### Examples

```bash
arx audit
arx audit --format json
arx audit --format html -o report.html
arx audit --trend
arx audit --since 2026-04-01
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No violations found |
| 1 | Violations found or error occurred |

---

## arx explain

Show detailed explanation for a specific violation.

```
Usage: arx explain [violation-id]
```

### Description

Provides comprehensive guidance: code context, architectural context, step-by-step refactoring, code examples, and auto-fix suggestions.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--list` | `false` | List all cached violations |
| `--last` | `false` | Show most recent violation |
| `--suggest` | `""` | Show fix suggestion for a specific rule |

### Examples

```bash
arx explain D-01         # Explain specific violation
arx explain              # Show most recent violation
arx explain --last       # Alias for most recent
arx explain --list       # List all cached violations
arx explain --suggest D-01  # Show fix for a rule
```

---

## arx suggest

Show and apply fix suggestions for architecture violations.

```
Usage: arx suggest [violation-id]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--apply` | `false` | Apply the suggested fixes to files |
| `--force` | `false` | Skip confirmation when using `--apply` |
| `--output` | `-o` | `""` | Write diffs to file instead of stdout |
| `--all` | `false` | Collect all fixes with conflict detection and interactive review |
| `--dry-run` | `false` | Show all fixes without applying |

### Examples

```bash
arx suggest                     # Show fixes for all violations
arx suggest D-01                # Show fix for violation D-01
arx suggest --apply             # Apply fixes with confirmation
arx suggest --all               # Interactive batch apply
arx suggest --dry-run           # Preview all fixes
arx suggest --apply --force     # Apply without confirmation
```

---

## arx init

Initialize arx configuration for a project.

```
Usage: arx init [path]
```

### Description

Scans the directory structure, detects languages, and generates an `arx.yaml` with sensible defaults. Updates `.gitignore` with arx-related entries.

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `-o` | `arx.yaml` | Output file path for generated config |
| `--force` | `-f` | `false` | Overwrite existing configuration |
| `--preset` | `-p` | `""` | Use preset template: `clean`, `hexagonal`, `ddd`, `layered`, `onion` |
| `--detect` | `-d` | `false` | Dry-run scan: show detected layers without writing |

### Examples

```bash
arx init                          # Auto-detect and generate
arx init ./my-project             # Specific directory
arx init --preset hexagonal       # Use hexagonal architecture preset
arx init --detect                 # Preview only, no file written
arx init --output config/arx.yaml # Custom output path
arx init --force                  # Overwrite existing
```

---

## arx config

Manage arx configuration.

```
Usage: arx config [subcommand]
```

### Subcommands

| Command | Description |
|---------|-------------|
| `arx config get` | Read a field from arx.yaml |
| `arx config set` | Update a field in arx.yaml |
| `arx config validate` | Validate arx.yaml |

### arx config get

```
Usage: arx config get <field>
```

Get a configuration value by key. Supports dotted paths.

```bash
arx config get version
arx config get severity_mapping.critical
arx config get layers
```

### arx config set

```
Usage: arx config set <field> <value>
```

Set a configuration value. Supports dotted paths and JSON array values.

```bash
arx config set version "1.0"
arx config set severity_mapping.critical "error"
arx config set exclude '["vendor/**", "node_modules/**"]'
```

### arx config validate

```
Usage: arx config validate [path]
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--path` | `-p` | `""` | Path to config file (default: arx.yaml) |
| `--strict` | `-s` | `false` | Fail on unknown config keys |
| `--schema` | | `false` | Show JSON Schema reference for config |
| `--override` | | `""` | Path to override YAML config (deep-merged into base) |

### Examples

```bash
arx config validate                     # Validate arx.yaml
arx config validate --strict            # Also check for unknown keys
arx config validate --schema            # Print JSON Schema
arx config validate --override custom.yaml
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Config is valid |
| 1 | Config is invalid or error occurred |

---

## arx baseline

Create a baseline of current violations for incremental CI adoption.

```
Usage: arx baseline [path]
```

### Description

Captures current violations to `.arx-baseline.json`. Future `arx check` runs suppress baselined violations — only new ones are reported.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--reset` | `false` | Overwrite existing baseline |
| `--output` | `-o` | `""` | Custom output path for baseline file |
| `--diff` | `false` | Compare current violations against last snapshot |
| `--history` | `false` | Show baseline history trend table |
| `--refresh-threshold` | `3` | Consecutive clean checks before auto-refresh |

### Examples

```bash
arx baseline                    # Create baseline
arx baseline --reset            # Overwrite existing baseline
arx baseline --diff             # Show diff since last snapshot
arx baseline --history          # Show baseline trend
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Baseline created successfully |
| 1 | Error occurred |

---

## arx workspace

Run architecture audit across multiple projects in a workspace.

```
Usage: arx workspace [path]
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--json` | `-j` | `false` | Output JSON report to stdout |
| `--verbose` | `-v` | `false` | Show detailed per-project breakdown |
| `--output` | `-o` | `""` | Write report to file |

### Examples

```bash
arx workspace                    # Check all workspace projects
arx workspace ./monorepo         # Specify workspace root
arx workspace --json             # JSON output
arx workspace --verbose          # Per-project details
arx workspace --output report.json
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All projects pass (no violations) |
| 1 | Any project has violations or errors |

---

## arx server

Start an HTTP server with an interactive dashboard and REST API.

```
Usage: arx server
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--port` | `-p` | `8080` | Server port |
| `--bind` | | `127.0.0.1` | Bind address |
| `--path` | `-d` | `.` | Project root path |

### Examples

```bash
arx server                    # Start on localhost:8080
arx server --port 3000        # Port 3000
arx server --bind 0.0.0.0     # All interfaces
arx server -d ./my-project    # Specific project
```

---

## arx lsp

Start a Language Server Protocol server for real-time architecture diagnostics.

```
Usage: arx lsp
```

### Description

Communicates over stdin/stdout using JSON-RPC 2.0 with Content-Length headers. Compatible with VS Code, Neovim, Helix, Zed, and any LSP-compatible editor.

### Examples

```bash
arx lsp
```

---

## arx pr-check

Run an architecture check scoped to the changes introduced by a pull request.

```
Usage: arx pr-check
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--base` | `""` | Base ref (e.g., `main`, `HEAD~1`) |
| `--head` | `""` | Head ref (e.g., `feature/branch`, `HEAD`) |
| `--repo` | `-r` | `.` | Project root path |
| `--json` | `-j` | `false` | Output in JSON format |
| `--verbose` | `-v` | `false` | Show detailed information |
| `--approve` | `false` | Auto-approve PR via GitHub API when no violations |

### Examples

```bash
arx pr-check --base HEAD~1 --head HEAD
arx pr-check --base origin/main --head feature/branch --json
arx pr-check --base main --head feature --repo /path/to/repo
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No new violations introduced |
| 1 | New violations found |

---

## arx diagram

Generate an architecture dependency diagram.

```
Usage: arx diagram [path]
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--format` | `-f` | `ascii` | Output format: `ascii`, `dot`, `mermaid` |
| `--output` | `-o` | `""` | Output file path (default: stdout) |

### Examples

```bash
arx diagram                           # ASCII diagram
arx diagram --format mermaid          # Mermaid diagram
arx diagram --format dot -o deps.dot  # DOT to file
arx diagram ./my-project --format mermaid
```

---

## arx doctor

Run diagnostics on arx project health.

```
Usage: arx doctor [path]
```

### Checks

1. Project root exists and is accessible
2. Config file (arx.yaml) exists and is valid
3. Language detectors can find source files
4. Git repository status
5. Arx version information

### Examples

```bash
arx doctor
arx doctor ./my-project
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All critical checks passed |
| 1 | One or more critical checks failed |

---

## arx test

Run architecture rule tests defined in YAML files.

```
Usage: arx test [path]
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--fixture` | | `""` | Override fixture path for all test cases |
| `--rule` | | `""` | Filter tests by rule ID (glob) |
| `--verbose` | `-v` | `false` | Show detailed match info |
| `--ci` | | `false` | CI mode: exit code reflects pass/fail |
| `--junit` | | `""` | Write JUnit XML to file |

### Examples

```bash
arx test                           # Run all tests
arx test tests/                    # Run tests in directory
arx test tests/my_test.yaml        # Run specific test file
arx test --ci                      # CI mode
arx test --junit results.xml       # JUnit output
arx test --verbose                 # Detailed info
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All tests pass |
| 1 | Some tests fail |
| 2 | Internal error |

---

## arx fmt

Format arx.yaml configuration file.

```
Usage: arx fmt [path]
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--check` | `-c` | `false` | Check if file is formatted (exit 1 if not) |

### Examples

```bash
arx fmt                    # Format arx.yaml
arx fmt ./config/arx.yaml  # Specific file
arx fmt --check            # Check format only
```

---

## arx schema

Manage JSON Schema for arx configuration.

```
Usage: arx schema [subcommand]
```

### Subcommands

| Command | Description |
|---------|-------------|
| `arx schema generate` | Generate JSON Schema for arx configuration |

### arx schema generate

Generate a JSON Schema document for IDE autocompletion.

```
Usage: arx schema generate
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `-o` | `""` | Write schema to file instead of stdout |
| `--pretty` | | `false` | Pretty-print the schema (auto-detected) |
| `--minified` | | `false` | Minified output (no whitespace) |

### Examples

```bash
arx schema generate
arx schema generate --output arx-schema.json
arx schema generate --minified
```

---

## arx rollback

Restore files that were backed up during `arx suggest --apply`.

```
Usage: arx rollback [file]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--list` | `false` | Show available backups |
| `--all` | `false` | Restore all backed-up files |
| `--clean` | `false` | Remove orphaned backup directories |

### Examples

```bash
arx rollback test.go         # Restore single file
arx rollback --list          # Show available backups
arx rollback --all           # Restore everything
arx rollback --clean         # Clean orphaned backups
```

---

## arx hook

Manage git pre-commit hooks for architecture validation.

```
Usage: arx hook [subcommand]
```

### Subcommands

| Command | Description |
|---------|-------------|
| `arx hook install` | Install the pre-commit hook |
| `arx hook uninstall` | Remove the pre-commit hook |

### arx hook install

Creates `.git/hooks/pre-commit` that runs `arx check --no-cache` before each commit.

```bash
arx hook install
arx hook install ./my-project
```

### arx hook uninstall

Removes the pre-commit hook.

```bash
arx hook uninstall
```

### Bypass

Skip the hook for a single commit:

```bash
SKIP=arx git commit -m "..."
```

---

## arx skill

Manage AI coding assistant integrations.

```
Usage: arx skill [subcommand]
```

### Subcommands

| Command | Description |
|---------|-------------|
| `arx skill install` | Install arx-setup skill to AI coding assistants |

### arx skill install

Install the arx-setup skill to opencode, Claude Code, Cursor, or GitHub Copilot.

```
Usage: arx skill install [tool...]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Install to all detected tools without prompting |

### Examples

```bash
arx skill install                 # Interactive: select from detected tools
arx skill install opencode        # Install to opencode only
arx skill install opencode claude # Install to multiple tools
arx skill install --all           # Install to all detected
```

---

## arx completion

Generate shell completion scripts.

```
Usage: arx completion [shell]
```

### Supported Shells

| Shell | Command |
|-------|---------|
| bash | `arx completion bash` |
| zsh | `arx completion zsh` |
| fish | `arx completion fish` |
| powershell | `arx completion powershell` |

### Examples

```bash
# Bash
arx completion bash > /etc/bash_completion.d/arx

# Zsh
arx completion zsh > "${fpath[1]}/_arx"

# Fish
arx completion fish > ~/.config/fish/completions/arx.fish

# PowerShell
arx completion powershell | Out-String | Invoke-Expression
```

---

## arx man

Generate man pages for all arx commands.

```
Usage: arx man
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--output` | `-o` | `docs/man` | Output directory for man pages |

### Examples

```bash
arx man
arx man --output /usr/local/share/man/man1/
```

---

## arx help

Display help for any command.

```
Usage: arx help [command]
```

### Examples

```bash
arx help
arx help check
arx help workspace
```
