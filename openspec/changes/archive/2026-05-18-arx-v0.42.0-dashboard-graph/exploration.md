## Exploration: Dashboard Dependency Graph

### Current State

The dashboard is a **single self-contained HTML file** (`dashboard.html`) embedded via `go:embed` and rendered as a Go template. It uses **zero external JS/CSS dependencies** — only vanilla JS for client-side interactivity.

**Rendering flow:**
1. Server renders initial HTML with `dashboardData` (violations, coupling, debt, metrics)
2. On page load, JS starts a 30s polling loop via `setInterval`
3. Each poll fires 5 XHR requests in parallel: `/api/violations`, `/api/coupling`, `/api/debt`, `/api/status`, `/api/metrics`
4. JS re-renders the violations table, coupling matrix, and summary cards from JSON responses
5. Client-side filtering (severity, layer, search text) and sorting operate on the full cached dataset

**Current display sections:**
- Summary cards (errors, warnings, info, debt, check duration, files, deps, detectors)
- Violations table (with filter bar + sortable columns)
- Coupling matrix table (from/to/count/percentage)
- Technical debt breakdown

**No graph visualization exists yet** — coupling is shown only as a table.

### Data Sources

All data needed for a dependency graph is **already** available via existing API endpoints:

| Data | API Endpoint | Structure | Use for Graph |
|------|-------------|-----------|---------------|
| Coupling matrix | `/api/coupling` | `[{from, to, count, percentage}]` | **Primary** — dependency arrows between layers |
| Violations | `/api/violations` | `[{source_layer, target_layer, severity}]` | Color coding — layer health status |
| Layer config | `/api/config` | `{layers: ["domain","application",...]}` | Node names + canonical ordering |
| Metrics | `/api/metrics` | `{files_scanned, total_deps}` | Sizing nodes by file/import count |

**Key insight:** The coupling matrix is the ideal data source. Violations enrich it with severity status for color coding. Metrics add file/dep counts for node sizing.

The `GetLayerStats` method on `CouplingMatrix` already provides `OutgoingDeps`, `IncomingDeps`, and `NetCoupling` per layer — usable directly for node sizing or edge thickness.

### Approaches

1. **Client-rendered SVG (vanilla JS)** — **RECOMMENDED**
   - Render SVG nodes + arrows from JS using the existing API polling data
   - Force-directed simulation is unnecessary for ≤10 layers; a circular or layered layout works
   - Arrow rendering via SVG `<path>` with marker-end for direction
   - Pros: No deps, full interactivity (click/hover), fits existing architecture, works with auto-refresh
   - Cons: Need to implement layout algorithm in vanilla JS (moderate complexity, ~200 lines)
   - Effort: **Medium** (~250-350 lines of JS)

2. **Server-rendered SVG (Go backend)**
   - Generate SVG on the Go side, serve at a new `/api/graph` endpoint as `<svg>` XML
   - Client embeds it inline or as `<img src="/api/graph">`
   - Pros: Simplest rendering, Go's `fmt.Fprintf` + SVG strings is straightforward
   - Cons: No interactivity (click/hover) without re-requesting, harder to animate transitions, coupling data is duplicated (already sent to client via `/api/coupling`)
   - Effort: **Low-Medium** (~100-150 lines of Go)

3. **CDN-lightweight library (e.g., SVG.js from CDN)**
   - Load SVG.js from CDN for declarative SVG construction
   - Pros: Less manual SVG DOM code, cleaner API for arrows/markers
   - Cons: Adds an external dependency (failure point), breaks "self-contained" nature, violates current architecture ethos
   - Effort: **Low** (but adds network dependency)

### Recommendation

**Approach 1 — Client-rendered vanilla SVG.**

Rationale:
- The current dashboard already polls API data and renders it client-side; the graph is a natural extension
- The coupling matrix data is **already fetched every 30s** — the graph just needs a new JS rendering function consuming the same data
- For 4 layers (domain, application, infrastructure, interfaces) a simple **circular layout** or **top-to-bottom layered layout** is sufficient; no force-directed algorithm needed
- SVG `<path>` with `marker-end` for directional arrows is ~20 lines of boilerplate
- Color coding per layer is trivial: check violations for each layer, assign green/yellow/red
- Interactive features (hover tooltip, click-to-filter) use standard DOM events already wired in the dashboard
- Zero new dependencies, zero build step changes, zero Go backend changes

**Layout approach**: Use a **vertical layered layout** that mirrors architectural flow:
```
       ┌──────────┐
       │ interfaces│
       └────┬─────┘
            │
       ┌────▼─────┐
       │application│
       └────┬─────┘
            │
       ┌────▼─────┐
       │infrastr. │
       └────┬─────┘
            │
       ┌────▼─────┐
       │  domain  │
       └──────────┘
```
Or a circular layout with uniform edge lengths for up to 10 layers.

### Risks

- **Layout algorithm complexity**: With many layers (10+), a manual layout may produce overlapping edges. Mitigation: use circular layout as fallback, keep max layers reasonable.
- **SVG performance on mobile**: Many arrows + hover effects could be slow on low-end devices. Mitigation: limit visible edges, use CSS `will-change`, debounce hover events.
- **Arrow overlap**: Dense coupling data produces many crossing arrows. Mitigation: use bezier curves with different curvatures, or allow filtering by selected layer.

### Ready for Proposal

Yes. The data, architecture fit, and approach are clear. The proposal should define:
- Which layout algorithm (circular vs layered)
- Node sizing strategy (files, imports, or both)
- Interactive behaviors (tooltip, click-to-filter, hover highlight)
- Color assignment logic (per-layer severity aggregation)
- How it integrates with the existing 30s polling loop

---

**Note:** This requires only frontend changes to `dashboard.html` — no Go backend changes needed. The `/api/coupling` endpoint already returns all necessary data.
