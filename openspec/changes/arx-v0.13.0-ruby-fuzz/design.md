# Design: Ruby Detector + Parser Fuzz Tests

## Technical Approach

Add a Ruby language detector following the exact Rust detector pattern (two-file split: `detector.go` + `parser.go`), plus fuzz tests for the Java, C#, and Rust parsers following the existing `config_fuzz_test.go` pattern. The Ruby detector registers via the existing `GetDetectors()` factory. No domain or port interfaces change — `RubyDetector` implements `ports.Detector`.

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|----------|---------|----------|----------|
| File structure | Single file vs detector+parser split | Split matches Rust pattern, keeps parser regexes isolated | Two files: `detector.go` + `parser.go` |
| Detection signal | `Gemfile` vs `*.rb` file scan | `Gemfile` is unambiguous, fast single stat call | `Gemfile` in project root |
| Source dirs | `src/` (Rust convention) vs project root + `lib/` | Ruby convention places code at root and `lib/` | `["", "lib/"]` |
| External skip | `require 'gem'` vs whitelist stdlib | Ruby has no stable stdlib namespace; skip anything NOT `require_relative`/`require_all`/`require File.expand_path` | Skip all bare `require` |
| Fuzz test style | Go native `testing.F` vs go-fuzz | Native `testing.F` is built-in, matches existing `config_fuzz_test.go` | Native `FuzzXxxParse` |

## Data Flow

```
GetDetectors() ──→ RubyDetector (new entry)
                        │
                        ├── Detect() ──→ os.Stat("Gemfile") ──→ bool
                        │
                        └── ExtractImports()
                                │
                                ├── FindRubyFiles() ──→ filepath.Walk + skip spec/test/vendor
                                │
                                └── parseFile() ──→ regex extractImportsFromLine()
                                                        │
                                                        ├── require_relative → local import
                                                        ├── require_all    → local import
                                                        ├── require File.expand_path → local import
                                                        └── require 'gem'  → SKIP (external)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/infrastructure/detector/ruby/detector.go` | Create | `RubyDetector` struct, `Name()`, `Detect()`, `ExtractImports()`, `FindRubyFiles()`, `resolveImport()`, `parseFile()`, `shouldSkip()`, `shouldSkipPath()` |
| `internal/infrastructure/detector/ruby/parser.go` | Create | Regex patterns for `require`, `require_relative`, `require_all`, `require File.expand_path(...)`, `extractImportsFromLine()`, `isExternal()` |
| `internal/infrastructure/detector/ruby/detector_test.go` | Create | Unit tests for detector and parser |
| `internal/infrastructure/detector/ruby/parser_fuzz_test.go` | Create | `FuzzRubyParse(f *testing.F)` |
| `internal/infrastructure/detector/java/parser_fuzz_test.go` | Create | `FuzzJavaParse(f *testing.F)` |
| `internal/infrastructure/detector/csharp/parser_fuzz_test.go` | Create | `FuzzCSharpParse(f *testing.F)` |
| `internal/infrastructure/detector/rust/parser_fuzz_test.go` | Create | `FuzzRustParse(f *testing.F)` |
| `internal/infrastructure/detector/registry.go` | Modify | Import `rubydetector`, append `rubydetector.New()` to `GetDetectors()` |

## Interfaces / Contracts

**RubyDetector** implements `ports.Detector`:

```go
type RubyDetector struct {
    modulePrefix string
    sourceDirs   []string  // {"", "lib/"}
}

func (d *RubyDetector) Name() string                              // "ruby"
func (d *RubyDetector) Detect(ctx, projectRoot) (bool, error)     // os.Stat(Gemfile)
func (d *RubyDetector) ExtractImports(ctx, projectRoot, layers) ([]domain.Dependency, error)
```

**Parser regex patterns** (in `parser.go`):

```go
// require_relative 'path/to/file'  → local
requireRelativePattern = regexp.MustCompile(`^\s*require_relative\s+['"]([^'"]+)['"]`)

// require_all 'path'  → local
requireAllPattern = regexp.MustCompile(`^\s*require_all\s+['"]([^'"]+)['"]`)

// require File.expand_path('path', ...)  → local
requireExpandPattern = regexp.MustCompile(`^\s*require\s+File\.expand_path\s*\(\s*['"]([^'"]+)['"]`)

// require 'library'  → external (skip)
requirePattern = regexp.MustCompile(`^\s*require\s+['"]([^'"]+)['"]`)
```

**Fuzz function signature** (all three existing parsers):

```go
func FuzzXxxParse(f *testing.F) {
    // Seed with valid import lines
    f.Add([]byte("valid import line\n"))
    f.Fuzz(func(t *testing.T, data []byte) {
        result := extractImportsFromLine(string(data))
        _ = result // return early on any panic-inducing input
    })
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit — Ruby detector | `Detect()` returns true for `Gemfile`, false otherwise | Table-driven tests with temp dirs |
| Unit — Ruby parser | `extractImportsFromLine()` for all 4 require variants + comments + edge cases | Table-driven tests matching C#/Rust pattern |
| Unit — Ruby resolve | `resolveImport()` maps `require_relative` paths to layers via `shared.MatchImportToLayer` | Table-driven with mock layerMap |
| Fuzz — Java/C#/Rust parsers | `extractImportsFromLine()` with arbitrary bytes; seed with valid imports | Native `testing.F`, return on error |
| Fuzz — Ruby parser | Same pattern, seed with all 4 require variants | Native `testing.F` |
| Integration | Full `ExtractImports()` on a fixture Ruby project | Fixture in `test/fixtures/ruby-project/` |

## Migration / Rollout

No migration required. Ruby detector is a new registry entry — won't activate unless `Gemfile` exists in project root. Fuzz tests are additive and run alongside existing tests.

## Open Questions

- [ ] Should `require` with a relative-looking path (e.g., `require './lib/foo'`) be treated as local? Current proposal skips ALL bare `require` as external. This can be refined during implementation if needed.
