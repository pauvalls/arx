# Design: Arx v0.15.0 — Man Pages, Hygiene, Ignore, Fuzz Tests

## Technical Approach

Four independent additive work streams: (1) `arx man` Cobra subcommand using `cobra.GenManTree`, (2) project hygiene files (`.editorconfig`, `.github/dependabot.yml`), (3) `.arxignore` domain type integrated into detector file discovery, (4) fuzz tests for Kotlin/PHP/Swift parsers following existing `parser_fuzz_test.go` pattern. No port or domain interfaces change except the new `ArxIgnore` type.

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|----------|---------|----------|----------|
| Man page output | `GenMan()` to stdout vs `GenManTree()` to dir | `GenManTree` matches standard man page conventions (section 1, one file per subcommand); stdout is simpler but non-standard | `GenManTree` with `--output` flag, default to `./man/` |
| Man page section | Section 1 (user commands) vs Section 8 (admin) | Arx is a developer CLI, not a system admin tool | Section 1 |
| ArxIgnore location | `internal/domain/` vs `internal/infrastructure/` | Domain holds pure business logic; ignore rules are domain concepts | `internal/domain/ignore.go` |
| Glob matching | `path.Match` (std) vs `doublestar` (third-party) | `path.Match` handles `*`, `?`, `[...]` — sufficient for `.arxignore`; no new dependency | `path.Match` from stdlib |
| Ignore integration | Modify each `FindXxxFiles` vs central filter in `ExtractImports` | Each detector has its own walk logic; central filter would require refactoring all detectors | Pass `*ArxIgnore` to each `FindXxxFiles` method; each detector calls `IsIgnored` during walk |
| ArxIgnore missing file | Return error vs return empty (no-op) | Missing `.arxignore` is the common case; error would break every project without one | Return empty `ArxIgnore` (no patterns) — silent no-op |
| Comment/blank lines in .arxignore | Treat as patterns vs skip | Standard convention: `#` comments and blank lines are not patterns | Skip lines starting with `#` or empty after trim |
| Fuzz seed corpus | Empty vs representative samples | Seed corpus guides fuzzing toward meaningful paths; existing tests use valid import lines | Seed with 3-5 valid import lines per language |

## Data Flow

### arx man
```
rootCmd.Execute()
  └── manCmd.RunE()
        └── rootCmd.GenManTree(header, dir)
              ├── arx.1
              ├── arx-check.1
              ├── arx-audit.1
              └── ... (one per subcommand)
```

### arxignore
```
LoadArxIgnore(root)
  ├── os.Stat(".arxignore") → missing → return empty ArxIgnore
  └── os.ReadFile(".arxignore")
        ├── skip blank lines
        ├── skip # comments
        └── collect Patterns []string

Detector.FindXxxFiles(projectRoot, ignore)
  └── filepath.Walk(projectRoot)
        ├── shouldSkipPath(path) → SkipDir
        └── ignore.IsIgnored(relPath) → skip file
```

### Fuzz tests
```
FuzzKotlinParse(f)
  ├── f.Add("import kotlin.collections.List")
  ├── f.Add("import com.example.domain.Order")
  └── f.Fuzz(data) → extractImportsFromLine(string(data))

FuzzPhpParse(f)    → extractImportsFromLine(string(data))
FuzzSwiftParse(f)  → extractImportsFromLine(string(data))
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/arx/man.go` | Create | `arx man` Cobra command with `--output` flag, calls `GenManTree` |
| `.editorconfig` | Create | Standard Go/editor settings (indent, charset, trim whitespace, final newline) |
| `.github/dependabot.yml` | Create | Dependabot config for Go modules and GitHub Actions |
| `internal/domain/ignore.go` | Create | `ArxIgnore` struct, `LoadArxIgnore()`, `IsIgnored()` |
| `internal/domain/ignore_test.go` | Create | Tests: parse patterns, comments, empty file, missing file, glob matching |
| `internal/infrastructure/detector/kotlin/detector.go` | Modify | `FindKotlinFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/php/detector.go` | Modify | `FindPHPFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/swift/detector.go` | Modify | `FindSwiftFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/go/detector.go` | Modify | `FindGoFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/typescript/detector.go` | Modify | `FindTSFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/python/detector.go` | Modify | `FindPythonFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/java/detector.go` | Modify | `FindJavaFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/rust/detector.go` | Modify | `FindRustFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/csharp/detector.go` | Modify | `FindCSharpFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/ruby/detector.go` | Modify | `FindRubyFiles` accepts `*ArxIgnore`, filters during walk |
| `internal/infrastructure/detector/kotlin/parser_fuzz_test.go` | Create | `FuzzKotlinParse` — seed corpus + fuzz `extractImportsFromLine` |
| `internal/infrastructure/detector/php/parser_fuzz_test.go` | Create | `FuzzPhpParse` — seed corpus + fuzz `extractImportsFromLine` |
| `internal/infrastructure/detector/swift/parser_fuzz_test.go` | Create | `FuzzSwiftParse` — seed corpus + fuzz `extractImportsFromLine` |
| `internal/ports/detector.go` | Modify | Add `ArxIgnore` param to `ExtractImports` signature (or pass via context) |

## Interfaces / Contracts

**ArxIgnore** (`internal/domain/ignore.go`):

```go
package domain

type ArxIgnore struct {
    Patterns []string
}

// LoadArxIgnore reads .arxignore from project root.
// Returns empty ArxIgnore (no patterns) if file does not exist.
func LoadArxIgnore(root string) (*ArxIgnore, error)

// IsIgnored checks if a path matches any pattern using path.Match.
// Patterns support *, ?, [...] glob syntax.
func (a *ArxIgnore) IsIgnored(path string) bool
```

**arx man command** (`cmd/arx/man.go`):

```go
var manCmd = &cobra.Command{
    Use:   "man",
    Short: "Generate man pages for arx",
    RunE: func(cmd *cobra.Command, args []string) error {
        output, _ := cmd.Flags().GetString("output")
        header := &cobra.GenManHeader{
            Title:   "ARX",
            Section: "1",
        }
        return rootCmd.GenManTree(header, nil, output)
    },
}
```

**Detector integration** — each `FindXxxFiles` method gains an optional `*domain.ArxIgnore` parameter:

```go
// Before:
func (d *KotlinDetector) FindKotlinFiles(projectRoot string) ([]string, error)

// After:
func (d *KotlinDetector) FindKotlinFiles(projectRoot string, ignore *domain.ArxIgnore) ([]string, error)
```

Inside the walk callback:
```go
relPath, _ := filepath.Rel(projectRoot, path)
if ignore != nil && ignore.IsIgnored(relPath) {
    return nil // skip file
}
```

**`.editorconfig`**:

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{md,yaml,yml,json,toml}]
indent_style = space
indent_size = 2
```

**`.github/dependabot.yml`**:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit — ArxIgnore | `LoadArxIgnore` with missing file returns empty | Temp dir without `.arxignore` |
| Unit — ArxIgnore | `LoadArxIgnore` parses patterns, skips `#` comments and blanks | Temp dir with fixture `.arxignore` |
| Unit — ArxIgnore | `IsIgnored` matches `*`, `?`, `[...]` glob patterns | Table-driven with `path.Match` semantics |
| Unit — ArxIgnore | `IsIgnored` returns false for non-matching paths | Table-driven |
| Unit — Detector integration | `FindKotlinFiles` with ignore skips matched paths | Temp fixture with `.arxignore` |
| Unit — man command | `arx man --output ./man` generates `.1` files | Temp dir + check file existence |
| Fuzz — Kotlin | `FuzzKotlinParse` never panics on random input | `go test -fuzz` |
| Fuzz — PHP | `FuzzPhpParse` never panics on random input | `go test -fuzz` |
| Fuzz — Swift | `FuzzSwiftParse` never panics on random input | `go test -fuzz` |

## Migration / Rollout

No migration required. All changes are additive:
- `arx man` is a new subcommand — no existing behavior changes
- `.editorconfig` and `dependabot.yml` are new files
- `.arxignore` is opt-in — projects without the file see no change (empty `ArxIgnore` is a no-op)
- Fuzz tests are new test files — no production code changes

## Open Questions

- [ ] Should `IsIgnored` use `filepath.Match` (OS-specific separator) or `path.Match` (always `/`)? Since `.arxignore` patterns are typically written with `/` regardless of OS (like `.gitignore`), `path.Match` with normalized paths is more portable.
- [ ] Should all 10 detectors be updated in one PR or split? Given the 400-line review budget, updating all detectors is ~10 small changes (one line each in the walk callback). Low risk to batch together, but could split into "core ignore logic" + "detector integration" if needed.
- [ ] Should `arx man` default to stdout (single merged output) or directory (standard man tree)? Directory output is standard for `GenManTree` and more useful — users can install individual pages.
