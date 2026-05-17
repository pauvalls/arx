# Design: arx v0.24.0 — Embedded Web Dashboard

## Technical Approach

Add `arx server`, a long-running HTTP process that reuses the existing `CheckService` to produce architecture reports on demand. Data flows through a single in-memory cache refreshed by fsnotify + periodic timer. The dashboard is a single embedded HTML file served at `/`, with four JSON API endpoints. No external web dependencies — stdlib `net/http`, `html/template`, and `//go:embed` only.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Server layer | `internal/infrastructure/server/` | `internal/infrastructure/http/` or `cmd/arx/server.go` only | Follows existing infrastructure package pattern (`config/`, `detector/`, `output/`, `watcher/`). Keeps HTTP logic out of cmd layer. |
| Template engine | `html/template` + `//go:embed` | SPA framework, external templating | Zero dependencies, single binary, matches project stdlib-only constraint. |
| Data refresh | fsnotify + 30s timer (whichever fires first) | Timer only, or fsnotify only | fsnotify catches edits fast; timer catches non-file events (config reload, network mounts). Both together cover edge cases. |
| State storage | In-memory struct with `sync.RWMutex` | Disk cache, SQLite, Redis | Server is ephemeral; data is re-computed from source. No persistence needed between restarts. |
| Dashboard JS | Vanilla `fetch` + `setInterval` | WebSockets, SSE, HTMX | Polling every 30s is sufficient for architecture checks (which take seconds). WebSockets add complexity for no real benefit. |
| Cobra command location | `cmd/arx/server.go` (same package `main`) | Separate package | Follows existing pattern: `check.go`, `audit.go`, `baseline.go` all live in `cmd/arx/main`. |
| Check re-use | Call `CheckService` methods directly | Shell out to `arx check` subprocess | Reusing the service avoids process overhead, gives typed access to violations/coupling/debt, and shares the existing cache. |

## Data Flow

```
arx server --port 8080 --path ./my-project
    │
    ├──► Create CheckService (same wiring as arx check)
    ├──► Run initial check → store in ServerState (sync.RWMutex)
    │
    ├──► Start fsnotify watcher on project root
    │         │
    │         └──► On file event → debounce 500ms → re-run check → update state
    │
    ├──► Start 30s ticker → re-run check → update state
    │
    └──► Start http.Server
              │
              ├──► GET /          → render dashboard (html/template)
              ├──► GET /api/status    → JSON: version, uptime, last_check, violations
              ├──► GET /api/violations → JSON: []Violation
              ├──► GET /api/coupling   → JSON: CouplingMatrix entries
              └──► GET /api/debt       → JSON: DebtScore
```

```
    ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
    │  fsnotify   │────▶│              │     │  Dashboard  │
    │  + 30s timer│     │  ServerState │◀───▶│  (browser)  │
    └─────────────┘     │  (RWMutex)   │     │  fetch /api │
                        │              │     └─────────────┘
    ┌─────────────┐     └──────┬───────┘
    │ CheckService│────────────┘
    └─────────────┘
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/arx/server.go` | Create | Cobra `server` command, flag parsing (`--port`, `--path`), server lifecycle, graceful shutdown. |
| `internal/infrastructure/server/server.go` | Create | `Server` struct, `http.Server` setup, route registration, `ServerState` with `sync.RWMutex`, fsnotify + ticker loop. |
| `internal/infrastructure/server/dashboard.go` | Create | `//go:embed` for `dashboard.html`, `dashboardHandler` renders template with initial data. |
| `internal/infrastructure/server/dashboard.html` | Create | Single HTML file: CSS variables for theming, responsive layout, vanilla JS polling `/api/*` every 30s. |
| `internal/infrastructure/server/api.go` | Create | HTTP handlers for `/api/status`, `/api/violations`, `/api/coupling`, `/api/debt`. All return JSON. |
| `internal/infrastructure/server/server_test.go` | Create | Tests for route registration, state concurrency, graceful shutdown. |
| `internal/infrastructure/server/dashboard_test.go` | Create | Tests for template rendering, handler response codes, embedded asset availability. |
| `internal/infrastructure/server/api_test.go` | Create | Tests for API endpoint JSON shape, empty-state behavior, error responses. |

## Interfaces / Contracts

### ServerState (in-memory cache)

```go
type ServerState struct {
    mu          sync.RWMutex
    lastCheck   time.Time
    uptime      time.Time
    violations  []domain.Violation
    coupling    domain.CouplingMatrix
    debt        domain.DebtScore
    config      *domain.Config
    version     VersionInfo
    checkError  error
}
```

### API Response Shapes

```go
// GET /api/status
type StatusResponse struct {
    Version      string    `json:"version"`
    Uptime       string    `json:"uptime"`
    LastCheck    time.Time `json:"last_check"`
    Violations   int       `json:"violation_count"`
    DebtScore    int       `json:"debt_score"`
    CheckError   string    `json:"check_error,omitempty"`
}

// GET /api/violations → []domain.Violation (existing struct, JSON tags already present)
// GET /api/coupling   → []domain.CouplingEntry (existing struct)
// GET /api/debt       → domain.DebtScore (existing struct)
```

### Server constructor

```go
func New(port int, projectRoot string, service *application.CheckService) *Server
func (s *Server) Start() error
func (s *Server) Stop(ctx context.Context) error
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `ServerState` concurrent read/write | `go test -race` with goroutines calling Get/Set simultaneously |
| Unit | API handlers return correct JSON shape | `httptest.NewRecorder` + `json.Unmarshal` against response body |
| Unit | Empty state returns valid defaults (0 violations, empty arrays) | Initialize state without running check, assert JSON responses |
| Unit | Dashboard template renders without error | Call `dashboardHandler` with mock state, assert 200 + HTML content-type |
| Unit | Embedded assets are accessible | Verify `dashboard.html` embeds correctly, no missing files |
| Integration | `arx server` starts and responds on port | Launch server in test, `http.Get` each endpoint, verify status codes |
| Integration | fsnotify triggers re-check on file change | Touch a file in test project, assert `last_check` timestamp updates |

## Migration / Rollout

No migration required. Fully additive feature:
- `arx server` is a new subcommand; existing commands are unaffected.
- No config changes needed — works with any existing `arx.yaml`.
- Dashboard is self-contained in the binary; no external assets or build step.

## Open Questions

- [ ] Should the server support a `--bind` flag (default `127.0.0.1`) to allow binding to `0.0.0.0` for container/Docker use? **Recommendation: yes**, add `--bind` with default `127.0.0.1` for safety.
- [ ] Should the dashboard include a manual "Refresh" button in addition to auto-polling? **Recommendation: yes**, simple button that triggers an immediate `/api/*` fetch.
- [ ] Should we expose a `/api/health` endpoint for container health checks? **Recommendation: yes**, returns `{"status": "ok"}` with 200 — useful for Docker `HEALTHCHECK`.
