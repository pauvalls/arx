# GitHub Action

Run arx in CI/CD with one line.

## Quick Start

Add this to `.github/workflows/arx-ci.yml`:

```yaml
name: Architecture Audit
on: [push, pull_request]

jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/arx-action
```

This will:
- Run `arx check --ci` on your project
- Upload SARIF results to GitHub Code Scanning
- Upload architecture diagrams as artifacts

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `path` | `.` | Project root directory |
| `config` | `arx.yaml` | Config file path |
| `format` | `sarif` | Output format (`sarif`, `json`, `terminal`) |
| `baseline` | `.arx-baseline.json` | Baseline file path |
| `diagram` | `true` | Generate dependency diagram |

## Example with All Options

```yaml
- uses: ./.github/actions/arx-action
  with:
    path: ./my-service
    config: ./my-service/arx.yaml
    format: sarif
    baseline: .arx-baseline.json
    diagram: true
```

## SARIF Integration

The action uploads SARIF results to GitHub Code Scanning,
which shows violations directly in the **Security** tab and on pull request diffs:

```
Security → Code scanning → Arx Architecture Audit
```

## Manual CI Workflow (without the action)

```yaml
name: Architecture Audit
on: [push, pull_request]

jobs:
  arx:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

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

## GitLab CI

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

## Exit Codes in CI

| Scenario | Exit Code | Meaning |
|----------|-----------|---------|
| No violations | 0 | Architecture is clean |
| Only suppressed violations | 0 | Baseline/overrides match |
| New violations found | 1 | Fails CI pipeline |
| Config error | 1 | Invalid config |
