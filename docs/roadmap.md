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

## ✅ v0.40 — v0.50 Roadmap (Completed)

### ✅ v0.40 — Language Detector Hardening
**Effort:** M

- [x] Go detector: 79% coverage (was N/A), 6 tests + fuzz (first tests ever!)
- [x] Python detector: 73.6% coverage (was 37.7%), new fuzz test
- [x] TypeScript detector: 45.4% coverage (was 19.9%), new fuzz test
- [x] All 12 detectors pass with race detector
- [x] Integration tests hardened (uses -count=1 to avoid cache issues)

### ✅ v0.41 — Config Strict Mode + Validation
**Effort:** S

- [x] `arx config validate --strict` fails on unknown keys
- [x] Better error messages with violation IDs and line numbers
- [x] Schema `--dry-run` shows what would change
- [x] Config upgrade command: `arx config upgrade` migrates old formats

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

### ✅ v0.45 — Performance Pass
**Effort:** M

- Profile-guided optimization of the detection pipeline
- Parallel detector execution tuning
- Benchmark-driven improvements (target: 2x faster detection)
- Optimize the cross-language detector (file caching)
- Add `arx check --profile` for performance breakdown

### ✅ v0.46 — Baseline Auto-Refresh + History
**Effort:** M

- [x] Auto-refresh baseline when violations are consistently resolved
- [x] Baseline history tracking (`.arx-baseline-history/`)
- [x] Trend visualization: "violations over time"
- [x] `arx baseline --diff` shows what changed since last baseline
- [x] Integration with `arx check --diff`

### ✅ v0.47 — Config Includes + Schema Generation
**Effort:** M

- [x] `!include` directive for splitting configs (recursive, cycle-detected)
- [x] `arx schema generate` — generates JSON Schema from domain.Config struct
- [x] Config composition: base config + overrides via `--override` flag
- [x] Environment variable interpolation (`$VAR`, `${VAR}`, `${VAR:-default}`)
- [x] SchemaGenerator port + reflection-based engine with enum/required support
- [x] Pipeline integration (env vars → includes → env vars → parse)
- [x] Schema drift acceptance test in domain layer

### ✅ v0.48 — Suggest Batch + Conflict Detection
**Effort:** M

- [x] `arx suggest --all` applies ALL fixes with smart conflict detection
- [x] Staged changes (like git add -p): review fixes before applying (y/N/s/e/q)
- [x] Rollback per-file (not just full rollback) — `arx rollback <file>`
- [x] Fix preview with `--dry-run`
- [x] Integration with `arx explain` for each fix

### ✅ v0.49 — Dashboard Real-Time (SSE)
**Effort:** M

- [x] Replace 5 polling fetch() calls with single EventSource (SSE) connection
- [x] SSE client registry with thread-safe non-blocking broadcast
- [x] Real-time push: `check_complete` events after every audit check
- [x] Live config reload notification via `config_reload` event
- [x] Connection status indicator (green/yellow/red dot in header)
- [x] Heartbeat every 30s with 60s timeout detection
- [x] Graceful fallback to polling when EventSource unavailable
- [x] Auto-reconnect (native EventSource behavior)

### ✅ v0.50 — Ultimate Quality Pass
**Effort:** L

- [x] 100% test coverage in all core packages (`internal/domain`, `internal/application`)
- [x] Fuzz tests for ALL language parsers (10 languages + expression + config)
- [x] `go test -race -count=5 ./...` — flaky test detection in CI
- [x] Performance benchmarks in CI with benchstat hard fail on >5% regression
- [x] All dogfooding violations in arx's own codebase fixed
- [x] Documentation audit: every feature reference in README
- [x] Security audit: .gitignore coverage, 0700 permissions on .arx-cache
- [x] Final v0.50.0 release — the most polished version of arx
- [x] Fuzz seed corpora for all 13 fuzz functions (hand-crafted from real fixtures)

---

## 🔜 v0.51 — v0.60 Roadmap (Next Generation)

### ✅ v0.51 — Plugin System & Custom Detectors
**Status:** Complete | **Release:** v51.0

**What was built:**
- **Plugin Detector Protocol** — JSON-based subprocess protocol for external detectors. Input: JSON on stdin with action, project_root, layers. Output: JSON on stdout with dependencies.
- **`plugins:` config section** — New configuration block for registering external detectors:
  ```yaml
  plugins:
    - name: dart-detector
      command: dart run bin/detect.dart
      languages: [dart]
      timeout: 30s
  ```
- **Plugin lifecycle** — Timeout, error isolation, non-zero exit handling, stderr capture.
- **Capability discovery** — Plugins can advertise name, languages, version via `capabilities` action.
- **3 mock test plugins** — Go-based mock plugin for testing all paths (detect-only, full, slow, error).
- **Documentation** — Authoring guide in Go, Python, and shell script.

**Implementation details:**
- `internal/domain/plugin.go` — `PluginConfig` type with validation (name regex, builtin conflict detection, timeout parsing)
- `internal/domain/protocol.go` — All request/response types (`PluginRequest`, `PluginResponse`, `PluginCapabilities`, etc.)
- `internal/infrastructure/detector/plugin/runner.go` — `RunPlugin()` with `os/exec.CommandContext` for timeout
- `internal/infrastructure/detector/plugin/detector.go` — `PluginDetector` implementing `ports.Detector`
- `internal/infrastructure/detector/registry.go` — `GetPlugins()` and `GetDetectorsForConfig()` updated
- `cmd/arx/root.go` — `newCheckService()` accepts optional config for plugin injection
- `cmd/arx/check.go`, `baseline.go`, `workspace.go`, `server.go` — Updated for plugin support
- `docs/plugins.md` — Full plugin authoring guide
- `test/testdata/plugins/mockplugin.go` — Go-based mock plugin for testing
- `test/integration/plugin_test.go` — Full round-trip integration test

**All 24 tasks completed across 4 phases. All tests pass race-clean.**

---

### 🔲 v0.52 — GitHub Integration & PR Review
**Priority:** High | **Effort:** XL | **Target:** v52.0

**Problem:** Architecture violations discovered post-merge are too late. The highest-leverage intervention point is the pull request — before bad patterns propagate.

**Solution:** A GitHub App that automatically comments on PRs with architecture diff analysis, blocking merges on critical violations while allowing warnings to pass.

**Scope:**
- **`arx pr-check` command** — CLU tool that given a PR diff and base branch, produces a structured report of new, resolved, and unchanged violations.
- **GitHub App** — Webhook receiver for `pull_request` events. Runs `arx pr-check` against the merge commit and posts a check run with annotations.
- **Diff-aware analysis** — Only flag violations introduced by the PR diff lines. Suppress pre-existing violations (already baselined).
- **Check Run API** — Proper GitHub Checks API integration with:
  - **Conclusion:** `success` (no new violations), `neutral` (warnings only), `failure` (new critical violations), `action_required` (needs human review).
  - **Annotations:** Per-violation file annotations on the exact diff lines.
  - **Summary:** Markdown summary with counts, severity breakdown, and trend data.
- **Optional auto-approve** — When all checks pass and no new violations introduced, auto-approve the PR.
- **Configuration** — Per-repo config via `.github/arx-config.yaml` with severity thresholds, auto-approve enable/disable.

**Out of scope:** Inline fix suggestions via GitHub Suggestions API, batch fixing across multiple files.

**Risks:** Rate limits on large PRs (mitigation: sampling or diff-only analysis). False positives on unrelated files (mitigation: diff filtering).

**Dependencies:** v51 (plugin system) — plugin detectors benefit from the same PR-aware pipeline.

---

### ✅ v0.53 — arx LSP (Language Server Protocol)
**Status:** Complete | **Release:** v53.0

**What was built:**
- **Stdlib-only JSON-RPC 2.0** — Hand-written protocol types (Position, Range, Diagnostic, CodeAction, Hover, etc.) with Content-Length header parsing over stdin/stdout. No LSP SDK imports.
- **`arx lsp` command** — Cobra command that wires CheckService, FixEngine, and starts the LSP server. Handles SIGINT/SIGTERM for graceful shutdown.
- **Server state machine** — Initialize → (open/change/close) → Shutdown → Exit lifecycle with proper error codes (-32002 before init, -32600 after shutdown).
- **Document synchronization** — didOpen/didChange/didClose with in-memory document cache.
- **Diagnostic types** — Violation → Diagnostic mapping with severity conversion (error→1, warning→2, info→3, hint→4), source "arx", rule ID as code.
- **Code actions** — Wraps FixEngine.SuggestFix per diagnostic, returns quickfix CodeAction with arx.fix command.
- **Hover provider** — Scans import lines, resolves layer via Layer.MatchesPath, returns markdown with layer name and applicable rules.
- **Config file watching** — Handles workspace/didChangeWatchedFiles for arx.yaml changes.
- **Push diagnostics** — textDocument/publishDiagnostics notification support via PushDiagnostics().
- **Integration test** — Full LSP lifecycle test as subprocess: initialize → didOpen → codeAction → hover → shutdown → exit.
- **All 8 tasks completed** across 5 phases. All tests pass race-clean.

**Implementation details:**
- `internal/infrastructure/lsp/protocol.go` — 350+ lines of LSP + JSON-RPC types, ReadMessage/WriteMessage
- `internal/infrastructure/lsp/server.go` — Server state, initialize/didOpen/didChange/didClose/shutdown/exit
- `internal/infrastructure/lsp/handler.go` — Message dispatcher, method routing, PushDiagnostics
- `internal/infrastructure/lsp/diagnostics.go` — Violation→Diagnostic mapping, severity conversion
- `internal/infrastructure/lsp/codeaction.go` — ComputeCodeActions wrapping FixEngine
- `internal/infrastructure/lsp/hover.go` — ComputeHover, import path extraction, layer resolution
- `cmd/arx/lsp.go` — Cobra command wiring CheckService + FixEngine + Server
- `test/integration/lsp_test.go` — Full lifecycle subprocess integration test

---

### ✅ v0.55 — Policy as Code & WASM Rules
**Priority:** Low | **Effort:** XL | **Completed:** v55.0

**Problem:** YAML-configured rules cover common patterns (cannot-depend, must-depend, circular) but complex organizational policies require arbitrary logic — cross-cutting concerns, data flow validation, custom metrics.

**Solution:** Allow rules to be authored in any language and compiled to WebAssembly, evaluated at native speed with full isolation.

**Scope:**
- ✅ **WASM runtime** — Embedded wazero (pure Go, zero CGO). Each policy is a WASM module exposing an `evaluate() -> i32` function via wazero's runtime.
- ✅ **Policy definition** — New `wasm` field on `Rule`:
  ```yaml
  rules:
    - id: P-01
      wasm:
        path: policies/layer-balance.wasm
        params: { min: 3, max: 8 }
      severity: error
  ```
- ✅ **Host API** — WASM modules import 5 host functions from the `arx` module:
  - `arx_deps() -> i32` — total dependency count.
  - `arx_layer_count() -> i32` — configured layer count.
  - `arx_violation_count() -> i32` — violations from previous phases.
  - `arx_get_dep(idx) -> i64` — dep data as JSON in guest memory.
  - `arx_emit_violation(ptr, len) -> i32` — emit violation JSON from guest.
- ✅ **Cache** — SHA-256 keyed disk cache in `.arx-cache/policies/<hash>/wasm.bin`. In-memory cache via `sync.RWMutex` map. Manager provides singleton evaluators per WASM path.
- ✅ **EvaluateRules Phase 4** — WASM policies evaluated after templates and circular checks. Graceful degradation on errors and timeouts (5s default).
- ✅ **Pre-built reference policies** (TinyGo source + pre-built .wasm):
  - `layer-balance` — Checks deps/layer ratio.
  - `dependency-symmetry` — Checks asymmetry via violations/deps ratio.
  - `no-leak` — Flags deps without any violations.

**Out of scope:** WASM module marketplace, debugger, online editor.

**Risks:** WASM runtime adds a C dependency or performance overhead (mitigation: wazero is pure Go, zero CGO, ~1µs per call). Security: WASM sandbox is naturally isolated.

**Dependencies:** v51 (plugin system proves the extensibility model).

---

---

## 🎯 v0.56 — v0.60: The Road to 1.0

No más features nuevas. El producto está completo. Estas 5 releases son exclusivamente **pulido, estabilidad, y ecosistema** para llegar a una **v1.0.0** sólida.

| Release | Enfoque | ¿Por qué? |
|---------|---------|-----------|
| **v0.56** | Quality Hardening | Cerrar los últimos gaps |
| **v0.57** | Documentation & Onboarding | Que cualquiera pueda arrancar |
| **v0.58** | Config Migration & Versioning | ✅ Done |
| **v0.59** | Performance Polish | ✅ Done |
| **v0.60** | 1.0.0 Release | Homebrew core, Marketplace, contrato |

---

### 🔲 v0.56 — Quality Hardening
**Priority:** 🔴 Alta | **Effort:** M

**Qué:** Cerrar los últimos gaps de calidad antes de congelar el contrato.

**Scope:**
- **100% coverage en domain + application** — Extraer interfaces para los paths que dependen de git (GitClient interface). Los `if err != nil` sin test hoy son porque no podés mockear `exec.Command`. Con una interfaz, sí.
- **Fix C-01 en registry.go** — La última violación de dogfooding que quedó enmascarada. Extraer la lógica compartida a `ports` o reestructurar el registry.
- **Fuzz test corpora final** — Pasar los 13 fuzz tests con `-count=1000` cada uno, sin crashes. Congelar los corpora en el repo.
- **Flaky test pass** — `go test -count=10 -race ./...` pasa sin failures 10/10 veces.
- **Race detector hardening** — `go test -race -count=5 ./...` pasa en CI como gate obligatorio.
- **WASM 1MB size limit** — Implementar el límite de tamaño de módulo que estaba en el spec de v55 pero no se implementó.

**Criterio de éxito:** `go test -count=10 -race ./...` pasa siempre. Cobertura 100% en domain + application. 0 dogfooding violations.

---

### ✅ v0.57 — Documentation & Onboarding
**Status:** Complete | **Release:** v57.0

**What was built:**
- **Quickstart (5 minutes)** — `docs/quickstart.md` — install, init, check, explain, baseline, next steps. Zero assumptions, real 5-minute experience.
- **Conceptual guides** — 5 standalone guides under `docs/guides/`:
  - **Layers & Rules** — How layers work, path globs (`**`, `*`), rule types (Cannot/Must/Can/MustNotCircular), severity levels, overrides, excludes.
  - **Detectors** — Detection pipeline (Detect → ExtractImports), 10 built-in languages, performance profiling (`--profile`), plugin system, cross-language detection.
  - **Expression DSL** — Syntax reference, 12 built-in functions (count, deps, layers, has_circular, files, ratio, violations, threshold, all, any, filter, map), filter predicates, user-defined functions, real-world examples.
  - **WASM Policies** — Wazero-based WASM evaluation, host API (5 functions), 3 reference policies (layer-balance, dependency-symmetry, no-leak), authoring guide in TinyGo/Rust/C, building with `make wasm-policies`.
  - **Workspace Mode** — arx-workspace.yaml format, shared config + per-project overrides (shallow merge), error isolation, aggregated reports, JSON output for CI.
- **CLI reference** — `docs/reference/cli.md` — Every command documented with name, usage, flags table, examples, exit codes. 25+ commands covered.
- **Config reference** — `docs/reference/config.md` — Every field in arx.yaml documented with type, required, default, description, example. Complete example config.
- **API reference** — `docs/reference/api.md` — REST endpoints (9 endpoints), SSE events (check_complete, config_reload, heartbeat), GitHub webhook (HMAC-SHA256, pull_request events), LSP protocol (JSON-RPC 2.0, 8 methods, error codes).
- **Tutorials** — 4 tutorials under `docs/tutorials/`:
  - **CI/CD** — GitHub Actions (basic, baseline, SARIF, annotations, arx-action) + GitLab CI + pre-commit hook.
  - **Workspace Monorepo** — Step-by-step: create config, per-project overrides, glob patterns, legacy project handling.
  - **Custom Plugin** — Full Dart detector in Go with JSON protocol, alternative Python implementation, build + register + test.
  - **GitHub App** — Create app, configure webhook, PR checks, auto-approve, CI integration.
- **Editor setup** — 4 guides under `docs/editors/`:
  - **VS Code** — settings.json, keybindings, tasks.
  - **Neovim** — lspconfig Lua snippet, keybindings, telescope integration.
  - **Helix** — languages.toml configuration.
  - **Zed** — settings.json configuration.
- **FAQ** — `docs/faq.md` — 15+ questions: architectures supported, custom rules, language detection, file exclusion, suppress vs baseline, CI usage, monorepo, comparison to other tools, circular deps, updating arx.
- **README update** — Install section at top, quick example (3 lines), link to quickstart, features table (16 features), badges (build, coverage, go version, license).
- **Roadmap update** — v0.57 marked complete, scope and criteria updated with implementation details.

**Implementation details:**
- All docs are standalone markdown under `docs/` — no external wiki dependency
- CLI ref hand-written from Cobra command definitions (all flags verified against source)
- Config ref extracted from `internal/domain/config.go`, `arx-schema.json`, and rule/workspace/plugin/wasm domain types
- API ref extracted from `internal/infrastructure/server/server.go`, `sse.go`, and `internal/infrastructure/lsp/`
- Tutorials tested conceptually against real CLI behavior
- 14 new documentation files created, 2 existing files updated (README.md, roadmap.md)
- All tests pass with `go test -count=1 -race ./...`

**Success criteria:**
- ✅ New user can go from zero to `arx check` in ≤5 minutes (quickstart covers curl → init → check → explain → baseline)
- ✅ Every CLI command documented with flags, exit codes, and examples (25+ commands in cli.md)
- ✅ Every arx.yaml field documented in a single reference page (config.md with complete example)
- ✅ All REST, SSE, and LSP endpoints documented with request/response schemas (api.md)
- ✅ CI/CD tutorial produces working workflow file (ci-cd.md with GitHub Actions + GitLab)
- ✅ 4 tutorials complete: CI/CD, workspace, plugins, GitHub App
- ✅ 4 editor guides complete: VS Code, Neovim, Helix, Zed
- ✅ FAQ covers 15+ common questions

---

### ✅ v0.58 — Config Migration & Versioning
**Priority:** 🟡 Media | **Effort:** M

**Qué:** Garantizar que nadie se rompe cuando el schema evoluciona.

**Scope:**
- **`arx config migrate`** — Comando que migra `arx.yaml` entre versiones de schema. Soporta `--to`, `--dry-run`, `--backup`.
- **Migration registry** — Funciones de migración programáticas keyeadas por `(from, to)`. Transforman nodos YAML preservando comentarios.
- **Deprecation warnings** — Cuando se usa un campo deprecado, `arx check` warn ea con la versión en que se va a eliminar y el comando de migración.
- **Schema version endpoint** — `GET /api/config/schema` devuelve la versión activa y las migraciones disponibles.
- **Schema freeze** — El schema v1 queda congelado. Cualquier cambio breaking requiere schema v2 con su migración desde v1.
- **Output contract freeze** — `arx check --json` output se versiona. Se congela v1. Tests que rompen si cambia sin bumpear versión.

**Criterio de éxito:** Un usuario puede migrar de schema v1 a v2 con un solo comando, sin perder datos ni comentarios.

---

### ✅ v0.59 — Performance Polish
**Priority:** 🟡 Media | **Effort:** S

**Qué:** Publicar baseline de rendimiento, optimizar lo que queda.

**Scope:**
- **Benchmark baseline público** — El `.bench-baseline` se publica como artefacto de CI. Cada release compara contra el anterior.
- **Benchmark dashboard** — Tabla en el README o en una GitHub Page mostrando evolución de tiempos por release.
- **`--jobs` flag** — Número configurable de workers para detección paralela.
- **Cross-project cache sharing** — En workspace mode, compartir el file content cache entre proyectos.
- **Profile output en JSON** — `arx check --profile --json` incluye per-detector timing en el output JSON.

**Criterio de éxito:** `go test -bench=. -benchmem ./internal/application/` se publica en CI. Cualquier regresión >5% bloquea el merge.

---

### 🔲 v0.60 — 1.0.0 Release
**Priority:** 🔴 Alta | **Effort:** M

**Qué:** La release final. Contrato congelado, ecosistema completo, ready for production.

**Scope:**
- **Contract freeze** — JSON output v1, config schema v1, deprecation policy publicada. Cualquier cambio breaking requiere v2.0.0.
- **Homebrew core** — Formula promovida de `pauvalls/homebrew-arx` a `Homebrew/homebrew-core`. Pasar el review de Homebrew (exigente: build reproducible, tests, sin red en build).
- **GitHub Actions Marketplace** — `arx-action` publicada y verificada. Con screenshot, descripción, ejemplos.
- **Deprecation policy documentada** — Los campos deprecados tienen mínimo 3 releases de ventana antes de eliminarse.
- **Security audit final** — `gitleaks` en CI, permisos de archivos correctos, sin secretos hardcodeados.
- **v1.0.0 tag** — Release final con changelog completo, release notes en GitHub, anuncio.

**Criterio de éxito:** `v1.0.0` taggeada, Homebrew core aceptado, Action en el Marketplace, documentación completa.

---

## 📊 Resumen del Camino a 1.0

| Release | Tema | Esfuerzo | Depende de | Criterio clave |
|---------|------|----------|------------|----------------|
| **v0.56** | Quality Hardening | M | — | 100% coverage, 0 dogfooding |
| **v0.57** | Documentation | M | — | Onboarding ≤5 min |
| **v0.58** | Config Migration | M | v47 | Schema freeze + migración |
| **v0.59** | Performance Polish | S | v45 | Benchmark público |
| **v0.60** | **1.0.0 Release** | M | v56-v59 | Homebrew core + Marketplace |

## 🎯 Checklist 1.0.0

- [ ] `internal/domain` + `internal/application` → 100% cobertura
- [ ] 0 dogfooding violations (fix C-01 en registry.go)
- [ ] `go test -count=10 -race ./...` → siempre verde
- [ ] Fuzz tests 13/13 pasan `-count=1000`
- [ ] JSON output v1 congelado (tests rompen si cambia)
- [ ] Config schema v1 congelado (migración v1→v2 lista)
- [ ] Deprecation policy publicada
- [x] Quickstart ≤5 minutos documentado y verificado
- [x] CLI reference + config reference + API reference completas
- [x] Tutoriales: CI/CD, workspace, plugins, GitHub App
- [ ] Benchmark baseline público con CI guard
- [ ] Homebrew core aceptado
- [ ] GitHub Actions Marketplace publicado
- [ ] Security audit: gitleaks + permisos + sin secrets
- [ ] `v1.0.0` taggeada y releaseada
