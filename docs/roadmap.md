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

## ✅ v40 — v50 Roadmap (Completed)

### ✅ v40 — Language Detector Hardening
**Effort:** M

- [x] Go detector: 79% coverage (was N/A), 6 tests + fuzz (first tests ever!)
- [x] Python detector: 73.6% coverage (was 37.7%), new fuzz test
- [x] TypeScript detector: 45.4% coverage (was 19.9%), new fuzz test
- [x] All 12 detectors pass with race detector
- [x] Integration tests hardened (uses -count=1 to avoid cache issues)

### ✅ v41 — Config Strict Mode + Validation
**Effort:** S

- [x] `arx config validate --strict` fails on unknown keys
- [x] Better error messages with violation IDs and line numbers
- [x] Schema `--dry-run` shows what would change
- [x] Config upgrade command: `arx config upgrade` migrates old formats

### ✅ v42 — Dashboard Dependency Graph
**Effort:** M

- [x] Interactive SVG dependency graph with circular layout
- [x] Directional bezier arrows from coupling matrix data
- [x] Color-coded nodes: green (clean), yellow (warnings), red (errors)
- [x] Hover tooltip with incoming/outgoing/violations counts
- [x] Click node to filter violations table by layer
- [x] Arrow highlighting on hover
- [x] 9 new tests, TDD flow, zero Go backend changes

### ✅ v43 — Rule Testing Framework
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

### ✅ v44 — Multi-Project / Workspace Mode
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

### ✅ v45 — Performance Pass
**Effort:** M

- Profile-guided optimization of the detection pipeline
- Parallel detector execution tuning
- Benchmark-driven improvements (target: 2x faster detection)
- Optimize the cross-language detector (file caching)
- Add `arx check --profile` for performance breakdown

### ✅ v46 — Baseline Auto-Refresh + History
**Effort:** M

- [x] Auto-refresh baseline when violations are consistently resolved
- [x] Baseline history tracking (`.arx-baseline-history/`)
- [x] Trend visualization: "violations over time"
- [x] `arx baseline --diff` shows what changed since last baseline
- [x] Integration with `arx check --diff`

### ✅ v47 — Config Includes + Schema Generation
**Effort:** M

- [x] `!include` directive for splitting configs (recursive, cycle-detected)
- [x] `arx schema generate` — generates JSON Schema from domain.Config struct
- [x] Config composition: base config + overrides via `--override` flag
- [x] Environment variable interpolation (`$VAR`, `${VAR}`, `${VAR:-default}`)
- [x] SchemaGenerator port + reflection-based engine with enum/required support
- [x] Pipeline integration (env vars → includes → env vars → parse)
- [x] Schema drift acceptance test in domain layer

### ✅ v48 — Suggest Batch + Conflict Detection
**Effort:** M

- [x] `arx suggest --all` applies ALL fixes with smart conflict detection
- [x] Staged changes (like git add -p): review fixes before applying (y/N/s/e/q)
- [x] Rollback per-file (not just full rollback) — `arx rollback <file>`
- [x] Fix preview with `--dry-run`
- [x] Integration with `arx explain` for each fix

### ✅ v49 — Dashboard Real-Time (SSE)
**Effort:** M

- [x] Replace 5 polling fetch() calls with single EventSource (SSE) connection
- [x] SSE client registry with thread-safe non-blocking broadcast
- [x] Real-time push: `check_complete` events after every audit check
- [x] Live config reload notification via `config_reload` event
- [x] Connection status indicator (green/yellow/red dot in header)
- [x] Heartbeat every 30s with 60s timeout detection
- [x] Graceful fallback to polling when EventSource unavailable
- [x] Auto-reconnect (native EventSource behavior)

### ✅ v50 — Ultimate Quality Pass
**Effort:** L

- [x] 100% test coverage in all core packages (`internal/domain`, `internal/application`)
- [x] Fuzz tests for ALL language parsers (10 languages + expression + config)
- [x] `go test -race -count=5 ./...` — flaky test detection in CI
- [x] Performance benchmarks in CI with benchstat hard fail on >5% regression
- [x] All dogfooding violations in arx's own codebase fixed
- [x] Documentation audit: every feature reference in README
- [x] Security audit: .gitignore coverage, 0700 permissions on .arx-cache
- [x] Final v50.0 release — the most polished version of arx
- [x] Fuzz seed corpora for all 13 fuzz functions (hand-crafted from real fixtures)

---

## 🔜 v51 — v60 Roadmap (Next Generation)

### 🔲 v51 — Plugin System & Custom Detectors
**Priority:** High | **Effort:** XL | **Target:** v51.0

**Problem:** arx supports 10 languages out of the box, but every codebase has unique technology choices. Users working with Dart, Elixir, Scala, or internal DSLs cannot benefit from architecture validation.

**Solution:** A plugin system that allows third-party detectors to be authored in any language and executed as subprocesses communicating via JSON over stdout.

**Scope:**
- **Plugin Detector Protocol** — Well-defined JSON contract for subprocess-based detectors. Input: project root + layers. Output: dependencies with file, line, source, target.
- **`detectors:` config section** — New configuration block for registering external detectors alongside built-in ones:
  ```yaml
  detectors:
    - type: plugin
      name: dart-detector
      command: "dart run bin/detect.dart"
      languages: [dart]
      timeout: 30s
  ```
- **Lifecycle management** — Plugin execution with timeout, error isolation, output validation, and graceful degradation on failure.
- **Discovery protocol** — Plugins can advertise capabilities (languages, version, confidence) via `--capabilities` flag for interactive `arx init --detect`.
- **Caching** — Plugin results participate in the existing per-file SHA256 caching infrastructure transparently.
- **Documentation** — Authoring guide with examples in Go, Python, and TypeScript.

**Out of scope:** Plugin marketplace, WASM-based plugins, remote plugin execution.

**Risks:** Subprocess overhead for large projects. Mitigation: caching and optional persistent plugin daemon mode. Security: plugins run with user privileges, documented as trusted code.

**Dependencies:** None — purely additive, no existing code changes.

---

### 🔲 v52 — GitHub Integration & PR Review
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

### 🔲 v53 — arx LSP (Language Server Protocol)
**Priority:** Medium | **Effort:** L | **Target:** v53.0

**Problem:** Users discover violations only when they remember to run `arx check`. Real-time feedback while editing is orders of magnitude more valuable for enforcing architectural discipline.

**Solution:** A Language Server Protocol implementation that provides real-time diagnostics, code actions, and hover information for any editor with LSP support (VS Code, Neovim, Helix, Zed, etc.).

**Scope:**
- **`arx lsp` command** — LSP server implementing `initialize`, `textDocument/didChange`, `textDocument/diagnostic`, `textDocument/codeAction`, and `textDocument/hover`.
- **Diagnostic push** — On file save or change, run the detection pipeline for the affected file only (not full project) and push diagnostics showing:
  - Severity-mapped diagnostic level (error/warning/information/hint).
  - Exact file+line range from the violation.
  - Rule ID and explanation as diagnostic message.
- **Code actions** — For each violation, offer a "Show fix" code action that opens a diff view or applies the fix inline (using the existing FixEngine).
- **Hover provider** — Hovering over an import shows its resolved layer and any applicable rules.
- **Workspace-level diagnostics** — On project open, run full check and cache results. Serve cached diagnostics for instant feedback on subsequent opens.
- **Config file watching** — Detect `arx.yaml` changes and recompute diagnostics automatically.
- **Editor installation guides** — Scripts for VS Code (`arx skill install vscode`), Neovim (lspconfig snippet), Helix, Zed.

**Out of scope:** Inlay hints, semantic tokens, document symbols.

**Risks:** Performance of full-project check on LSP startup (mitigation: background async check with progress notification). Memory usage on large monorepos (mitigation: workspace-level caching with LRU eviction).

**Dependencies:** None — additive, leverages existing CheckService and FixEngine.

---

### 🔲 v54 — Team Dashboard & Multi-User Mode
**Priority:** Medium | **Effort:** L | **Target:** v54.0

**Problem:** The dashboard is single-user. Teams share a codebase but have no shared view of architectural health, no audit trail of who suppressed what, and no notification system when violations regress.

**Solution:** Multi-user dashboard with authentication, session management, notification webhooks, and team-wide aggregated views.

**Scope:**
- **Authentication** — Token-based auth for API endpoints. Optional GitHub OAuth for dashboard login. Multiple user roles: `admin`, `developer`, `viewer`.
- **Session management** — In-memory session store with configurable TTL. Dashboard login page with session cookie.
- **Audit trail** — Log every `arx baseline` suppress/refresh action with user identity and timestamp. API endpoint `/api/audit-log` retrieves history.
- **Team view** — Aggregated workspace view showing health across multiple projects with historical trend lines, per-team member violation introduction rate.
- **Webhook notifications** — Configurable webhook URLs per event type:
  - `violation.introduced` — New violation detected.
  - `violation.resolved` — Violation cleared.
  - `baseline.refreshed` — Baseline automatically updated.
  - `threshold.exceeded` — Violation count exceeds configured threshold.
- **Slack integration** — Pre-formatted Slack message blocks for each event type, with violation ID, file, severity, and direct link.
- **Dashboard alerts** — In-dashboard toast notifications for real-time events (SSE already supports push).

**Out of scope:** SAML/SSO, LDAP, team management UI, RBAC with custom roles.

**Risks:** Stateful sessions complicate horizontal scaling (mitigation: session persistence to `.arx-cache/`, documented single-server deployment).

**Dependencies:** v49 (SSE infrastructure reused for push notifications).

---

### 🔲 v55 — Policy as Code & WASM Rules
**Priority:** Low | **Effort:** XL | **Target:** v55.0

**Problem:** YAML-configured rules cover common patterns (cannot-depend, must-depend, circular) but complex organizational policies require arbitrary logic — cross-cutting concerns, data flow validation, custom metrics.

**Solution:** Allow rules to be authored in any language and compiled to WebAssembly, evaluated at native speed with full isolation.

**Scope:**
- **WASM runtime** — Embed a lightweight WASM runtime (wasmer-go or wazero, zero C dependencies). Each policy is a WASM module exposing a `evaluate(params) -> []Violation` function.
- **Policy definition** — New `policy` rule type:
  ```yaml
  rules:
    - id: P-01
      type: policy
      path: policies/layer-metrics.wasm
      severity: error
      description: "Validates layer dependency metrics"
  ```
- **Host API** — WASM modules receive a context object with access to:
  - All dependencies and their layers.
  - Layer count, file count, import count per layer.
  - User-defined configuration parameters.
  - Violations from previous phases (for cross-cutting checks).
- **Cache** — WASM modules are cached in memory after first load. Module bytecode cached in `.arx-cache/policies/`.
- **Pre-built policies** — Ship 3-5 reference policies:
  - `layer-balance` — Enforce minimum/maximum file counts per layer.
  - `dependency-symmetry` — Dependency direction between layers should be symmetric.
  - `no-orphaned-types` — Every exported type must be used by another layer.
  - `change-impact` — Flag high-impact layers (used by 80%+ of other layers).
  - `decay-metrics` — Architectural debt score based on violation age and severity.

**Out of scope:** WASM module marketplace, debugger, online editor.

**Risks:** WASM runtime adds a C dependency or performance overhead (mitigation: wazero is pure Go, zero CGO, ~1µs per call). Security: WASM sandbox is naturally isolated.

**Dependencies:** v51 (plugin system proves the extensibility model).

---

### 🔲 v56 — Config Migration & Versioning
**Priority:** Medium | **Effort:** M | **Target:** v56.0

**Problem:** As arx evolves, config schema changes. Users with existing configs need a safe, automated migration path — not manual diffs.

**Solution:** Versioned config schema with automatic migration tool, format upgrades, and deprecation warnings.

**Scope:**
- **Config version field** — `version` field in `arx.yaml` determines the schema version. Current is `1`. New versions bump as schema evolves.
- **`arx config migrate`** — Command that upgrades config files between versions:
  ```bash
  arx config migrate            # Auto-detect current version, migrate to latest
  arx config migrate --to v2    # Migrate to specific version
  arx config migrate --dry-run  # Show what would change
  arx config migrate --backup   # Create .arx.yaml.bak before modifying
  ```
- **Migration registry** — Programmatic migration functions keyed by `(from, to)`. Each function transforms the raw YAML nodes, preserving comments and formatting where possible.
- **Deprecation warnings** — When using a deprecated config field, warn on `arx check` with the version when it will be removed and the migration command to run.
- **Schema version endpoints** — `GET /api/config/schema` returns the active schema version and available migrations.
- **Git integration** — `arx config migrate` commits the migration automatically if in a git repo.

**Out of scope:** Online schema registry, multi-version concurrent support.

**Risks:** Comment preservation in YAML transformations (mitigation: use yaml.Node manipulation, not marshal/unmarshal round-trip).

**Dependencies:** v47 (schema generation and !include infrastructure).

---

### 🔲 v57 — Performance v2: Parallelism & Profiling
**Priority:** Medium | **Effort:** M | **Target:** v57.0

**Problem:** v45 fixed the concurrency model (no more errgroup cancellation) but didn't optimize parallel execution granularity. Large monorepos still bottleneck on sequential detector phases.

**Solution:** Fine-grained parallel execution with adaptive worker pools, shared caching, and actionable profiling.

**Scope:**
- **Adaptive worker pool** — Configurable `--jobs N` flag for detector execution. Default: `runtime.NumCPU()`. Workers are shared across detectors, not per-detector. Idle detectors don't consume workers.
- **Cross-project cache sharing** — In workspace mode, share the file content cache across projects. If `project-a` and `project-b` share a dependency, the second project skips parsing entirely.
- **Profile dashboard** — New `/api/performance` endpoint and dashboard tab showing:
  - Per-detector execution time (histogram over last 100 runs).
  - Cache hit/miss ratios.
  - Worker pool utilization (active/idle/waiting).
  - Total check time trend (sparkline over last 50 runs).
- **Benchmark comparison** — `arx check --benchmark` mode: runs the check, compares against stored baseline, reports performance delta.
- **Profile-guided optimization hints** — If a detector consistently takes >50% of check time, suggest: "Go detector is the bottleneck. Consider --jobs or file caching."

**Out of scope:** Distributed execution across multiple machines, GPU acceleration.

**Risks:** Worker pool contention on I/O-bound detectors (mitigation: separate I/O vs CPU detector pools).

**Dependencies:** v45 (caching infrastructure and benchmark suite).

---

### 🔲 v58 — Architecture Reports & Visualizations
**Priority:** Low | **Effort:** M | **Target:** v58.0

**Problem:** The dashboard and CLI provide raw violation data but not actionable, shareable architecture reports that teams can discuss in meetings or attach to RFCs.

**Solution:** Rich, exportable architecture reports with interactive diagrams, trend analysis, and narrative summaries.

**Scope:**
- **`arx report` command** — Generates a standalone HTML report with:
  - Executive summary: architecture health score, trend arrow, top 3 issues.
  - Layer dependency diagram (SVG, interactive, reusing existing graph layout).
  - Violation table (sortable, filterable, with severity badges).
  - Trend chart (violations over time from baseline history).
  - Per-layer breakdown (files, imports, dependencies, violations).
  - Rule coverage table (which rules triggered, which didn't).
  - Export to PDF (via CSS print styles).
- **Report themes** — Configurable branding: `default`, `minimal`, `executive` (summary only).
- **Scheduled reports** — `arx report --schedule "0 9 * * 1"` generates a weekly report every Monday.
- **Email delivery** — SMTP configuration for sending reports via email.
- **Report comparison** — `arx report --diff <previous>` shows side-by-side comparison of two points in time.

**Out of scope:** Interactive dashboard builder, custom report templates.

**Risks:** PDF quality from HTML print (mitigation: dedicated CSS print stylesheet with page breaks and no interactive elements).

**Dependencies:** v46 (baseline history for trend data), v49 (SSE for live data not needed for reports).

---

### 🔲 v59 — GraphQL API & Integrations
**Priority:** Low | **Effort:** XL | **Target:** v59.0

**Problem:** The REST API is simple but forces clients to fetch multiple endpoints and over-fetch data. Integrations (CI/CD, dashboards, bots) need a flexible query interface.

**Solution:** GraphQL API that exposes the full arx data model with composable queries, subscriptions for real-time events, and an integrated GraphiQL explorer.

**Scope:**
- **GraphQL endpoint** — `POST /api/graphql` with schema covering:
  - `Violation` (id, rule, file, line, severity, message, layer pair).
  - `Project` (layers, rules, detectors, dependencies).
  - `Workspace` (projects, aggregated health, trends).
  - `Baseline` (snapshots, history, trends).
  - `Performance` (timings, cache stats, benchmark results).
- **Subscriptions** — Real-time GraphQL subscriptions over SSE for violation events, config changes, check completions.
- **GraphiQL IDE** — Embedded GraphiQL explorer at `GET /api/graphql/ide` for ad-hoc queries during development.
- **Authentication** — Respects the same token/OAuth as the REST API.
- **Rate limiting** — Configurable query depth and complexity limits to prevent abusive queries.
- **Persisted queries** — Support for persisted queries (hash-based) for production CI/CD pipelines.

**Out of scope:** Federation for multi-service GraphQL, Apollo/Relay client libraries.

**Risks:** GraphQL adds significant complexity for a CLI tool's API (mitigation: keep REST API as primary, GraphQL as secondary integration point).

**Dependencies:** v54 (authentication infrastructure), v49 (SSE for subscriptions).

---

### 🔲 v60 — Editor Ecosystem: VS Code & JetBrains Plugins
**Priority:** Medium | **Effort:** XL | **Target:** v60.0

**Problem:** The LSP (v53) provides language-agnostic editor support, but native plugins offer deeper integration: gutter icons, project views, configuration UI, and one-click installation.

**Solution:** Official VS Code and JetBrains IntelliJ plugins that wrap the LSP and add IDE-specific features.

**Scope:**

**VS Code Extension (`arx-vscode`):**
- Wraps `arx lsp` as the language server.
- **Gutter decorations** — Visible icons next to lines with violations (red/green/yellow dots).
- **Architecture view** — Side panel showing layer hierarchy, dependency graph (reusing SVG renderer), and violation list grouped by layer.
- **Configuration UI** — Form-based editor for `arx.yaml` with schema validation, autocomplete, and inline documentation.
- **Command palette** — `Arx: Check project`, `Arx: Show violations`, `Arx: Apply fix`, `Arx: Baseline refresh`.
- **Status bar** — Architecture health score in the status bar, clickable to open violations panel.
- **Auto-install** — Detection of `arx.yaml` in workspace root prompts to install arx.

**JetBrains Plugin (`arx-jetbrains`):**
- Same feature set as VS Code, targeting IntelliJ IDEA and GoLand.
- **Tool window** — Architecture tool window integrated with the IDE's standard tool window layout.
- **Quick-fix** — Alt+Enter on a violation shows "Apply arx fix" action.
- **Project settings** — Architecture settings page in Preferences/Settings.
- **Inspection** — Custom IntelliJ inspection backed by arx for real-time highlighting of violations.

**Out of scope:** Sublime Text, Emacs, Vim (covered by LSP). Plugin marketplace listing (deferred to v61 1.0).

**Risks:** Maintaining two plugin codebases (mitigation: LSP protocol means the Go backend is shared — plugins are thin wrappers).

**Dependencies:** v53 (LSP provides the backend protocol both plugins consume).

---

## 📊 Release Matrix

| Release | Theme | Effort | Dependencies | Primary Value |
|---------|-------|--------|-------------|---------------|
| **v51** | Plugin System | XL | — | Extensibility |
| **v52** | GitHub PR Review | XL | v51 | Team workflow |
| **v53** | LSP | L | — | Developer experience |
| **v54** | Team Dashboard | L | v49 | Team visibility |
| **v55** | Policy as Code | XL | v51 | Enterprise readiness |
| **v56** | Config Migration | M | v47 | Stability |
| **v57** | Performance v2 | M | v45 | Speed |
| **v58** | Architecture Reports | M | v46, v49 | Communication |
| **v59** | GraphQL API | XL | v54, v49 | Integration |
| **v60** | Editor Plugins | XL | v53 | Ecosystem |

---

## 🎯 Beyond v60: The 1.0 Horizon

Before a stable 1.0 release, the following criteria must be met:

1. **Stable JSON contract** — `arx check --json` output is versioned and guarantees backward compatibility within v1.x.
2. **Config schema v1 stable** — No breaking changes without a major version bump.
3. **100% core coverage** — `internal/domain` and `internal/application` at 100% statement coverage, including git-dependent paths via interface extraction.
4. **Zero dogfooding violations** — arx's own architecture violations are eliminated.
5. **Complete documentation** — Getting started guide, conceptual guides, CLI reference, config reference, API reference, tutorials for CI/CD and workspace mode.
6. **Homebrew core** — Formula promoted from `pauvalls/homebrew-arx` to `Homebrew/homebrew-core`.
7. **GitHub Actions Marketplace** — `arx-action` published and verified.
8. **Benchmark baseline** — Published performance baseline with automatic regression detection.
9. **Deprecation policy** — Documented policy for field deprecation with minimum 3-release migration window.
