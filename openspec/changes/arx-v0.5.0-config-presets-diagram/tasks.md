# Implementation Tasks: Arx v0.5.0 — Config Presets + Dependency Diagram

**Spec**: `openspec/changes/arx-v0.5.0-config-presets-diagram/proposal.md`  
**Design**: `openspec/changes/arx-v0.5.0-config-presets-diagram/design.md`  
**Created**: 2026-05-14

---

## Task Breakdown

### T1 — Create preset template files (embed FS)
- **Description**: Create three YAML preset templates in `internal/infrastructure/preset/`:
  - `clean.yaml` — Clean Architecture layers
  - `hexagonal.yaml` — Ports & Adapters layers
  - `ddd.yaml` — DDD layers
- **Files**: 
  - `internal/infrastructure/preset/clean.yaml` (new)
  - `internal/infrastructure/preset/hexagonal.yaml` (new)
  - `internal/infrastructure/preset/ddd.yaml` (new)
  - `internal/infrastructure/preset/presets.go` (new)
  - `internal/infrastructure/preset/presets_test.go` (new)
  - `configs/presets/.gitkeep` (new)
- **Dependencies**: None
- **Effort**: S (1h)
- **Acceptance Criteria**:
  - [x] Three YAML files exist with valid syntax
  - [x] Each defines layers matching design spec
  - [x] Files follow Go embed conventions
  - [x] `ListPresets()` returns available names
  - [x] `LoadPreset(name)` validates and loads raw bytes
  - [x] Tests cover valid names, invalid names, non-existent presets

---

### T2 — Create preset loader service
- **Description**: Implement `internal/application/preset.go` con `PresetServiceImpl` que usa `infrastructure/preset.LoadPreset()`, parsea YAML a `domain.Config` y valida
- **Files**:
  - `internal/application/preset.go` (new)
  - `internal/application/preset_test.go` (new)
  - `internal/ports/preset.go` (new)
  - `internal/infrastructure/preset/clean.yaml` (modified - schema fix)
  - `internal/infrastructure/preset/hexagonal.yaml` (modified - schema fix)
  - `internal/infrastructure/preset/ddd.yaml` (modified - schema fix)
- **Dependencies**: T1
- **Effort**: M (2-3h)
- **Acceptance Criteria**:
  - [x] `PresetService` interface en `internal/ports/preset.go`
  - [x] `PresetServiceImpl` con `LoadPreset(name string) (*domain.Config, error)`
  - [x] Valida que el preset existe antes de cargar
  - [x] Parsea YAML a `domain.Config`
  - [x] Valida config resultante (layers definidos, reglas referencian layers existentes)
  - [x] Retorna error descriptivo si validation falla
  - [x] Tests: TestPresetService_LoadValidPreset, TestPresetService_InvalidPresetName, TestPresetService_ListPresets

---

### T3 — Modify init service to support presets
- **Description**: Extend `internal/application/init.go` con `LoadPreset()` integration y modificación de `GenerateConfig()` para aceptar preset
- **Files**:
  - `internal/application/init.go` (modified)
  - `internal/application/init_test.go` (modified)
- **Dependencies**: T2
- **Effort**: M (2-3h)
- **Acceptance Criteria**:
  - [ ] `GenerateConfig(preset, projectRoot)` signature updated
  - [ ] When preset provided, loads template + applies customization
  - [ ] When no preset, existing behavior unchanged
  - [ ] Tests verify both paths

---

### T4 — Add --preset flag to init CLI command
- **Description**: Add `--preset` / `-p` flag to `cmd/arx/init.go`, pass to service layer
- **Files**:
  - `cmd/arx/init.go` (modified)
- **Dependencies**: T3
- **Effort**: S (1h)
- **Acceptance Criteria**:
  - [ ] `--preset` flag accepts `clean`, `hexagonal`, `ddd`
  - [ ] Flag value passed to `GenerateConfig()`
  - [ ] Help text documents preset options
  - [ ] Integration test: `arx init --preset hexagonal` creates valid `arx.yaml`

---

### T5 — Create diagram port interface
- **Description**: Define `internal/ports/diagram.go` interface for diagram generation (follows existing port pattern)
- **Files**:
  - `internal/ports/diagram.go` (new)
- **Dependencies**: None (parallel track starts)
- **Effort**: S (30min)
- **Acceptance Criteria**:
  - [ ] Interface defines `BuildGraph()` and `Export()` methods
  - [ ] Follows naming conventions of existing ports
  - [ ] Documented with godoc comments

---

### T6 — Create diagram service
- **Description**: Implement `internal/application/diagram.go` con `DiagramService`, `BuildGraph()` que reusa detectors existentes
- **Files**:
  - `internal/application/diagram.go` (new)
  - `internal/application/diagram_test.go` (new)
- **Dependencies**: T5
- **Effort**: L (4-5h)
- **Acceptance Criteria**:
  - [ ] `DiagramService` accepts detector registry
  - [ ] `BuildGraph()` extracts dependencies from project
  - [ ] Returns `Graph` struct with layers, nodes, edges
  - [ ] Mock detector tests verify graph structure
  - [ ] Handles empty projects gracefully

---

### T7 — Create DOT exporter
- **Description**: Implement `internal/infrastructure/output/dot.go` con `DOTExporter.Export()` para Graphviz DOT format
- **Files**:
  - `internal/infrastructure/output/dot.go` (new)
  - `internal/infrastructure/output/dot_test.go` (new)
- **Dependencies**: T6
- **Effort**: M (2-3h)
- **Acceptance Criteria**:
  - [ ] Valid DOT 1.2 syntax output
  - [ ] Layer-based subgraphs with proper ranking
  - [ ] Edge weights reflect import count
  - [ ] Golden file tests verify DOT syntax
  - [ ] Can be rendered by Graphviz 2.40+

---

### T8 — Create ASCII renderer
- **Description**: Implement `internal/infrastructure/output/ascii.go` con `ASCIIRenderer.Render()` para terminal output (usar lipgloss si aplica)
- **Files**:
  - `internal/infrastructure/output/ascii.go` (new)
  - `internal/infrastructure/output/ascii_test.go` (new)
- **Dependencies**: T6
- **Effort**: M (3h)
- **Acceptance Criteria**:
  - [ ] ASCII art renders dependency tree
  - [ ] Uses box-drawing characters correctly
  - [ ] Optional: lipgloss colors for layers
  - [ ] Snapshot tests for terminal output
  - [ ] Works on standard terminals (xterm, tmux)

---

### T9 — Create diagram CLI command
- **Description**: Implement `cmd/arx/diagram.go` con cobra command, flags (`--output`, `--format`, `--max-depth`)
- **Files**:
  - `cmd/arx/diagram.go` (new)
- **Dependencies**: T7, T8
- **Effort**: M (2h)
- **Acceptance Criteria**:
  - [ ] `arx diagram [path]` command registered
  - [ ] `--output` / `-o` flag for file output
  - [ ] `--format` / `-f` flag: `dot`, `ascii`, `auto`
  - [ ] `--max-depth` / `-d` flag limits graph depth
  - [ ] Auto-detect format: ASCII if stdout, DOT if file
  - [ ] Integration test: outputs valid diagram

---

### T10 — Add integration tests for config presets
- **Description**: E2E tests: `arx init --preset {clean,hexagonal,ddd}` → verify `arx.yaml` → run `arx check`
- **Files**:
  - `test/integration/presets_test.go` (new)
- **Dependencies**: T4
- **Effort**: M (2h)
- **Acceptance Criteria**:
  - [ ] Test project created for each preset
  - [ ] Generated config passes validation
  - [ ] `arx check` runs without errors on preset config
  - [ ] Cleanup removes test artifacts

---

### T11 — Add integration tests for diagram command
- **Description**: E2E tests: `arx diagram` → validate DOT syntax via Graphviz (si disponible) → verify ASCII renders
- **Files**:
  - `test/integration/diagram_test.go` (new)
- **Dependencies**: T9
- **Effort**: M (2h)
- **Acceptance Criteria**:
  - [ ] Test project with known dependency structure
  - [ ] DOT output validated (syntax check)
  - [ ] ASCII output non-empty, properly formatted
  - [ ] `--max-depth` flag tested
  - [ ] `--exclude` flag tested (si se implementa)

---

### T12 — Update README.md with new features
- **Description**: Document `arx init --preset` y `arx diagram` en README.md con ejemplos de uso
- **Files**:
  - `README.md` (modified)
- **Dependencies**: T4, T9
- **Effort**: S (1h)
- **Acceptance Criteria**:
  - [ ] Preset examples for all 3 templates
  - [ ] Diagram usage with format options
  - [ ] Screenshots/ASCII examples (optional)
  - [ ] Mention Graphviz as optional dependency

---

### T13 — Update CONTRIBUTING.md with development notes
- **Description**: Add section on adding new presets, diagram architecture overview
- **Files**:
  - `CONTRIBUTING.md` (modified)
- **Dependencies**: T2, T6
- **Effort**: S (1h)
- **Acceptance Criteria**:
  - [ ] How to create custom preset templates
  - [ ] Diagram service architecture explained
  - [ ] Testing guidelines for new features

---

## Review Workload Forecast

| Metric | Estimate |
|--------|----------|
| **Total files created** | 11 new files |
| **Total files modified** | 5 existing files |
| **Lines added (estimated)** | ~1,200-1,500 LOC |
| **Lines of tests** | ~400-500 LOC |
| **Total PRs recommended** | 3-4 PRs (see below) |

### Recommended PR Breakdown

**PR #1 — Config Presets** (T1-T4, T10)
- Files: 6 new/modified
- LOC: ~400-500
- Risk: LOW (additive, no breaking changes)
- Reviewer focus: YAML template correctness, embed FS usage

**PR #2 — Diagram Core** (T5-T8)
- Files: 5 new
- LOC: ~500-600
- Risk: MEDIUM (graph algorithm complexity)
- Reviewer focus: Graph building logic, DOT syntax correctness

**PR #3 — Diagram CLI + Tests** (T9, T11)
- Files: 2 new
- LOC: ~200-250
- Risk: LOW (CLI wiring, integration tests)
- Reviewer focus: Flag handling, E2E test coverage

**PR #4 — Documentation** (T12, T13)
- Files: 2 modified
- LOC: ~100-150
- Risk: NONE
- Reviewer focus: Clarity, examples accuracy

---

## Critical Path

**Longest dependency chain** (cannot be parallelized):

```
T1 (presets) → T2 (loader) → T3 (init service) → T4 (init CLI) → T10 (preset tests)
  1h            2.5h          2.5h               1h              2h
  └─────────────────────────────────────────────────────────────────┘
                          9 hours total

T5 (diagram port) → T6 (diagram service) → T7 (DOT) → T9 (diagram CLI) → T11 (diagram tests)
     0.5h              4.5h                2.5h        2h                  2h
  └──────────────────────────────────────────────────────────────────────────┘
                              11.5 hours total
```

**CRITICAL PATH**: T5 → T6 → T7 → T9 → T11 = **11.5 hours** (diagram feature)

---

## Parallelization Opportunities

### Can run in parallel from start:
- **Track A (Presets)**: T1 → T2 → T3 → T4 → T10
- **Track B (Diagram)**: T5 → T6 → T7 + T8 (parallel) → T9 → T11

### Parallel branches within Track B:
```
                    T7 (DOT exporter)
                   ↗                    ↘
T6 (diagram service)                      → T9 (CLI)
                   ↘                    ↗
                    T8 (ASCII renderer)
```

**T7 y T8 can be done in parallel** after T6 completes — they both depend on the `Graph` struct but are independent implementations.

### Documentation track (can start anytime after features complete):
- T12, T13 can start once T4 y T9 are done

---

## Risk Assessment

| Task | Risk Level | Mitigation |
|------|------------|------------|
| T2 (embed FS) | LOW | Standard Go pattern, well-documented |
| T6 (graph building) | MEDIUM | Complex logic — pair review recommended |
| T7 (DOT syntax) | LOW | Use golden files, validate with Graphviz |
| T8 (ASCII render) | MEDIUM | Terminal compatibility — test on multiple terminals |
| T9 (CLI flags) | LOW | Follows existing cobra patterns |

**Overall project risk**: LOW-MEDIUM
- Both features are additive (no breaking changes)
- Rollback is simple (remove new files)
- Main complexity: graph building algorithm in T6

---

## Estimated Total Effort

| Effort | Count | Hours |
|--------|-------|-------|
| S (<2h) | 5 tasks | 5-7h |
| M (2-4h) | 6 tasks | 13-18h |
| L (>4h) | 2 tasks | 8-10h |
| **Total** | **13 tasks** | **26-35 hours** |

**Forecast**: 3-4 working days for single developer  
**With 2 developers** (parallel tracks): 2 working days

---

## Next Steps

1. **Confirm task breakdown** with team
2. **Assign owners** to Track A (presets) y Track B (diagram)
3. **Create GitHub issues** for each task (link to this document)
4. **Start with T1 + T5** (both have no dependencies, can run in parallel)
