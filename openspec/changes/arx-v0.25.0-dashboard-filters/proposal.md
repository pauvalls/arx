# Proposal: arx v0.25.0 — Dashboard Filters, State Persistence, and Check Diff

## Why

The dashboard (v0.24.0) shows all violations in a flat table. As projects grow, this becomes hard to navigate. Three improvements are needed:
1. **Client-side filters** — severity, source layer, search, and sorting to find relevant violations quickly.
2. **State persistence** — survive server restarts so the dashboard shows the last known state immediately.
3. **Check diff** — compare current violations against the previous run to spot regressions at a glance.

## What Changes

### 1. Dashboard Filters (client-side JS)
- Severity filter checkboxes (error, warning, info)
- Source layer dropdown populated from violation data
- Search text input with 300ms debounce
- Sortable column headers (click to toggle asc/desc)
- Filter summary: "Showing X of Y violations"
- CSS improvements for active filter indicators

### 2. State Persistence (server-side Go)
- `SaveToFile(path string) error` and `LoadFromFile(path string) error` on `ServerState`
- `CachePath string` field on `Server`
- Save state after each check, load on startup before initial check

### 3. Check Diff (CLI Go)
- `--diff` bool flag on `arx check`
- Save violations to `.arx-cache/last-check.json` after each run
- On `--diff`: load previous violations, compare by fingerprint, show diff summary
- Terminal output only (not JSON mode)

## Impact

- **Modified files**: 6 (dashboard.html, dashboard_test.go, state.go, server.go, check.go, check_test.go)
- **New files**: 0
- **Dependencies**: None new
- **Breaking changes**: None

## Approach

Dashboard filters are pure vanilla JS additions — no Go backend changes. State persistence reuses the existing `.arx-cache/` directory pattern. Check diff mirrors the existing watch-mode diff logic but persists between invocations.
