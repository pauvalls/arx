# Output Formats

Arx supports multiple output formats for different use cases.

## Terminal (Default)

Colored, educational output with explanations. Best for local development:

```
╔═══════════════════════════════════════════════════════════╗
║         ARCHITECTURE VIOLATIONS DETECTED                  ║
╚═══════════════════════════════════════════════════════════╝

❌ [D-01] domain/order.go:14
   Rule: "domain" → "infrastructure"

   Why this matters:
   The domain layer is the heart of your business logic...

   How to fix:
   1. Define an interface in the domain layer
   2. Move the concrete implementation to infrastructure
   3. Inject via constructor (Dependency Inversion)

═══════════════════════════════════════════════════════════
Found 1 violation (1 errors, 0 warnings, 0 info)
```

## JSON (CI/CD)

Machine-readable for CI pipelines and tooling:

```bash
arx check --format json > results.json
# or
arx check --ci > results.json
```

```json
{
  "version": "1.0",
  "tool": "arx",
  "violations": [
    {
      "id": "D-01",
      "rule_id": "domain-cannot-depend-on-infrastructure",
      "severity": "error",
      "file": "internal/domain/order.go",
      "line": 14,
      "source_layer": "domain",
      "target_layer": "infrastructure",
      "import": "github.com/example/app/internal/infrastructure/postgres",
      "message": "The domain layer is the heart of your business logic...",
      "overridden": false
    }
  ],
  "summary": {
    "total": 1,
    "errors": 1,
    "warnings": 0,
    "info": 0,
    "overridden_count": 0
  }
}
```

## SARIF

Static Analysis Results Interchange Format — compatible with GitHub Code Scanning:

```bash
arx check --format sarif > results.sarif
```

Upload to GitHub:

```yaml
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

## Markdown

For embedding in documentation or PR comments:

```bash
arx check --format markdown > violations.md
```

## Diff Output

`arx diff` supports terminal and JSON formats:

```bash
arx diff HEAD~1 HEAD                                    # Terminal
arx diff main feature --format json                     # JSON
```

## Audit Output

`arx audit` adds terminal and JSON with comprehensive metrics:

```bash
arx audit                               # Terminal report
arx audit --format json                 # JSON with all metrics
arx audit --output report.json          # Write to file
arx audit --trend                       # Trend comparison only
```
