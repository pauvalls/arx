# Design: arx v0.25.0 — Dashboard Filters, State Persistence, and Check Diff

## Technical Approach

Three independent changes that share no cross-dependencies:
1. **Dashboard filters** — pure client-side JS/CSS additions to `dashboard.html`, no backend changes.
2. **State persistence** — JSON serialization of `ServerState` to `.arx-cache/server-state.json`, loaded on startup.
3. **Check diff** — persist violations to `.arx-cache/last-check.json`, reuse existing `domain.DiffViolations` for comparison.

All changes follow existing patterns: vanilla JS (no frameworks), stdlib JSON for persistence, and the `.arx-cache/` directory convention established by the performance cache (v0.7.0).

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Filter implementation | Client-side JS filtering of in-memory array | Server-side query params | Zero backend changes, instant response, matches "no Go changes needed" scope. |
| Debounce approach | `setTimeout`/`clearTimeout` wrapper (300ms) | `input` event with throttle | `setTimeout` is simpler, ES5-compatible, no extra dependencies. |
| Sort state | Single-column sort with toggle (asc→desc→none) | Multi-column sort | Single-column matches user mental model for a violations table; multi-column adds UI complexity. |
| State persistence format | JSON file (`.arx-cache/server-state.json`) | SQLite, gob encoding | JSON is human-readable for debugging, matches existing cache pattern, stdlib `encoding/json`. |
| State load timing | Load before initial check; if load fails, proceed with empty state | Block startup on load failure | Server should always start; stale state is better than no state. |
| Check diff storage | `.arx-cache/last-check.json` (full violations array) | Fingerprint-only file | Need full violation data to display added/resolved details, not just counts. |
| Diff comparison | Reuse existing `domain.DiffViolations` (used by watch mode) | New comparison function | DRY — watch mode already implements this logic correctly. |
| Diff output scope | Terminal mode only | Also JSON mode | JSON consumers can implement their own diff; terminal users benefit most from the summary. |

## Data Flow

### Dashboard Filters

```
User interacts with filter UI
    │
    ├──► Checkbox change → update activeFilters.severity[]
    ├──► Dropdown change → update activeFilters.sourceLayer
    ├──► Input (debounced 300ms) → update activeFilters.searchText
    └──► Header click → update activeFilters.sortColumn / sortDirection
              │
              ▼
    applyFilters(allViolations) → filteredViolations
              │
              ▼
    renderViolationsTable(filteredViolations)
    updateSummary("Showing X of Y violations")
```

### State Persistence

```
arx server startup
    │
    ├──► LoadFromFile(.arx-cache/server-state.json)
    │         ├─ Success → populate ServerState with cached data
    │         └─ Failure → start with empty state (no error)
    │
    ├──► Run initial check → update ServerState
    │
    └──► On each subsequent check:
              │
              ├──► Update ServerState (in-memory)
              └──► SaveToFile(.arx-cache/server-state.json)
```

### Check Diff

```
arx check --diff
    │
    ├──► Run normal check → currentViolations
    │
    ├──► Load .arx-cache/last-check.json → previousViolations
    │         └─ If no file: show "no previous run to compare"
    │
    ├──► domain.DiffViolations(previous, current) → WatchResult
    │         ├─ Added (new violations)
    │         ├─ Resolved (fixed violations)
    │         └─ Unchanged
    │
    ├──► Print diff summary: "+3 violations, -1 resolved, 7 unchanged"
    │
    └──► Save currentViolations to .arx-cache/last-check.json (for next run)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/infrastructure/server/dashboard.html` | Modify | Add filter bar HTML (checkboxes, dropdown, search input), sortable th elements, filter summary text, JS filter/sort/debounce logic, CSS for active filter indicators. |
| `internal/infrastructure/server/dashboard_test.go` | Modify | Tests for filter bar presence in rendered HTML, sortable header attributes, filter summary text. |
| `internal/infrastructure/server/state.go` | Modify | Add `SaveToFile(path string) error` (JSON marshal + os.WriteFile) and `LoadFromFile(path string) error` (os.ReadFile + JSON unmarshal). Add serializable snapshot struct. |
| `internal/infrastructure/server/server.go` | Modify | Add `cachePath string` field to `Server`. In `Start()`, call `LoadFromFile` before initial check. In `runCheck()`, call `SaveToFile` after state update. |
| `cmd/arx/check.go` | Modify | Add `--diff` bool flag. After check run, save violations to `.arx-cache/last-check.json`. When `--diff` is set, load previous violations and print diff summary (terminal mode only). |
| `cmd/arx/check_test.go` | Modify | Test `--diff` flag registration and type. |

## Interfaces / Contracts

### State Persistence (state.go)

```go
// serverStateSnapshot is the serializable form of ServerState.
type serverStateSnapshot struct {
    LastCheck  time.Time            `json:"last_check"`
    Violations []domain.Violation   `json:"violations"`
    Coupling   domain.CouplingMatrix `json:"coupling"`
    Debt       domain.DebtScore     `json:"debt"`
    Config     *domain.Config       `json:"config,omitempty"`
    CheckError string               `json:"check_error,omitempty"`
}

// SaveToFile writes the current state to a JSON file.
func (s *ServerState) SaveToFile(path string) error

// LoadFromFile reads state from a JSON file and updates the ServerState.
func (s *ServerState) LoadFromFile(path string) error
```

### Check Diff Cache Format (.arx-cache/last-check.json)

```json
{
  "version": "1",
  "timestamp": "2025-01-15T10:30:00Z",
  "config_hash": "abc123...",
  "violations": [
    {
      "id": "v1",
      "rule_id": "no-infra-dep",
      "file": "internal/app/handler.go",
      "line": 42,
      "source_layer": "app",
      "target_layer": "infrastructure",
      "message": "app layer must not depend on infrastructure",
      "severity": "error"
    }
  ]
}
```

### Dashboard Filter State (client-side JS)

```javascript
// Internal filter state (not persisted, reset on page reload)
var activeFilters = {
  severities: ['error', 'warning', 'info'],  // which severities to show
  sourceLayer: '',                            // '' = all
  searchText: '',                             // lowercase, matched against rule_id, file, message
  sortColumn: '',                             // '' = default order, or column key
  sortDirection: 'asc'                        // 'asc' or 'desc'
};
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `ServerState.SaveToFile` / `LoadFromFile` round-trip | Create state, save to temp file, load into new state, assert equality |
| Unit | `LoadFromFile` with missing file | Assert no error, state remains at defaults |
| Unit | `LoadFromFile` with corrupt JSON | Assert error returned, state unchanged |
| Unit | `Server` calls `LoadFromFile` before initial check | Mock state, verify method called in `Start()` |
| Unit | `Server` calls `SaveToFile` after each check | Mock state, verify method called after `runCheck()` |
| Unit | `--diff` flag registered on checkCmd | `checkCmd.Flags().Lookup("diff")`, assert bool type, default false |
| Integration | Dashboard HTML contains filter bar elements | Parse rendered HTML, assert checkboxes, dropdown, search input exist |
| Integration | Dashboard HTML contains sortable th attributes | Assert `data-sortable` or `aria-sort` attributes on th elements |
| Integration | Filter summary text present in HTML | Assert "Showing X of Y" template exists |

## Migration / Rollout

No migration required. All changes are additive and backward-compatible:
- Dashboard filters are client-side only; existing dashboard works without them.
- State persistence uses a new file in `.arx-cache/`; absence of the file is handled gracefully.
- `--diff` flag is opt-in; `arx check` without `--diff` behaves identically to before.

## Open Questions

- [ ] Should the server state cache have a max age (e.g., discard if older than 24h)? **Recommendation: no for v0.25.0**, add later if stale data becomes an issue.
- [ ] Should `--diff` support a custom previous file path? **Recommendation: no**, default to `.arx-cache/last-check.json` for simplicity.
- [ ] Should filter state persist across page reloads (localStorage)? **Recommendation: defer**, out of scope for this change.
