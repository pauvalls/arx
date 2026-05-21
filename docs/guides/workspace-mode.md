# Workspace Mode

## What Is Workspace Mode?

Workspace mode lets you run architecture audits across **multiple projects** in a monorepo. Instead of running `arx check` in each directory separately, you define a workspace configuration and arx checks all projects in a single command, producing aggregated reports.

```
monorepo/
├── arx-workspace.yaml        # Workspace config
├── arx.yaml                  # Optional shared config (for plugins, overrides)
├── services/
│   ├── auth/
│   │   └── arx.yaml          # Per-project rules
│   ├── billing/
│   │   └── arx.yaml
│   └── notifications/
│       └── arx.yaml
└── internal/
    └── shared/
        └── arx.yaml
```

## arx-workspace.yaml Format

```yaml
version: "1.0"
projects:
  - path: services/auth
  - path: services/billing
  - path: services/notifications
    override:
      max_violations: 5
  - path: internal/libs/*        # Glob patterns supported
  - path: tools/migrate/*

shared:
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
      to: [infrastructure, interfaces]
      type: Cannot
      severity: error
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Workspace config version (`"1.0"`) |
| `projects` | array | Yes | List of projects to audit |
| `projects[].path` | string | Yes | Path or glob pattern to project |
| `projects[].override` | object | No | Per-project overrides |
| `shared` | object | No | Shared config (layers, rules, exclude) |

## Shared Config + Per-Project Overrides

### Shared Config

The `shared` section defines layers, rules, and excludes that apply to **all projects** by default. This avoids repeating the same config in every project.

```yaml
shared:
  layers:
    - name: domain
      paths: ["internal/domain/**"]
  rules:
    - id: no-infra-in-domain
      from: domain
      to: [infrastructure]
      type: Cannot
      severity: error
  exclude:
    - vendor/**
    - node_modules/**
```

### Per-Project Overrides

Projects can **fully replace** shared layers, rules, or excludes with their own. This is a **shallow merge** — if a project specifies `layers`, it replaces the shared layers entirely, not deep-merges.

```yaml
projects:
  - path: services/auth
    override:
      layers:
        - name: core
          paths: ["core/**"]
      rules:
        - id: custom-rule
          from: core
          to: [infrastructure]
          type: Cannot
          severity: error
      max_violations: 3
```

### Merge Behavior

| Field | Shared → Project |
|-------|-----------------|
| `layers` | Project layers fully replace shared |
| `rules` | Project rules fully replace shared |
| `exclude` | Project excludes fully replace shared |
| `max_violations` | Only from project override; shared has none |

## Full Workspace via arx.yaml

You can also define workspace inline in `arx.yaml` instead of a separate `arx-workspace.yaml`:

```yaml
version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain/**"]

workspace:
  projects:
    - path: services/auth
    - path: services/billing
      override:
        max_violations: 5
```

When both `arx-workspace.yaml` and the `workspace:` field exist, `arx-workspace.yaml` takes precedence.

## Running Workspace Checks

```bash
# Basic run — discovers projects and checks all
arx workspace

# Specify root directory
arx workspace ./monorepo

# JSON output for CI ingestion
arx workspace --json

# Verbose (per-project breakdown)
arx workspace --verbose

# Write report to file
arx workspace --output report.json
```

## Error Isolation

One of the key benefits of workspace mode: **failing projects don't block others**.

```
Project                  Status    Violations
──────────────────────────────────────────────
services/auth            PASS              0
services/billing         FAIL              3
services/notifications   PASS              0
internal/libs/shared     PASS              0
tools/migrate/v1         PASS              0
──────────────────────────────────────────────
4 of 5 projects PASS
```

A build failure in `services/billing` doesn't prevent `services/notifications` from being checked.

## Aggregated Reports

### Terminal Output

```
arx workspace
```

Summarizes all projects in a table with PASS/FAIL status and violation counts.

### JSON Output

```bash
arx workspace --json
```

Produces machine-readable output suitable for CI ingestion:

```json
{
  "summary": {
    "total_projects": 5,
    "passed_projects": 4,
    "failed_projects": 1,
    "total_violations": 3
  },
  "projects": [
    {
      "name": "services/auth",
      "path": "/repo/services/auth",
      "passed": true,
      "violations": []
    }
  ]
}
```

### File Output

```bash
arx workspace --output report.json
```

Writes the JSON report to a file instead of stdout.

## CI Integration

Workspace mode is designed for CI pipelines:

```yaml
# .github/workflows/arx.yml
jobs:
  architecture:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pauvalls/arx-action@v1
      - run: arx workspace --json --output arx-report.json
      - if: failure()
        run: arx workspace --verbose
```

## Project Discovery

- Projects are discovered via **glob patterns** in `arx-workspace.yaml`
- Duplicate paths are **automatically deduplicated**
- Unmatched globs cause an error — protecting against misconfiguration
- Project names are derived from the directory basename
- Results are sorted by path for deterministic output
