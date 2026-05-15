# Roadmap

## ✅ v0.9.0 (Current — Overrides, Rust, GitHub Action)

- [x] `overrides[]` per-rule — Path-based severity downgrade and rule disable
- [x] Rust detector — `Cargo.toml` detection, `use` statement parsing
- [x] `.github/actions/arx-action/` — GitHub Action for CI/CD
- [x] Override-aware exit code — 0 when only overridden violations remain
- [x] JSON `overridden_count` in summary

## ✅ v0.8.0 (Kotlin, Watch, Hooks, Custom Rules)

- [x] Kotlin detector — `.kt` files, `build.gradle.kts` support
- [x] `arx check --watch` — Continuous file monitoring
- [x] `arx hook install` — Pre-commit hook
- [x] Custom rule `pattern` field — Regex matching on imports

## ✅ v0.7.0 (Baseline, Diff, Cache)

- [x] `arx baseline` — Suppress existing violations for incremental adoption
- [x] `arx diff` — Compare architecture between git refs
- [x] Performance cache — Only re-parse changed files
- [x] Baseline-aware CI — Exit 0 if no new violations

## ✅ v0.6.0 (Java Detector + Audit)

- [x] Java detector — Maven/Gradle projects
- [x] `arx audit` — Health reports with coupling matrix, debt score, trends
- [x] History persistence — `.arx-history/` with retention policy

## ✅ v0.5.0 (Presets + Diagrams)

- [x] `arx init --preset {clean,hexagonal,ddd}`
- [x] `arx diagram` — ASCII + Graphviz DOT

## ✅ v0.4.0 (Python Detector)

- [x] Python AST-based detector

## ✅ v0.3.0 (Explain + Circular Detection)

- [x] `arx explain <id>` — Detailed fix guidance
- [x] Circular dependency detection

## ✅ v0.2.0 (SARIF + Markdown)

- [x] SARIF and Markdown output formats
- [x] Violation cache

## ✅ v0.1.0 (MVP)

- [x] Go and TypeScript detectors
- [x] Basic `arx check` command

---

## 🔜 Future (v0.10.0+)

### C# Detector
**Priority:** Medium

Support for C# projects via `using` statement parsing.

### HTML Reports
**Priority:** Medium

Self-contained HTML reports with interactive coupling matrix, trend charts.

### Custom Rule DSL
**Priority:** Low

Domain-specific language for complex architectural rules with JavaScript/TypeScript
evaluation engine.

### Arx Server (Web UI)
**Priority:** Low

Web interface for architecture visualization, violation timeline, team collaboration.
