# Roadmap

## ✅ v0.27.0 (Current — arx suggest / Auto-fix)

- [x] `arx suggest` command — shows fix suggestions for architecture violations
- [x] Fix templates: domain→infrastructure, application→infrastructure
- [x] Violation-specific suggestions: `arx suggest D-01`
- [x] `--apply` flag auto-applies fixes with `.arx-backup/` safety
- [x] `--force` flag skips confirmation prompt
- [x] `--output` flag writes diff to file
- [x] Atomic rollback on error (all-or-nothing)

## ✅ v0.26.0 (Performance Metrics, Config Improvements, Quality)

- [x] Performance metrics: check duration, files scanned, total deps, detectors run, uptime
- [x] `/api/metrics` endpoint + metrics cards on dashboard
- [x] `arx config set` supports dotted paths (`severity_mapping.critical`), JSON arrays, numbers
- [x] `arx config get` supports dotted paths for complex values
- [x] Quality pass: `go vet` clean, `go test -race` 0 data races

## ✅ v0.25.0 (Dashboard Filters, State Persistence, Check Diff)

- [x] Dashboard filtering by severity (checkboxes), layer (dropdown), and search text
- [x] Sortable violation columns (asc/desc/none with visual arrows)
- [x] Filter summary ("Showing X of Y violations") + empty state
- [x] Server state persistence (`.arx-cache/server-state.json`) — survives restart
- [x] `arx check --diff` — Shows violations added/removed since last check run
- [x] Color-coded diff output: red (new), green (resolved), dim (unchanged)

## ✅ v0.24.0 (Web Server + Dashboard)

- [x] `arx server` command with `--port`, `--bind`, `--path` flags
- [x] Embedded HTML dashboard with responsive CSS and print styles
- [x] REST API: `/api/health`, `/api/status`, `/api/violations`, `/api/coupling`, `/api/debt`
- [x] Auto-refresh via 30s ticker + fsnotify file watcher
- [x] Graceful shutdown on SIGINT/SIGTERM
- [x] Zero external web dependencies (stdlib net/http only)

## ✅ v0.23.0 (Hardening & E2E Mega Release)

- [x] E2E tests for 6 language fixtures + all CLI commands
- [x] Baseline workflow, threshold, expression rules E2E
- [x] Multi-language fixture (Go + TS + Python)
- [x] Python dot-to-slash import resolution fix

## ✅ v0.22.0 (Config CLI, Severity Filter)

- [x] `arx config get/set` — Read/modify arx.yaml from CLI
- [x] `arx check --severity <level>` — Filter violations by severity

## ✅ v0.21.0 (Audit HTML, JSON Metadata, Quality)

- [x] Full HTML audit report with coupling matrix, debt, trends
- [x] JSON check output with coupling matrix + detector metadata
- [x] Quality pass: go vet, fuzz, deprecated API removal

## ✅ v0.20.0 (Maturity Release)

- [x] JSON Schema for arx.yaml IDE autocompletion
- [x] NO_COLOR support
- [x] Smart arx init (auto .gitignore)
- [x] Verbose check (per-detector status)

## ✅ v0.19.0 — v0.1.0 (Previous releases)

*See [CHANGELOG.md](CHANGELOG.md) for complete history of all 19 previous releases.*

---

## ✅ v0.30.0 — filter()/map() (DSL completion)
**Priority:** Medium | **Effort:** M

- [x] `filter(deps(a,b), "field op value")` — filter deps by predicate string
- [x] `map(deps(a,b), "field")` — extract field values as ValueList
- [x] `ValueList` type with `count()` support
- [x] Predicate evaluator: ==/!= for string fields, all 6 ops for SourceLine
- [x] Tokenizer enhancement: quoted string support for predicate args

## ✅ v0.29.0 — Rule Hot-Reload
**Priority:** Medium | **Effort:** S

- [x] `POST /api/reload` — endpoint to force config re-read and full re-check
- [x] `GET /api/config` — endpoint returning layers, rules summary, and function names
- [x] File watcher logs when `arx.yaml` changes — config hot-reload with no restart

---

## ✅ v0.31.0 — Quality of Life (fmt, detect, dashboard)
**Priority:** Medium | **Effort:** S

- [x] `arx fmt` — Formats arx.yaml with consistent indentation and key order
- [x] `arx init --detect` — Dry-run scan: shows detected layers without writing config
- [x] Dashboard: config reloaded indicator (⚡ flash when arx.yaml changes)
- [x] Dashboard: pre-commit hook and docs links in footer
- [x] Release workflow with goreleaser auto-brew on tag

---

## ✅ v0.32.0 — Cross-Language Dependency Resolution (MVP)
**Priority:** High | **Effort:** L

- [x] `Language` field on Dependency — all 10 detectors set it
- [x] `cross_language.mappings` config section for proto→generated file rules
- [x] `CrossLanguageDetector` — glob matching, stem matching, header verification
- [x] Wired into `Check()` pipeline as post-processing phase
- [x] Synthetic `Dependency{Language: "cross"}` linking proto definitions to generated code

## ✅ v0.33.0 — Cross-Language Extensions (OpenAPI, glob strategy)
**Priority:** High | **Effort:** M

- [x] `MatchStrategy` field: `stem` (default) or `glob` (source×generated matching)
- [x] OpenAPI spec detection via `glob` strategy + auto-generated header
- [x] `HeaderPatterns` field for custom generated-file header patterns
- [x] Default patterns include protoc-gen, @generated, OpenAPI Generator, auto-generated
- [x] Config validation for match_strategy values

---

## ✅ v0.34.0 — Explain + Suggest, Benchmarks, Fuzz, Docs
**Priority:** Medium | **Effort:** M

- [x] `arx explain` now shows auto-fix suggestions from the suggest engine
- [x] Expression parser benchmarks (parse + eval, all builtins)
- [x] Expression parser fuzz tests (FuzzParseExpression, FuzzEvaluateExpression)
- [x] `docs/expression-rules.md` — full DSL reference with examples
- [x] `docs/cross-language.md` — proto, OpenAPI, custom headers
- [x] `docs/suggest.md` — auto-fix, explain, fix templates
- [x] Strict TDD enabled for all future SDD phases

## ✅ v0.35.0 — AI Assistant Integration (arx skill install)
**Priority:** Medium | **Effort:** S

- [x] `arx skill install` command — installs arx-setup skill to AI coding assistants
- [x] Auto-detects opencode, Claude Code, and Cursor
- [x] Interactive selector when run without arguments
- [x] `--all` flag to install to all detected tools
- [x] `contrib/opencode/arx-setup/SKILL.md` — distributable skill file
- [x] `docs/ai-integration.md` — documentation with examples

## ✅ v0.36.0 — arx explain: real code context + code-aware fixes
**Priority:** Medium | **Effort:** M

- [x] CODE CONTEXT section in explain: reads actual file, shows lines around violation
- [x] AUTO-FIX SUGGESTION with real diffs based on actual file content
- [x] FixEngine: layer-based fallback matching (source→target)
- [x] FixEngine: code-aware fix generation (reads file, finds import, generates suggestion)
- [x] Template aliases for common rule IDs (domain-no-infra, app-no-infra, etc.)

## ✅ v0.37.0 — Quality pass: dogfooding, Clean Architecture, CI gates
**Priority:** High | **Effort:** M

- [x] Clean Architecture refactor: BaselineStorage, PresetLoader interfaces in ports
- [x] DoctorService: detectors injected via constructor (no infra imports in app layer)
- [x] CI quality gates: vet + race + coverage checks (core >50%)
- [x] Makefile: vet, test-race, cover, quality targets
- [x] 37 releases, 26 packages, all passing with race detector

## ✅ v0.38.0 — init --detect with real import analysis
**Priority:** Medium | **Effort:** M

- [x] Lightweight Go import scanner (regex-based, no AST, ~100ms for 200+ files)
- [x] Per-layer dependency breakdown in `arx init --detect` output
- [x] Maps imports to detected layers for real architecture understanding
- [x] Detected layers, rules, and generated YAML in unified output

## ✅ v0.39.0 — TypeScript scanner + dependency stats in arx check
**Priority:** Medium | **Effort:** M

- [x] TypeScript/JavaScript import scanner (import, require, side-effect)
- [x] Multi-language scanner architecture (extensible per extension)
- [x] `arx check` now shows: "Dependencies: 1044 imports across 4 layers (221 files scanned)"
- [x] Works for both Go and TypeScript projects

---

## 🔜 v0.40.0 — v0.50.0 Roadmap

### ✅ v0.40 — Language Detector Hardening
**Effort:** M

- [x] Go detector: 79% coverage (was N/A), 6 tests + fuzz (first tests ever!)
- [x] Python detector: 73.6% coverage (was 37.7%), new fuzz test
- [x] TypeScript detector: 45.4% coverage (was 19.9%), new fuzz test
- [x] All 12 detectors pass with race detector
- [x] Integration tests hardened (uses -count=1 to avoid cache issues)

### v0.41 — Config Strict Mode + Validation
**Effort:** S

- `arx config validate --strict` fails on unknown keys
- Better error messages with violation IDs and line numbers
- Schema `--dry-run` shows what would change
- Config upgrade command: `arx config upgrade` migrates old formats

### ✅ v0.42 — Dashboard Dependency Graph
**Effort:** M

- [x] Interactive SVG dependency graph with circular layout
- [x] Directional bezier arrows from coupling matrix data
- [x] Color-coded nodes: green (clean), yellow (warnings), red (errors)
- [x] Hover tooltip with incoming/outgoing/violations counts
- [x] Click node to filter violations table by layer
- [x] Arrow highlighting on hover
- [x] 9 new tests, TDD flow, zero Go backend changes

### ✅ v0.43 — Rule Testing Framework
**Effort:** L

- [x] `arx test` command — test rules against fixtures
- [x] YAML-based test definitions: `given violations, expect result`
- [x] Built-in test fixtures for common architecture patterns
- [x] CI integration: `arx test --ci` with JUnit output
- [x] Examples in docs for writing custom rule tests
- [x] Domain types: TestSuite, TestCase, Expectation, MatchMode
- [x] ViolationMatcher: Count, Files, Layers, Patterns (AND logic)
- [x] Expectation matching via EvalSuite
- [x] RuleTestRunner with config reader + detection pipeline
- [x] YAML parser with validation (no duplicates, required fields)
- [x] Table output reporter (human-readable, --verbose)
- [x] JUnit XML reporter (CI-ready)
- [x] Application service orchestrating parse → run → report
- [x] Rule-based violation filtering per test case
- [x] Strict TDD: 40+ tests across all layers (unit + integration)

### ✅ v0.44 — Multi-Project / Workspace Mode
**Effort:** L

- [x] `arx workspace` — run check across multiple sub-projects
- [x] Shared config with per-project overrides (shallow merge)
- [x] Aggregated violation reports across the workspace
- [x] `arx.yaml` workspace discovery (globs, monorepo layout)
- [x] Glob-based project discovery with duplicate dedup
- [x] Terminal table output (PASS/FAIL per project)
- [x] JSON output (`--json` flag, `--output` file)
- [x] Error isolation (one failing project doesn't block others)
- [x] WorkspaceConfig domain types with validation
- [x] WorkspaceService orchestration layer
- [x] Strict TDD: 30+ tests across domain, application, output, CLI

### v0.45 — Performance Pass
**Effort:** M

- Profile-guided optimization of the detection pipeline
- Parallel detector execution tuning
- Benchmark-driven improvements (target: 2x faster detection)
- Optimize the cross-language detector (file caching)
- Add `arx check --profile` for performance breakdown

### v0.46 — Baseline Auto-Refresh + History
**Effort:** M

- Auto-refresh baseline when violations are consistently resolved
- Baseline history tracking (`.arx-baseline-history/`)
- Trend visualization: "violations over time"
- `arx baseline --diff` shows what changed since last baseline
- Integration with `arx check --diff`

### v0.47 — Config Includes + Schema Generation
**Effort:** M

- `!include` directive for splitting configs
- `arx schema generate` — generates JSON Schema from existing arx.yaml
- Config composition: base config + overrides
- Environment variable interpolation in config
- YAML anchor support (`&defaults`, `<<: *defaults`)

### v0.48 — Suggest Batch Mode + Conflict Detection
**Effort:** M

- `arx suggest --all` applies ALL fixes with smart conflict detection
- Staged changes (like git add -p): review fixes before applying
- Rollback per-file (not just full rollback)
- Fix preview with `--dry-run`
- Integration with `arx explain` for each fix

### v0.49 — Dashboard Real-Time (WebSocket)
**Effort:** M

- Replace polling with WebSocket for real-time updates
- Push notifications when violations change
- Live config reload indicator (already supported server-side)
- Connection status indicator
- Auto-reconnect on disconnect

### v0.50 — Ultimate Quality Pass
**Effort:** L

- 100% test coverage in all core packages (`internal/domain`, `internal/application`)
- Fuzz tests for ALL language parsers (10 languages)
- `go test -race -count=10 ./...` — flaky test detection
- Performance benchmarks in CI (regression detection)
- All dogfooding violations in arx's own codebase fixed
- Documentation audit: every feature has a doc with examples
- Security audit: no secret leaks, proper file permissions
- Final v50.0 release — the most polished version of arx

---

### Cross-Language Dependency Resolution
**Priority:** Low | **Effort:** XL

Detect dependencies across language boundaries:
- gRPC proto files → generated clients in any language
- REST API contracts → HTTP client calls
- Shared configuration/environment variables
- Message queue topics → consumers/producers

### GitHub App / Bot
**Priority:** Low | **Effort:** XL

Automated architecture review bot for PRs:
- Comments on PRs with new violations
- Suggests fixes inline
- Auto-approves PRs with no architectural impact
- Architecture score trend graph per PR

### Mobile Dashboard
**Priority:** Low | **Effort:** M

Mobile-responsive PWA version of the dashboard:
- Push notifications for new violations
- Architecture health score at a glance
- Team-wide violation trends

### IntelliJ / JetBrains Plugin
**Priority:** Low | **Effort:** M

Plugin for IntelliJ-based IDEs:
- Real-time violation highlighting
- Project-level architecture view
- Quick-fix suggestions

---

## 💡 Ideas (Not Yet Planned)

These are exploratory ideas that need validation before scheduling:

- **arx fmt** — Auto-format arx.yaml
- **arx schema** — Generate schema from existing projects
- **Rule hot-reload** — Edit arx.yaml without restarting server
- **Team dashboard** — Multi-user web dashboard with auth
- **Slack/Discord integration** — Post violation summaries to channels
- **arx init --detect** — Auto-detect architecture from codebase structure
- **HTML email reports** — Scheduled email reports with violation trends
