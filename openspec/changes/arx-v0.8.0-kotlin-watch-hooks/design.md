# Design: Arx v0.8.0 — Kotlin Detector, Watch Mode, Pre-commit Hooks, Custom Rules

## Technical Approach

Four additive features, each independent. Implementation order per proposal: custom rules → kotlin detector → pre-commit hook → watch mode. The custom-rules `pattern` field is foundational domain logic; Kotlin detector follows the exact Java detector pattern; hook is shell-script generation; watch mode builds on fsnotify + cache reuse.

## Architecture Decisions

### Decision: Shared glob matcher for import path resolution

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Export from `java` package and import in `kotlin` | Creates `kotlin → java` import coupling, breaks independence | Rejected |
| Extract to `internal/infrastructure/detector/shared/glob.go` | Clean dependency, both packages import from peer utility | **Chosen** |
| Duplicate in `kotlin/parser.go` | Divergent code, maintenance burden | Rejected |

Extract `importMatchesLayer` from `java/parser.go` into `detector/shared/glob.go` as exported `ImportMatchesLayer`. Java detector calls shared version. Kotlin detector imports shared.

### Decision: KotlinDetector detects alongside Java (not instead of)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| KotlinDetector replaces JavaDetector for `pom.xml` | Breaks mixed projects with Kotlin + Java sources | Rejected |
| Both detect independently on `pom.xml` | Both return `true`, both scan; Java scans `.java`, Kotlin scans `.kt` | **Chosen** |

Registry returns both detectors. On `pom.xml`, both activate and scan their respective file types. Duplicate detection is harmless — same layers, distinct files.

### Decision: Watch mode uses full re-scan with cache

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Incremental per-file re-parsing | Complex, requires per-file dependency tracking | Rejected |
| Full re-scan via existing cache pipeline | Simple, cache layer already handles hash invalidation, development use-case means one file changed at a time | **Chosen** |

Cache hashes ALL source files for a detector. A single change misses cache → full re-scan. Acceptable: debounce prevents bursts, and during development only one file typically changes between saves.

### Decision: Pre-commit hook exits based on baseline

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Hook runs `arx check` with `--only-staged` | Would require modifying detectors to read only staged file content | Rejected |
| Hook runs standard `arx check`, relies on baseline to suppress known violations | Zero modifications to check pipeline; baseline already handles NEW-vs-known distinction | **Chosen** |

### Decision: Watcher reads `.gitignore` manually, doesn't shell out to `git check-ignore`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Shell out to `git check-ignore` | Fast, accurate, but adds subprocess per event | Rejected |
| Parse `.gitignore` once, match patterns in-memory | Single read at startup, no subprocess. Slightly different semantics (no `core.excludesFile`) | **Chosen** |

### Decision: `compiledPattern` added as unexported field on `Rule`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Add `Pattern` string only, compile at every `Violates()` call | Simpler model, but regex compile on every dep × rule = N² compile cost | Rejected |
| Add both `Pattern string` (YAML) + `compiledPattern *regexp.Regexp` (compiled at load) | Transient field doesn't serialize in JSON/YAML, compiled once per rule at config validation time | **Chosen** |

## Data Flow

### Custom rules (pattern matching)

```
arx.yaml ──→ YAMLReader.Read()
                ↓
          domain.Config (Rule.Pattern string)
                ↓
          domain.Config.Validate()
                ↓
          Rule.Validate() → regexp.Compile(Pattern)
                ↓
          Rule.compiledPattern (*regexp.Regexp) set

Check flow:
  Violates(importPath, sourceLayer, targetLayer)
    → matches from/to layers? → matches compiledPattern? → violation
```

### Kotlin detector

```
Registry.GetDetectors()
  → append kotlin.New()

Detect(projectRoot):
  build.gradle.kts exists? → return true
  pom.xml exists? → check for *.kt files → return true/false

ExtractImports():
  Walk .kt/.kts files
    → parseFile() → extractImportsFromLine()
      → standard | wildcard | alias | nested
    → resolveImport() → shared.ImportMatchesLayer()
```

### Watch mode

```
arx check --watch ./project
    ↓
  runCheck() — normal full check
    ↓
  NewWatcher(projectRoot, interval, gitignore)
    ↓
  Event loop:
    fsnotify event → debounce timer (500ms)
      → re-run RunDetectorsCached()
      → re-run EvaluateArchitecture()
      → diff: oldViolations vs newViolations
      → print WatchResult (Added / Resolved)
      → if --json: output per-change JSON diff
```

### Pre-commit hook

```
arx hook install
    ↓
  shell script → .git/hooks/pre-commit
    ├── SKIP=arx guard
    ├── git diff --cached --name-only
    ├── arx check --no-cache
    └── exit code passes/fails commit

arx hook uninstall
    ↓
  os.Remove(.git/hooks/pre-commit)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/rule.go` | Modify | Add `Pattern string`, `compiledPattern *regexp.Regexp`; update `Validate()` and `Violates()` |
| `internal/domain/rule_test.go` | Modify | Add tests for pattern matching in Violates |
| `internal/domain/config.go` | Modify | `Config.Validate()` calls `Rule.Validate()` (already does — pattern validation inside rule) |
| `internal/domain/watch_result.go` | Create | `WatchResult` struct with Added/Resolved/Current violation sets |
| `internal/infrastructure/detector/shared/glob.go` | Create | Exported `ImportMatchesLayer()` — glob-to-regex import path matcher |
| `internal/infrastructure/detector/java/parser.go` | Modify | Refactor `importMatchesLayer` to call shared version |
| `internal/infrastructure/detector/kotlin/detector.go` | Create | `KotlinDetector` — Detect(), ExtractImports(), FindKotlinFiles() |
| `internal/infrastructure/detector/kotlin/parser.go` | Create | Regex patterns for Kotlin imports (standard, wildcard, alias, nested) |
| `internal/infrastructure/detector/registry.go` | Modify | Import and register `kotlin.New()` |
| `internal/infrastructure/watcher/watcher.go` | Create | `Watcher` struct with fsnotify, debounce, gitignore filtering |
| `cmd/arx/check.go` | Modify | Add `--watch` flag, watch loop after initial check, diff output |
| `cmd/arx/hook.go` | Create | `arx hook install/uninstall` commands |
| `go.mod` | Modify | Add `github.com/fsnotify/fsnotify` v1.7+ |

## Interfaces / Contracts

```go
// internal/domain/watch_result.go
type WatchResult struct {
    Added    []Violation `json:"added"`
    Resolved []Violation `json:"resolved"`
    Current  []Violation `json:"current"`
}

// internal/infrastructure/watcher/watcher.go
type Watcher struct {
    fsnotify  *fsnotify.Watcher
    debounce  time.Duration
    ignore    []string        // compiled .gitignore patterns
    events    chan WatchEvent
}

type WatchEvent struct {
    Op      fsnotify.Op
    Path    string
    Time    time.Time
}

func NewWatcher(projectRoot string, interval time.Duration, ignorePatterns []string) (*Watcher, error)
func (w *Watcher) Events() <-chan WatchEvent
func (w *Watcher) Close() error

// internal/domain/rule.go (modified)
type Rule struct {
    ID               string     `yaml:"id"`
    From             string     `yaml:"from"`
    To               []string   `yaml:"to"`
    Type             RuleType   `yaml:"type"`
    Severity         Severity   `yaml:"severity"`
    Explanation      string     `yaml:"explanation,omitempty"`
    Pattern          string     `yaml:"pattern,omitempty"`      // NEW
    compiledPattern  *regexp.Regexp                              // NEW, unexported
}

// internal/infrastructure/detector/shared/glob.go
func ImportMatchesLayer(importPath, layerPattern string) bool

// internal/infrastructure/detector/kotlin/detector.go
type KotlinDetector struct {
    modulePrefix    string
    sourceDirs      []string
}

func New() *KotlinDetector
func (d *KotlinDetector) Name() string                       // "kotlin"
func (d *KotlinDetector) Detect(ctx, projectRoot) (bool, error)
func (d *KotlinDetector) ExtractImports(ctx, projectRoot, layers) ([]domain.Dependency, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `Rule.Violates()` with `Pattern` | Table-driven: pattern matches, pattern doesn't match, invalid pattern rejected at Validate |
| Unit | Kotlin parser regex | Table-driven: standard import, wildcard, alias `as`, nested class, class with companion |
| Unit | KotlinDetector.Detect | Test with fake `build.gradle.kts`, `pom.xml`, empty dir |
| Unit | Watcher debounce | Timer-based: fire 2 events within 500ms, verify single callback |
| Unit | `ImportMatchesLayer` | Same tests as Java parser's import matching |
| Unit | Hook shell script | String comparison of generated script content (don't exec) |
| Unit | `WatchResult` diff | Two violation sets → verify Added/Resolved |
| Unit | Config validation with invalid pattern | `pattern: "[invalid"` → `Validate()` returns error |
| Unit | `isSourceFile` for Kotlin | `.kt` → true, `.kts` → true, `.java` → false (in cache.go) |
| Integration | Kotlin + Java mixed project | Both detectors active on `pom.xml` project, verify both contribute dependencies |

Key: Kotlin detector tests mirror Java detector test structure (`*_test.go` per file). No E2E tests for watch mode (time-dependent). Hook test verifies script content and file permissions, not execution.

## Migration / Rollout

No migration required. All features are additive:
- `Pattern` field is optional in YAML — existing configs load unchanged (`omitempty`)
- Kotlin detector is a new entry in registry — won't activate unless `build.gradle.kts` or `.kt` files exist
- `--watch` flag defaults to false — existing CLI usage unchanged
- Hook install writes to `.git/hooks/` — opt-in, no migration needed

## Open Questions

- [x] Should KotlinDetector scan `src/main/java` for `.kt` files? **Yes** — mixed projects put Kotlin alongside Java. The detector walks all dirs (skipping build output) regardless of Java sourceDirs.
- [x] Handle `.kts` (Gradle Kotlin Script) files? **Yes** — scan `.kt` and `.kts`, but `.kts` files are typically build scripts, not project source. Walk excludes `build/` and `target/` directories where generated `.kts` live. Source `.kts` is rare but supported.
