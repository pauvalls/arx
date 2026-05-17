# Design: arx v0.26.0 — Metrics, Config Set Improvements, and Quality Pass

## Technical Approach

Three independent changes with no cross-dependencies:

1. **Performance metrics** — Add a `Metrics` struct to `ServerState`, populated during each `RunCheck`, exposed via `GET /api/metrics`, and rendered as cards in the dashboard HTML. Follows the existing `ServerState` mutex-protected getter pattern.

2. **Config set improvements** — Replace the hardcoded `switch` in `configSetCmd` with a generic dotted-path resolver that navigates nested YAML maps, plus JSON array parsing for list values. Also extends `configGetCmd` symmetrically.

3. **Quality pass** — Run existing fuzz targets, race detector, and `go vet` against the codebase. Fix any issues found. No new code — only verification and bug fixes.

All changes follow existing conventions: vanilla JS (no frameworks), stdlib `encoding/json` for APIs, `gopkg.in/yaml.v3` for YAML manipulation, and the mutex-protected `ServerState` pattern.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Metrics storage | `Metrics` field in `ServerState`, updated via `SetCheckResult` | Separate `MetricsState` struct | `ServerState` already owns all check-related state; adding one field keeps the single source of truth. |
| Metrics API | New `GET /api/metrics` handler returning JSON | Extend `/api/status` with metrics fields | Separate endpoint keeps `/api/status` stable; metrics are a distinct concern with different consumers. |
| Metrics update timing | Updated at end of `RunCheck` (same as `SetCheckResult`) | Real-time streaming or per-step updates | Check is fast; per-step adds complexity with no UX benefit. Dashboard polls every 30s anyway. |
| Config set parsing | Dotted path (`a.b.c`) → recursive `map[string]interface{}` traversal; JSON-parse array values | Use a library like `goccy/go-yaml` for path-based updates | No new dependency needed; `yaml.v3` already loaded; dotted path is simple and matches `config get` convention. |
| Config get symmetry | Extend `config get` to also support dotted paths | Leave `config get` as-is (only top-level) | Users who `set` nested values need to `get` them back; symmetry reduces confusion. |
| Config set array values | Accept JSON string: `'["vendor/**"]'` → `[]interface{}` | Comma-separated strings or special syntax | JSON is unambiguous, handles edge cases (commas in values), and is familiar to developers. |
| Quality pass scope | Run all existing fuzz targets, race detector, `go vet` | Add new fuzz targets or benchmarks | This change is about verifying existing quality gates, not expanding them. |

## Data Flow

### Performance Metrics

```
RunCheck() executes
    │
    ├──► service.Load()       → config loaded
    ├──► service.Detect()     → deps []Dependency
    ├──► service.Evaluate()   → violations []Violation
    ├──► CouplingCalculator   → coupling matrix
    └──► DebtScore            → debt score
              │
              ▼
    Build Metrics struct:
      - CheckDurationMs  (time.Since(start))
      - FilesScanned     (len(deps) unique files)
      - TotalDeps        (len(deps))
      - DetectorsRun     (count of applicable detectors)
      - UptimeSeconds    (time.Since(uptime))
              │
              ▼
    state.SetCheckResult(violations, coupling, debt, cfg, metrics, nil)
              │
              ▼
    GET /api/metrics → handler reads state.Metrics() → JSON response
              │
              ▼
    Dashboard JS polling → fetch /api/metrics → update metric cards
```

### Config Set/Get

```
arx config set severity_mapping.critical '["vendor/**"]'
    │
    ├──► Read arx.yaml → map[string]interface{}
    ├──► Split key: ["severity_mapping", "critical"]
    ├──► Navigate: doc["severity_mapping"] → map
    │              map["critical"] → target
    ├──► Parse value: JSON array → []interface{}
    ├──► Set: map["critical"] = []interface{}{"vendor/**"}
    └──► Marshal → Write arx.yaml

arx config get severity_mapping.critical
    │
    ├──► Read arx.yaml → map[string]interface{}
    ├──► Split key: ["severity_mapping", "critical"]
    ├──► Navigate: doc["severity_mapping"]["critical"]
    └──► Print value (YAML marshal for complex types)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/infrastructure/server/state.go` | Modify | Add `Metrics` struct (CheckDurationMs, FilesScanned, TotalDeps, DetectorsRun, UptimeSeconds). Add `metrics` field to `ServerState`. Update `SetCheckResult` to accept metrics. Add `Metrics()` getter. Update `CacheData` and `SaveToFile`/`LoadFromFile` to persist metrics. |
| `internal/infrastructure/server/server.go` | Modify | Add `handleMetrics` handler registered at `/api/metrics`. Update `RunCheck` to build and pass `Metrics` to `SetCheckResult`. Add `MetricsResponse` struct. |
| `internal/infrastructure/server/dashboard.html` | Modify | Add 5 new metric cards (check duration, files scanned, total deps, detectors run, uptime) to the summary-cards grid. Add JS to fetch `/api/metrics` in `fetchData()` and update card values. |
| `cmd/arx/config.go` | Modify | Replace `configSetCmd` switch with generic dotted-path resolver. Add JSON array parsing for values. Extend `configGetCmd` to support dotted paths. Add helper functions: `resolvePath`, `setAtPath`, `getAtPath`, `parseValue`. |
| `cmd/arx/config_test.go` | Create | Tests for dotted path resolution, JSON array parsing, fallback to string, nested map creation, and error cases (invalid path, type mismatch). |

## Interfaces / Contracts

### Metrics Struct (state.go)

```go
// Metrics holds performance data from the last architecture check.
type Metrics struct {
	CheckDurationMs int64  `json:"check_duration_ms"`
	FilesScanned    int    `json:"files_scanned"`
	TotalDeps       int    `json:"total_deps"`
	DetectorsRun    int    `json:"detectors_run"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
}
```

### Updated SetCheckResult Signature (state.go)

```go
// Before:
func (s *ServerState) SetCheckResult(violations []domain.Violation, coupling domain.CouplingMatrix, debt domain.DebtScore, cfg *domain.Config, checkErr error)

// After:
func (s *ServerState) SetCheckResult(violations []domain.Violation, coupling domain.CouplingMatrix, debt domain.DebtScore, cfg *domain.Config, metrics Metrics, checkErr error)
```

### Metrics API Response (GET /api/metrics)

```json
{
  "check_duration_ms": 342,
  "files_scanned": 156,
  "total_deps": 892,
  "detectors_run": 3,
  "uptime_seconds": 1800
}
```

### Config Set Value Parsing

```
Input value parsing order:
1. Try json.Unmarshal(raw) → if succeeds, use parsed value (array, object, number, bool)
2. Fall back to raw string

Examples:
  '["vendor/**"]'     → []interface{}{"vendor/**"}
  '{"key": "val"}'    → map[string]interface{}{"key": "val"}
  '42'                → float64(42)  (JSON number)
  'true'              → bool(true)   (JSON bool)
  'hello'             → "hello"      (string fallback)
```

### Dashboard Metric Cards HTML

```html
<!-- Added to .summary-cards grid (after existing 4 cards) -->
<div class="summary-card metric">
  <span class="value" id="metric-duration">0</span>
  <span class="label">Check (ms)</span>
</div>
<div class="summary-card metric">
  <span class="value" id="metric-files">0</span>
  <span class="label">Files</span>
</div>
<div class="summary-card metric">
  <span class="value" id="metric-deps">0</span>
  <span class="label">Dependencies</span>
</div>
<div class="summary-card metric">
  <span class="value" id="metric-detectors">0</span>
  <span class="label">Detectors</span>
</div>
```

CSS: `.summary-card.metric .value { color: var(--color-text-muted); font-size: 1.75rem; }` — smaller, muted to distinguish from severity counts.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `Metrics` struct JSON round-trip | Marshal/unmarshal, assert all fields |
| Unit | `ServerState.Metrics()` getter | Set metrics, read back, assert thread safety |
| Unit | `SetCheckResult` with metrics | Verify metrics field is stored atomically |
| Unit | `handleMetrics` handler | HTTP test server, assert JSON response, 200 status |
| Unit | `handleMetrics` method check | Assert 405 on non-GET |
| Unit | Dotted path resolver: `resolvePath` | Test top-level, nested, missing path, type mismatch |
| Unit | `setAtPath` creates intermediate maps | Test `a.b.c` on empty map → nested structure |
| Unit | `parseValue` JSON array | `'["a","b"]'` → `[]interface{}` |
| Unit | `parseValue` JSON object | `'{"k":"v"}'` → `map[string]interface{}` |
| Unit | `parseValue` string fallback | `"hello"` → `"hello"` |
| Unit | `getAtPath` with dotted key | Navigate nested map, return value |
| Integration | `arx config set severity_mapping.critical '["vendor/**"]'` | Run command, read arx.yaml, assert nested array |
| Integration | `arx config get severity_mapping.critical` | Run command, assert output matches set value |
| Integration | Dashboard HTML contains metric card elements | Parse rendered HTML, assert metric card IDs exist |
| Quality | `go test -fuzz=Fuzz -fuzztime=10s ./...` | All 8 fuzz targets pass without crashes |
| Quality | `go test -race ./...` | No data races detected |
| Quality | `go vet ./...` | No vet issues |

## Migration / Rollout

No migration required. All changes are additive and backward-compatible:
- Metrics endpoint is new; existing API consumers are unaffected.
- Dashboard metric cards are additive; existing cards render identically.
- `config set` accepts the same top-level keys as before; dotted paths are a new capability, not a breaking change.
- `config get` accepts the same top-level keys as before; dotted paths are a new capability.
- Quality pass verifies existing code; no behavioral changes.

## Open Questions

- [ ] Should `UptimeSeconds` be computed at read time (dynamic) or stored at check time (snapshot)? **Decision: computed at read time** — uptime is always `time.Since(state.uptime)`, no need to store a snapshot. The `Metrics` struct will store it for API consistency but the getter computes it fresh.
- [ ] Should the dashboard metrics cards use a separate fetch or piggyback on the existing parallel fetches? **Decision: separate fetch** — `/api/metrics` is a single lightweight request; adding it to the parallel `fetchData()` keeps the pattern consistent with other endpoints.
- [ ] Should `config set` validate that the target key exists before setting? **Decision: no** — YAML is flexible; users should be able to create new nested keys. Validation happens at `arx config validate` time.
