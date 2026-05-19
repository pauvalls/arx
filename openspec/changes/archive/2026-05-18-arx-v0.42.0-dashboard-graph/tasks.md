# Tasks: Dashboard Dependency Graph

## Phase 1: Add `/api/config` to polling loop

- [x] 1.1 Add `fetchOne('/api/config', 'config')` to fetchData()

## Phase 2: Pass data to renderGraph in checkDone()

- [x] 2.1 Call `renderGraph({ coupling, violations, config })` after `renderCoupling()` in checkDone()

## Phase 3: CSS for dependency graph

- [x] 3.1 Add `.dep-graph`, `.dep-node`, `.dep-arrow`, `.dep-label`, `.dep-tooltip`, `.graph-empty` CSS classes

## Phase 4: renderGraph() function

- [x] 4.1 Implement `renderGraph(data)` that builds SVG with circular layout, nodes, arrows, tooltip, and click/hover interactions

## Phase 5: Graph section HTML template

- [x] 5.1 Add `<section id="graph-section">` after coupling section and before debt section
