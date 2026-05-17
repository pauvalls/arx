# Tasks: arx v0.24.0 — Embedded Web Dashboard

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900–1400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (server core) → PR 2 (API) → PR 3 (dashboard) → PR 4 (integration) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Server struct, HTTP lifecycle, graceful shutdown, cobra command | PR 1 | Base = main; standalone, tests included |
| 2 | REST API handlers for /api/* endpoints | PR 2 | Base = PR 1 branch; depends on server routes |
| 3 | Embedded dashboard HTML + template rendering | PR 3 | Base = PR 2 branch; depends on server + API |
| 4 | fsnotify wiring, CheckService integration, docs | PR 4 | Base = PR 3 branch; integration + polish |

## Phase 1: Server Core

- [x] 1.1 Create `internal/infrastructure/server/state.go` — `ServerState` struct with `sync.RWMutex`, getters/setters for `lastCheck`, `violations`, `coupling`, `debt`, `config`, `version`, `checkError`; include `VersionInfo` type alias or import from `cmd/arx`
- [x] 1.2 Create `internal/infrastructure/server/server.go` — `Server` struct with `New(port int, bind string, projectRoot string, service *application.CheckService)` constructor; `Start()` launches `http.Server` on `bind:port`; `Stop(ctx)` graceful shutdown
- [x] 1.3 Create `internal/infrastructure/server/server_test.go` — Test `ServerState` concurrent read/write with `go test -race`; test `Start`/`Stop` lifecycle with `httptest`
- [x] 1.4 Create `cmd/arx/server.go` — cobra `serverCmd` with `--port` (default 8080), `--bind` (default `127.0.0.1`), `--path` (default `.`) flags; wires `CheckService` via `newCheckService()`, creates `Server`, handles SIGINT/SIGTERM for graceful shutdown; registers to `rootCmd` in `init()`

## Phase 2: REST API

- [x] 2.1 Create `internal/infrastructure/server/api.go` — HTTP handlers: `GET /api/health` → `{"status":"ok"}` 200; `GET /api/status` → `StatusResponse{version, uptime, last_check, violation_count, debt_score, check_error}`; `GET /api/violations` → `[]domain.Violation`; `GET /api/coupling` → `[]domain.CouplingEntry`; `GET /api/debt` → `domain.DebtScore`; all read from `ServerState` via `RLock`
- [x] 2.2 Create `internal/infrastructure/server/api_test.go` — Test each endpoint returns correct JSON shape via `httptest.NewRecorder`; test empty-state defaults (0 violations, empty arrays); test `/api/health` always returns 200

## Phase 3: Dashboard UI

- [x] 3.1 Create `internal/infrastructure/server/dashboard.html` — single HTML file: CSS variables for theming, responsive grid layout, sections for status/violations/coupling/debt, vanilla JS `fetch` polling `/api/*` every 30s, manual Refresh button
- [x] 3.2 Create `internal/infrastructure/server/dashboard.go` — `//go:embed dashboard.html`; `dashboardHandler` reads `ServerState`, executes `html/template` with embedded HTML, sets `Content-Type: text/html`
- [x] 3.3 Create `internal/infrastructure/server/dashboard_test.go` — Test template renders without error; test handler returns 200 + `text/html`; verify `dashboard.html` embed is accessible

## Phase 4: Integration & Polish

- [x] 4.1 Wire existing `watcher.Watcher` into `Server.Start()` — create watcher on `projectRoot`, debounce 500ms, on file event re-run check via `CheckService`, update `ServerState`; add 30s `time.Ticker` as fallback refresh trigger
- [x] 4.2 Wire `CheckService` into refresh loop — `runCheck()` method on `Server` calls `service.Load()`, `service.DetectCached()`, `service.Evaluate()`, computes `CouplingMatrix` and `DebtScore`, writes all to `ServerState`
- [x] 4.3 Create `internal/infrastructure/server/integration_test.go` — Launch server in test, `http.Get` each endpoint, verify status codes; touch a file in test project, assert `last_check` timestamp updates within debounce window
- [x] 4.4 Update `CHANGELOG.md` with v0.24.0 server feature entry; update `README.md` with `arx server` usage examples
