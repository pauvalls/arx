# Detectors

## What Detectors Do

Detectors are the component that **discovers your project's dependencies**. They scan source files, extract import/require/use statements, and build a dependency graph that arx evaluates against your rules.

The detection pipeline is:

```
Project files → Detect (language presence) → ExtractImports (dependency extraction) → Dependency graph
```

Each detector handles one language. arx comes with built-in detectors for 10 languages and supports custom detectors via the plugin system.

## Built-in Languages

| Language | Detection File | Import Method | Since |
|----------|---------------|---------------|-------|
| Go | `go.mod` | AST parsing | v0.1.0 |
| TypeScript | `tsconfig.json` | Regex (`import` / `require`) | v0.1.0 |
| Python | `requirements.txt` / `setup.py` | AST parsing | v0.4.0 |
| Java | `pom.xml` / `build.gradle` | Regex (`package` + `import`) | v0.6.0 |
| Kotlin | `build.gradle.kts` | Regex (`import` + `alias`) | v0.8.0 |
| Rust | `Cargo.toml` | Regex (`use` statements) | v0.9.0 |
| C# | `.csproj` / `.sln` | Regex (`using` directives) | v0.10.0 |
| Ruby | `Gemfile` | Regex (`require` / `require_relative`) | v0.13.0 |
| PHP | `composer.json` | Regex (`use` / `use function`) | v0.14.0 |
| Swift | `Package.swift` | Regex (`import` / `@_exported import`) | v0.14.0 |

### Detection by Language

**Go** is the most advanced detector. It uses actual Go AST parsing to extract import statements, so it understands aliased imports, grouped imports, and blank imports. It's the reference implementation.

**TypeScript/JavaScript** handles both `import` (ES modules) and `require()` (CommonJS) syntax, including dynamic imports, default imports, and side-effect-only imports.

**Python** uses AST parsing for `import x` and `from x import y` statements. It handles relative imports (`.`, `..`) and converts them to absolute paths.

**Java, Kotlin, Rust, C#, Ruby, PHP, Swift** use regex-based pattern matching tuned to each language's import syntax. These are simpler but cover the vast majority of real-world projects.

## How Detection Works

### Phase 1: Detect

Arx checks whether a project uses a given language by looking for marker files:

```bash
# Go
go.mod exists? → Go is present

# TypeScript
tsconfig.json or package.json with typescript dep? → TypeScript detected

# Python
setup.py, setup.cfg, or requirements.txt? → Python detected
```

Detect is fast — it's a simple file existence check. It's always run first to avoid scanning irrelevant files.

### Phase 2: ExtractImports

For each detected language, arx walks the project's source files, parses imports, and records each dependency with:

- **Source file** — where the import comes from
- **Source line** — line number of the import statement
- **Import path** — the raw import string
- **Resolved layer** — which layer the import resolves to (based on your arx.yaml)

Dependencies are then classified by layer membership. If an import points to a file in the `infrastructure` layer, its resolved layer is `infrastructure`.

### Performance

Detectors are optimized for speed:

- **Cache**: File contents are cached in `.arx-cache/` between runs. Only modified files are re-scanned.
- **Parallelism**: Detectors run in parallel where possible.
- **Selective scanning**: Only files matching a detector's extensions are processed.

You can see performance breakdown with:

```bash
arx check --profile
```

This shows per-detector timing:

```
Performance Profile:
Detector              Duration
──────────────────────────────────────
Go                     12.3ms
TypeScript             45.1ms
Python                  8.7ms
──────────────────────────────────────
Total                  66.1ms
```

## Plugin System

For languages not covered by built-in detectors, arx supports **external plugins**. A plugin is any executable that implements the [arx plugin protocol](../plugins.md):

1. Arx runs the plugin as a subprocess
2. Sends a JSON request on stdin (action: `detect` or `extract`)
3. Plugin responds with JSON on stdout

Configure plugins in `arx.yaml`:

```yaml
plugins:
  - name: dart-detector
    command: dart run bin/detect.dart
    languages: [dart]
    timeout: 30s
    extensions: [.dart]
```

See the [Plugin System docs](../plugins.md) for the full protocol specification and authoring guide.

## Cross-Language Detection

Arx can detect relationships between files in **different languages**. This is essential for projects that generate code from specifications:

```
proto/user.proto (Protobuf) → internal/pb/user.pb.go (Generated Go)
openapi/spec.yaml (OpenAPI) → internal/client/api_client.ts (Generated TypeScript)
```

Configure mappings in `arx.yaml`:

```yaml
cross_language:
  mappings:
    - source_pattern: "proto/**/*.proto"
      generated_pattern: "**/*.pb.go"
      language: go
      match_strategy: stem

    - source_pattern: "openapi/**/*.yaml"
      generated_pattern: "**/api/*.ts"
      language: typescript
      match_strategy: glob
      header_patterns: ["@generated", "auto-generated"]
```

### Match Strategies

| Strategy | Behavior |
|----------|----------|
| `stem` | Matches by filename stem (e.g., `user.proto` → `user.pb.go`) |
| `glob` | Matches by glob pattern (e.g., `openapi/**/*.yaml` → `**/api/*.ts`) |

### Header Patterns

Custom header patterns help identify auto-generated files. Default patterns include `protoc-gen`, `@generated`, `OpenAPI Generator`, and `auto-generated`.

## Diagnostic Tools

### Verbose mode

```bash
arx check --verbose
```

Shows per-detector status:

```
Detectors:
  ✓ Go: 142 dependencies
  ✓ TypeScript: 89 dependencies
  ✗ Python: no project files found
```

### Doctor

```bash
arx doctor
```

Checks project root, config file validity, detector applicability, git status, and arx version in one command.

### Profile

```bash
arx check --profile
```

Shows timing breakdown per detection phase, helping identify bottlenecks.
