# CI/CD Integration

## GitHub Actions

Create `.github/workflows/arx.yml` in your repository:

```yaml
name: Architecture Audit

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Install arx
        run: go install github.com/pauvalls/arx/cmd/arx@latest

      - name: Init config (if not present)
        run: |
          if [ ! -f arx.yaml ]; then
            arx init
          fi
          arx config validate

      - name: Run architecture audit
        run: arx check --ci
```

### With Baseline

For existing codebases with known violations:

```yaml
name: Architecture Audit

on:
  push:
    branches: [main]
  pull_request:

jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Install arx
        run: go install github.com/pauvalls/arx/cmd/arx@latest

      - name: Initialize baseline (first run only)
        run: |
          arx init --force
          arx baseline --reset
        if: github.event_name == 'push' && github.ref == 'refs/heads/main'

      - name: Run audit with baseline
        run: arx check --ci
```

### With SARIF and Code Scanning

Upload results to GitHub Code Scanning:

```yaml
name: Architecture Audit

on:
  push:
    branches: [main]
  pull_request:

jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Install arx
        run: go install github.com/pauvalls/arx/cmd/arx@latest

      - name: Run architecture audit
        run: arx check --format sarif --output results.sarif

      - name: Upload SARIF to GitHub
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: results.sarif
          category: arx
```

### With GitHub Annotations

Inline annotations on PR diffs:

```yaml
- name: Run with annotations
  run: arx check --format annotations
```

### Using the GitHub Action (arx-action)

```yaml
jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pauvalls/arx-action@v1
```

See the [GitHub Action docs](../github-action.md) for configuration options.

---

## GitLab CI

Create `.gitlab-ci.yml`:

```yaml
architecture-audit:
  stage: test
  image: golang:1.25
  script:
    - go install github.com/pauvalls/arx/cmd/arx@latest
    - if [ ! -f arx.yaml ]; then arx init; fi
    - arx check --format json --output arx-report.json
  artifacts:
    paths:
      - arx-report.json
    when: always
```

### With baseline strategy for GitLab

```yaml
architecture-audit:
  stage: test
  image: golang:1.25
  script:
    - go install github.com/pauvalls/arx/cmd/arx@latest
    
    # Restore baseline from cache (if exists)
    - if [ -f .arx-baseline-cache/baseline.json ]; then
    -   cp .arx-baseline-cache/baseline.json .arx-baseline.json
    - fi
    
    # Initialize if needed
    - if [ ! -f arx.yaml ]; then arx init; fi
    
    # Run audit
    - arx check --format json
    
    # Save baseline for next run
    - mkdir -p .arx-baseline-cache
    - if [ -f .arx-baseline.json ]; then
    -   cp .arx-baseline.json .arx-baseline-cache/baseline.json
    - fi
    
  cache:
    paths:
      - .arx-baseline-cache/
  artifacts:
    reports:
      junit: arx-test-results.xml
```

### With workspace mode for monorepo

```yaml
architecture-audit:
  stage: test
  image: golang:1.25
  script:
    - go install github.com/pauvalls/arx/cmd/arx@latest
    - arx workspace --json --output workspace-report.json
  artifacts:
    paths:
      - workspace-report.json
```

---

## Pre-commit Hook

For local development, install the pre-commit hook:

```bash
arx hook install
```

This adds a `.git/hooks/pre-commit` script that runs `arx check --no-cache` before every commit. Only NEW violations (not baselined ones) block the commit.

Bypass for a single commit:

```bash
SKIP=arx git commit -m "WIP: temporary"
```
