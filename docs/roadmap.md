# Roadmap

## ✅ v0.25.0 (Current — Dashboard Filters, State Persistence, Check Diff)

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

## 🔜 Near-term (v0.25.0 — v0.27.0)

### Dashboard Filtering & Search
**Priority:** High | **Effort:** M

Add client-side filtering to the web dashboard:
- Filter violations by severity, layer, file path
- Search by rule ID, file name, or message text
- Sortable columns in violations table
- Real-time filter without page reload

### arx check --diff Mode
**Priority:** High | **Effort:** M

Show violations introduced since last run (not just git diff):
- `arx check --diff` shows violations added/removed since previous check
- Stores previous result in `.arx-cache/last-check.json`
- Faster feedback loop than `arx diff` (no git needed)

### Performance Metrics Dashboard
**Priority:** Medium | **Effort:** S

Add performance metrics to the web dashboard:
- Check duration (ms)
- Total files scanned
- Total dependencies extracted
- Detector-level breakdown (time per language)

---

## 🔜 Medium-term (v0.28.0 — v0.30.0)

### VSCode Extension
**Priority:** Medium | **Effort:** M

VSCode extension showing violations inline in the editor:
- Highlights violating imports with squiggly lines
- Shows fix suggestions on hover
- Runs arx check on save
- Requires separate TypeScript repository

### arx suggest (Auto-fix)
**Priority:** Medium | **Effort:** M

Auto-fix recommendations for common violations:
- `arx suggest` analyzes violations and suggests code changes
- Go support first (AST-based), then TypeScript
- Shows suggested fixes as code diffs
- `--apply` flag to auto-apply (opt-in)

### Custom Rule DSL (Extended)
**Priority:** Medium | **Effort:** L

Extended expression engine with:
- More built-in functions: `all()`, `any()`, `filter()`, `map()`
- Multi-line rule checks
- Rule composition (AND/OR/NOT across rules)
- User-definable functions in arx.yaml

---

## 🔜 Long-term (v0.31.0+)

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
