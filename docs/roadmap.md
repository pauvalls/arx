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

---

## 🔜 v0.33.0+ (Next)

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
