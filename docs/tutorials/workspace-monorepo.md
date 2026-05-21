# Setting Up Workspace Mode for a Monorepo

This tutorial walks through setting up arx workspace mode for a typical monorepo with multiple services and shared libraries.

## Scenario

```
monorepo/
├── services/
│   ├── auth/           # Authentication service
│   ├── billing/        # Billing service (legacy, known violations)
│   └── notifications/  # Notifications service
├── internal/
│   ├── shared/         # Shared libraries
│   └── proto/          # Protobuf definitions
├── tools/
│   └── migrate/        # Database migration tool
└── arx-workspace.yaml  # We'll create this
```

## Step 1: Create arx-workspace.yaml

```yaml
version: "1.0"

projects:
  - path: services/auth
  - path: services/billing
    override:
      max_violations: 10      # Legacy project — allow more violations
  - path: services/notifications
  - path: internal/shared
  - path: tools/migrate

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
      to: [application, infrastructure]
      type: Cannot
      severity: error
      explanation: Domain must not depend on application or infrastructure

    - id: no-circular
      type: MustNotCircular
      severity: error

  exclude:
    - vendor/**
    - node_modules/**
    - generated/**
```

## Step 2: Add per-project configs

Each service can have its own `arx.yaml` with project-specific rules:

```yaml
# services/auth/arx.yaml
version: "1.0"
layers:
  - name: handlers
    paths: ["handlers/**"]
  - name: usecases
    paths: ["usecases/**"]

rules:
  - id: handlers-call-usecases
    from: handlers
    to: [usecases]
    type: Must
    severity: error
```

If a project has no `arx.yaml`, only the shared config from `arx-workspace.yaml` applies.

## Step 3: Run workspace check

```bash
cd monorepo
arx workspace
```

Expected output:

```
Project                  Status    Violations
──────────────────────────────────────────────
services/auth            PASS              0
services/billing         WARN              3
services/notifications   PASS              0
internal/shared          PASS              0
tools/migrate            PASS              0
──────────────────────────────────────────────
4 of 5 projects PASS. 3 violations (all warnings).
```

## Step 4: JSON output for CI

```bash
arx workspace --json --output arx-report.json
```

## Step 5: Add a new project to the workspace

When adding a new service:

1. Create the directory
2. Optionally create `services/new-service/arx.yaml`
3. Add to `arx-workspace.yaml`:
   ```yaml
   projects:
     - path: services/new-service
   ```
4. Re-run: `arx workspace`

## Step 6: Using glob patterns for many projects

If you have many similar projects under a directory, use globs:

```yaml
projects:
  - path: services/*           # All services
  - path: internal/libs/*      # All libs
  - path: tools/*              # All tools
```

## Step 7: Advanced — override layers per project

A React frontend in the monorepo needs different rules:

```yaml
projects:
  - path: web/frontend
    override:
      layers:
        - name: components
          paths: ["components/**"]
        - name: pages
          paths: ["pages/**"]
        - name: api
          paths: ["api/**"]

      rules:
        - id: components-no-api
          from: components
          to: [api]
          type: Cannot
          severity: error
```

## Tips

1. **Start with the shared config** — put common rules in `shared`, project specifics in project `arx.yaml`
2. **Use overrides for legacy projects** — `max_violations` lets you set different thresholds
3. **Globs for growing monorepos** — `services/*` catches new services automatically
4. **Run `arx workspace --verbose`** to debug per-project detection
5. **Check unmatched globs** — arx errors if a glob pattern matches nothing, preventing silent misconfigurations
