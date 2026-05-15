# Proposal: Arx v0.5.0 — Config Presets + Dependency Diagram

## Intent

Arx requires manual configuration writing for new projects, creating friction for adoption. Users must understand hexagonal architecture concepts before they can run their first audit. Additionally, there's no visual way to understand the current dependency graph of a codebase, making it hard to communicate architecture to teams.

This change solves two problems:
1. **Onboarding friction**: New users struggle to create their first `arx.yaml` without deep architecture knowledge
2. **Visibility gap**: Teams cannot visualize their actual dependency graph to understand coupling

## Scope

### In Scope
- `arx init --preset {clean,hexagonal,ddd}` — Three predefined configuration templates
- `arx diagram [--output file.dot]` — Generate dependency graph in Graphviz DOT format
- Terminal rendering of dependency graph (ASCII art fallback)
- Template files stored in `configs/presets/`
- Integration with existing detector infrastructure for graph data

### Out of Scope
- GUI visualization (web interface)
- Interactive diagram exploration
- Additional output formats (PNG, SVG) — DOT only for v0.5.0
- Auto-suggest rules from project structure (deferred to v0.6.0)
- Layer coupling matrix visualization

## Capabilities

### New Capabilities
- `config-presets`: Predefined configuration templates for common architecture patterns (clean, hexagonal, ddd)
- `dependency-diagram`: Generate visual representation of layer dependencies in DOT format

### Modified Capabilities
- `project-initialization`: Extended to support `--preset` flag for template-based config generation

## Approach

**Config Presets**:
1. Create `configs/presets/` directory with three YAML templates:
   - `clean.yaml` — Clean Architecture (domain, application, infrastructure, interface)
   - `hexagonal.yaml` — Hexagonal/Ports-Adapters (domain, ports, adapters, infrastructure)
   - `ddd.yaml` — Domain-Driven Design (domain, application, infrastructure, interfaces)
2. Modify `cmd/arx/init.go` to accept `--preset` flag
3. Load preset template, apply project-specific paths, write config

**Dependency Diagram**:
1. Add `cmd/arx/diagram.go` command handler
2. Create `internal/application/diagram.go` service
3. Reuse existing detector infrastructure to build dependency graph
4. Export to DOT format with layer-based subgraphs
5. ASCII fallback for terminal output (no Graphviz installed)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/arx/diagram.go` | New | CLI command for diagram generation |
| `cmd/arx/init.go` | Modified | Add `--preset` flag support |
| `internal/application/diagram.go` | New | Diagram service + DOT exporter |
| `internal/application/init.go` | Modified | Preset loading logic |
| `configs/presets/*.yaml` | New | Three preset templates |
| `internal/ports/diagram.go` | New | Diagram port interface |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| DOT format incompatible with user's Graphviz version | Low | Use standard DOT 1.2 syntax, test with Graphviz 2.40+ |
| Preset templates don't match user's project structure | Medium | Document that presets are starting points, require review |
| Large codebases produce unreadable diagrams | Medium | Add `--max-depth` flag, layer grouping, document limitations |
| ASCII rendering breaks on non-standard terminals | Low | Use simple box-drawing characters, test on common terminals |

## Rollback Plan

1. **Config presets**: Remove `--preset` flag from `init.go`, delete `configs/presets/` directory. Existing `arx.yaml` files remain unaffected.
2. **Diagram command**: Remove `cmd/arx/diagram.go` and `internal/application/diagram.go`. No persistent state is created.

Both features are additive — no breaking changes to existing functionality.

## Dependencies

- Go 1.21+ (already required)
- Graphviz (optional, for rendering DOT files — not required for generation)
- Existing detector infrastructure (Go, TypeScript, Python)

## Success Criteria

- [ ] `arx init --preset hexagonal` generates valid `arx.yaml` in <2 seconds
- [ ] All three presets (clean, hexagonal, ddd) produce working configurations
- [ ] `arx diagram` outputs valid DOT file that renders in Graphviz
- [ ] Terminal ASCII diagram renders correctly on standard terminals
- [ ] 80%+ test coverage for new features
- [ ] Documentation updated in README.md
