# Design: Arx v0.7.0 — Baseline, Diff Mode, and Performance Cache

## Technical Approach

Three additive capabilities layered onto the existing hexagonal architecture:

1. **Performance Cache** — wraps detector `ExtractImports` calls with SHA-256 file-hash keyed cache in `.arx-cache/`. Cache key includes config hash so any `arx.yaml` change invalidates everything. Detectors remain unchanged; cache is an infrastructure concern applied at the `RunDetectors` application level.

2. **Baseline** — new domain model `Baseline` storing violation fingerprints (rule_id + file + line). `arx baseline` writes `.arx-baseline.json`. `arx check` loads baseline and filters matched violations before reporting. Exit code = 1 only for *new* violations when baseline exists.

3. **Diff Mode** — `arx diff <ref-before> <ref-after>` uses `git worktree` to isolate refs, runs full audit on each, compares violation sets. Requires clean working tree. Renders with color-coded terminal output.

Implementation order: **cache → baseline → diff** (cache is foundational, baseline depends on violation model, diff reuses both).

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|----------|---------|----------|----------|
| Cache placement | Per-detector vs application-level | Per-detector adds interface changes; application-level is transparent | **Application-level** in `RunDetectors` — no detector interface change |
| Baseline storage | JSON vs SQLite vs embedded | JSON is simple, portable, versionable; SQLite overkill for fingerprints | **JSON** at `.arx-baseline.json` — matches existing `.arx-cache/` pattern |
| Diff isolation | `git stash` + checkout vs `git worktree` | Stash is fragile with dirty trees; worktree is clean but needs disk space | **`git worktree`** — safer, parallelizable, documented requirement |
| Violation fingerprint | Full struct hash vs (rule_id+file+line) | Full hash is brittle to message changes; tuple is stable | **Tuple (rule_id, file, line)** — stable across explanation changes |
| Config hash | SHA-256 of arx.yaml content | Simple, deterministic, catches any config change | **SHA-256 of config bytes** — used in cache key and baseline staleness |

## Data Flow

### Cache Flow

```
runCheck
  │
  ├─ Load config → compute configHash
  │
  ├─ RunDetectors (wrapped)
  │    ├─ For each detector:
  │    │    ├─ Walk project files
  │    │    ├─ For each file: SHA-256(content) → cacheKey
  │    │    ├─ Check .arx-cache/{detector}/{cacheKey}.json
  │    │    │    ├─ HIT → return cached imports
  │    │    │    └─ MISS → ExtractImports → write cache
  │    │    └─ Aggregate all imports
  │    └─ Return []Dependency
  │
  ├─ Evaluate rules → []Violation
  │
  └─ Report + exit
```

### Baseline Flow

```
arx baseline:
  runCheck → []Violation → Baseline.Generate() → .arx-baseline.json

arx check (with baseline):
  runCheck → []Violation
    ├─ Load .arx-baseline.json
    ├─ Baseline.Filter(violations) → newViolations
    ├─ Report newViolations
    └─ Exit 1 if len(newViolations) > 0, else 0
```

### Diff Flow

```
arx diff <before> <after>:
  ├─ Validate clean working tree
  ├─ git worktree add .arx-worktree/before <before>
  ├─ git worktree add .arx-worktree/after <after>
  ├─ Audit(before) → violationsBefore
  ├─ Audit(after)  → violationsAfter
  ├─ Compare sets:
  │    Added     = after - before
  │    Resolved  = before - after
  │    Unchanged = before ∩ after
  ├─ Render (green=resolved, red=added, dim=unchanged)
  └─ git worktree remove (cleanup)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/baseline.go` | Create | `Baseline` struct, `Fingerprint()`, `Generate()`, `Filter()` methods |
| `internal/application/baseline.go` | Create | `BaselineService` with `Generate()`, `Load()`, `IsSuppressed()` |
| `internal/infrastructure/baseline/storage.go` | Create | JSON file read/write for `.arx-baseline.json` |
| `internal/infrastructure/cache/file_cache.go` | Create | `FileCache` with `Get()`, `Put()`, `Clear()`, `ConfigHash()` |
| `internal/application/diff.go` | Create | `DiffService` with `Compare()` using git worktree |
| `internal/application/cache.go` | Create | Cache-aware `RunDetectorsCached()` wrapping existing `RunDetectors` |
| `cmd/arx/baseline.go` | Create | `baseline` cobra command |
| `cmd/arx/diff.go` | Create | `diff` cobra command with `<ref-before> <ref-after>` args |
| `cmd/arx/check.go` | Modify | Add `--no-cache`, `--baseline` flags; modify `runCheck` to use cache/baseline |
| `cmd/arx/root.go` | Modify | Wire cache and baseline services into `newCheckService` |
| `internal/domain/config.go` | Modify | Add `ConfigHash()` method on `Config` |
| `internal/infrastructure/output/diff_renderer.go` | Create | Color-coded terminal renderer for diff results |
| `internal/ports/cache.go` | Create | `Cache` interface for testability |

## Interfaces / Contracts

### Cache Interface

```go
// internal/ports/cache.go
package ports

type Cache interface {
    Get(key string) ([]byte, bool)
    Put(key string, data []byte) error
    Clear() error
    ConfigHash() string
}
```

### Baseline Domain Model

```go
// internal/domain/baseline.go
package domain

type Baseline struct {
    Version    string    `json:"version"`
    ConfigHash string    `json:"config_hash"`
    Generated  time.Time `json:"generated"`
    Violations []BaselineViolation `json:"violations"`
}

type BaselineViolation struct {
    RuleID      string `json:"rule_id"`
    File        string `json:"file"`
    Line        int    `json:"line"`
    SourceLayer string `json:"source_layer"`
    TargetLayer string `json:"target_layer"`
}

// Fingerprint returns a stable identifier for a violation
func (bv BaselineViolation) Fingerprint() string {
    return fmt.Sprintf("%s:%s:%d", bv.RuleID, bv.File, bv.Line)
}

// IsSuppressed checks if a violation matches any baseline entry
func (b *Baseline) IsSuppressed(v Violation) bool {
    fp := BaselineViolation{
        RuleID: v.RuleID, File: v.File, Line: v.Line,
        SourceLayer: v.SourceLayer, TargetLayer: v.TargetLayer,
    }.Fingerprint()
    for _, bv := range b.Violations {
        if bv.Fingerprint() == fp {
            return true
        }
    }
    return false
}

// Filter returns violations NOT present in baseline (i.e., new ones)
func (b *Baseline) Filter(violations []Violation) []Violation {
    var newViolations []Violation
    for _, v := range violations {
        if !b.IsSuppressed(v) {
            newViolations = append(newViolations, v)
        }
    }
    return newViolations
}

// Generate creates a baseline from current violations
func GenerateBaseline(violations []Violation, configHash string) *Baseline {
    b := &Baseline{
        Version:    "1.0",
        ConfigHash: configHash,
        Generated:  time.Now(),
    }
    for _, v := range violations {
        b.Violations = append(b.Violations, BaselineViolation{
            RuleID: v.RuleID, File: v.File, Line: v.Line,
            SourceLayer: v.SourceLayer, TargetLayer: v.TargetLayer,
        })
    }
    return b
}
```

### Config Hash

```go
// Added to internal/domain/config.go
func (c *Config) Hash() (string, error) {
    data, err := json.Marshal(c)
    if err != nil {
        return "", err
    }
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:]), nil
}
```

### Diff Result

```go
// internal/application/diff.go
type DiffResult struct {
    Added     []domain.Violation `json:"added"`
    Resolved  []domain.Violation `json:"resolved"`
    Unchanged []domain.Violation `json:"unchanged"`
    RefBefore string             `json:"ref_before"`
    RefAfter  string             `json:"ref_after"`
}
```

### File Cache Entry

```go
// internal/infrastructure/cache/file_cache.go
type CacheEntry struct {
    FileHash       string    `json:"file_hash"`
    ConfigHash     string    `json:"config_hash"`
    DetectorName   string    `json:"detector_name"`
    Dependencies   []domain.Dependency `json:"dependencies"`
    Timestamp      time.Time `json:"timestamp"`
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `Baseline.IsSuppressed()` / `Filter()` | Table-driven tests with matching/non-matching fingerprints |
| Unit | `FileCache.Get/Put/Clear` | Temp directory, verify JSON read/write, config hash invalidation |
| Unit | `DiffResult` comparison logic | Mock violation sets, verify set operations |
| Unit | `Config.Hash()` | Same config → same hash, different → different hash |
| Integration | `arx baseline` → `.arx-baseline.json` | Golden file test with known project |
| Integration | `arx check --baseline` exit codes | Seed baseline, add/remove violations, assert exit 0/1 |
| Integration | Cache hit/miss on re-run | Run check twice on unchanged project, verify cache hit on second |
| E2E | `arx diff HEAD~1 HEAD` | Git repo with known violation changes between commits |

## Migration / Rollout

No migration required. All three features are additive:
- Cache is opt-in via config section; detectors fall back to no-cache path
- Baseline only affects `arx check` when `.arx-baseline.json` exists
- Diff is a new command with no persistent state

## Open Questions

- [ ] Should `--baseline` flag accept a custom path (default: `.arx-baseline.json`)? Proposal says default only for v0.7.0.
- [ ] Cache max size eviction strategy: LRU by timestamp or hard delete when directory exceeds threshold? Propose timestamp-based for simplicity.
- [ ] Should diff mode support `--config` flag to use a different `arx.yaml` at each ref? Defer to v0.8.0.
