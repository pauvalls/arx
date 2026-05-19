# Proposal: Dashboard Dependency Graph

## Intent

Coupling shown as a table — hard to grasp dependency direction, density, and layer health. A visual graph makes architectural relationships immediately obvious.

## Scope

### In Scope
- Inline SVG graph rendered from existing API data (no Go changes)
- Layer bubbles sized by dep count, colored by severity: green (clean), yellow (warnings), red (errors)
- Directional bezier arrows from `/api/coupling`
- Click bubble → filter violations table to that layer
- Hover tooltip: OutgoingDeps, IncomingDeps, NetCoupling
- Circular layout, up to 10 layers
- New JS: `renderGraph()`, `layoutNodes()`, `drawArrows()`

### Out of Scope
- D3.js or external libs; server-rendered SVG; animated transitions; force-directed layout

## Capabilities

### New
- `dashboard-graph`: SVG dependency graph in the dashboard with circular layout, severity color-coding, click-to-filter, hover tooltips.

### Modified
None — no spec-level changes.

## Approach

Extend `dashboard.html` JS. On each 30s poll, after `fetchData()`, call `renderGraph(coupling, violations, metrics)`:

1. **layoutNodes(layers)** — polar → SVG cartesian. Radius proportional to layer count.
2. **drawArrows(entries, positions)** — SVG `<path>` + `marker-end`. Bezier curves staggered to avoid overlap. Thickness proportional to count.
3. **renderGraph()** — orchestrator: derive layers from coupling, compute per-layer severity from violations, size nodes from metrics, call layout + draw, attach event handlers.

Color: any error violation → red, any warning → yellow, all clean → green.

Click sets `activeFilters.sourceLayer` → `reRender()`. Hover shows `<div>` tooltip.

Graph section inserted between summary cards and violations table.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `dashboard.html` | Modified | +~250 JS, +~50 CSS, new SVG container |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Arrow overlap with dense coupling | Low | Bezier staggered curves; circular layout separates evenly |
| SVG performance | Low | Cap at 10 layers; `will-change` on hovered paths |
| Missing layer ordering | Low | Derive from coupling entries; `/api/config` as fallback |

## Rollback Plan

Revert `dashboard.html` to previous commit. Single file, no backend changes, no feature flag needed.

## Dependencies

None — all data from existing API endpoints.

## Success Criteria

- [ ] Graph renders all layers as colored circles with directional arrows
- [ ] Clicking a node filters violations table to that source layer
- [ ] Hovering shows tooltip with OutgoingDeps, IncomingDeps, NetCoupling
- [ ] Graph re-renders correctly on 30s poll with updated data
- [ ] All existing tests pass (`go test -race ./...`)
