package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/infrastructure/watcher"
)

// Server is an HTTP server that serves the arx dashboard and REST API.
type Server struct {
	port          int
	bind          string
	projectRoot   string
	cachePath     string
	service       *application.CheckService
	state         *ServerState
	mu            sync.Mutex
	httpServer    *http.Server
	watcherCancel context.CancelFunc
}

// New creates a new Server with the given configuration.
func New(port int, bind string, projectRoot string, cachePath string, service *application.CheckService, state *ServerState) *Server {
	return &Server{
		port:        port,
		bind:        bind,
		projectRoot: projectRoot,
		cachePath:   cachePath,
		service:     service,
		state:       state,
	}
}

// Start launches the HTTP server and blocks until shutdown.
// It registers signal handlers for graceful shutdown on SIGINT/SIGTERM.
// A 30s ticker and file watcher trigger periodic re-checks.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/violations", s.handleViolations)
	mux.HandleFunc("/api/coupling", s.handleCoupling)
	mux.HandleFunc("/api/debt", s.handleDebt)
	mux.HandleFunc("/api/metrics", s.handleMetrics)

	// Dashboard root
	mux.HandleFunc("/", s.handleDashboard)

	addr := fmt.Sprintf("%s:%d", s.bind, s.port)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.mu.Lock()
	s.httpServer = srv
	s.mu.Unlock()

	// Create context for background refresh (ticker + watcher)
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.watcherCancel = cancel
	s.mu.Unlock()

	// Try to load cached state for instant dashboard response
	if s.cachePath != "" {
		if err := s.state.LoadFromFile(s.cachePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load cached state: %v\n", err)
		}
	}

	// Start 30s auto-refresh ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runCheck(ctx)
			}
		}
	}()

	// Start file watcher for live reload
	w, err := watcher.NewWatcher([]string{s.projectRoot}, 500*time.Millisecond)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file watcher disabled: %v\n", err)
	} else {
		go func() {
			for {
				select {
				case <-ctx.Done():
					w.Close()
					return
				case <-w.Events():
					s.runCheck(ctx)
				case err := <-w.Errors():
					fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
				}
			}
		}()
		go func() {
			if err := w.Start(ctx); err != nil && err != context.Canceled {
				fmt.Fprintf(os.Stderr, "Watcher stopped: %v\n", err)
			}
		}()
	}

	// Graceful shutdown on signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopCh
		fmt.Fprintln(os.Stderr, "\nShutting down arx server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := s.Stop(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Server shutdown error: %v\n", err)
		}
	}()

	fmt.Printf("Arx server starting on http://%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the HTTP server and stops background refresh.
func (s *Server) Stop(ctx context.Context) error {
	// Cancel background refresh (ticker + watcher)
	s.mu.Lock()
	if s.watcherCancel != nil {
		s.watcherCancel()
	}
	s.mu.Unlock()

	s.mu.Lock()
	srv := s.httpServer
	s.mu.Unlock()

	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

// handleStatus returns the current server status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	version := s.state.Version()
	checkErr := s.state.CheckError()
	violations := s.state.Violations()

	bySeverity := make(map[string]int)
	for _, v := range violations {
		bySeverity[string(v.Severity)]++
	}

	resp := StatusResponse{
		Version:              version.Version,
		Uptime:               time.Since(s.state.Uptime()).Truncate(time.Second).String(),
		LastCheck:            s.state.LastCheck(),
		Violations:           len(violations),
		ViolationsBySeverity: bySeverity,
		DebtScore:            s.state.Debt().Total,
		CheckError:           "",
	}
	if checkErr != nil {
		resp.CheckError = checkErr.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}

// StatusResponse is the JSON response for GET /api/status.
type StatusResponse struct {
	Version             string            `json:"version"`
	Uptime              string            `json:"uptime"`
	LastCheck           time.Time         `json:"last_check"`
	Violations          int               `json:"violation_count"`
	ViolationsBySeverity map[string]int   `json:"violations_by_severity"`
	DebtScore           int               `json:"debt_score"`
	CheckError          string            `json:"check_error,omitempty"`
}

// handleViolations returns the current violations list.
func (s *Server) handleViolations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, s.state.Violations())
}

// handleCoupling returns the coupling matrix entries.
func (s *Server) handleCoupling(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	matrix := s.state.Coupling()
	entries := matrix.GetEntriesWithPercentage()
	if entries == nil {
		entries = []domain.CouplingEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleDebt returns the current debt score.
func (s *Server) handleDebt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, s.state.Debt())
}

// handleMetrics returns performance metrics from the last check.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, s.state.Metrics())
}

// writeJSON marshals and writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to encode JSON response: %v\n", err)
	}
}

// NewDefaultCheckService creates a CheckService with default infrastructure wiring.
// This mirrors the wiring in cmd/arx/root.go but without a specific output format
// (the server uses JSON for all API responses).
func NewDefaultCheckService() *application.CheckService {
	reader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	reporter := output.NewJSONReporter()
	return application.NewCheckService(reader, detectors, reporter)
}

// runCheck performs a full architecture check and updates the ServerState.
// This is the method version used by the ticker and file watcher.
func (s *Server) runCheck(ctx context.Context) {
	RunCheck(ctx, s.service, s.projectRoot, s.state)
	if s.cachePath != "" {
		if err := s.state.SaveToFile(s.cachePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state cache: %v\n", err)
		}
	}
}

// RunCheck performs a full architecture check and updates the ServerState.
// This standalone version is used by cmd/arx/server.go for the initial check.
func RunCheck(ctx context.Context, service *application.CheckService, projectRoot string, state *ServerState) {
	start := time.Now()
	configPath := configPathFor(projectRoot)

	cfg, err := service.Load(configPath)
	if err != nil {
		state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, nil, Metrics{}, fmt.Errorf("failed to load config: %w", err))
		return
	}

	result, err := service.DetectWithStatus(ctx, projectRoot, cfg.Layers)
	if err != nil {
		state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, cfg, Metrics{}, fmt.Errorf("detection failed: %w", err))
		return
	}
	deps := result.Dependencies

	// Count applicable detectors
	detectorsRun := 0
	for _, st := range result.Statuses {
		if st.Applicable {
			detectorsRun++
		}
	}

	// Count unique files
	fileSet := make(map[string]struct{}, len(deps))
	for _, d := range deps {
		fileSet[d.SourceFile] = struct{}{}
	}

	violations := service.Evaluate(deps, cfg.Rules, cfg.Layers)

	// Compute coupling matrix
	calc := domain.NewCouplingCalculator()
	coupling := calc.CalculateCouplingMatrix(deps, cfg.Layers)

	// Compute debt score
	debt := domain.NewDebtScore()
	for _, v := range violations {
		debt.AddViolation(string(v.Severity))
	}
	debt.Calculate()

	metrics := Metrics{
		CheckDurationMs: time.Since(start).Milliseconds(),
		FilesScanned:    len(fileSet),
		TotalDeps:       len(deps),
		DetectorsRun:    detectorsRun,
	}

	state.SetCheckResult(violations, coupling, debt, cfg, metrics, nil)
}

// configPathFor returns the expected arx.yaml path for a project root.
func configPathFor(projectRoot string) string {
	return projectRoot + "/arx.yaml"
}
