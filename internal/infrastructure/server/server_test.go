package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestServerState_ConcurrentReadWrite(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			violations := []domain.Violation{
				{ID: "rule-1", RuleID: "no-infra-dep", File: "test.go", Severity: domain.SeverityError},
			}
			coupling := domain.NewCouplingMatrix()
			coupling.Add("app", "domain")
			debt := domain.NewDebtScore()
			debt.AddViolation("error")
			debt.Calculate()

			state.SetCheckResult(violations, coupling, debt, nil, Metrics{}, nil)
			time.Sleep(time.Microsecond)
		}
		close(done)
	}()

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = state.Violations()
					_ = state.Coupling()
					_ = state.Debt()
					_ = state.LastCheck()
					_ = state.Version()
					_ = state.ViolationCount()
					_ = state.CheckError()
					time.Sleep(time.Microsecond)
				}
			}
		}()
	}

	wg.Wait()
}

func TestServerState_EmptyDefaults(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "0.1.0"})

	if state.ViolationCount() != 0 {
		t.Errorf("expected 0 violations, got %d", state.ViolationCount())
	}

	violations := state.Violations()
	if len(violations) != 0 {
		t.Errorf("expected empty violations slice, got %d items", len(violations))
	}

	entries := state.Coupling().FromTo
	if len(entries) != 0 {
		t.Errorf("expected empty coupling entries, got %d", len(entries))
	}

	debt := state.Debt()
	if debt.Total != 0 {
		t.Errorf("expected debt score 0, got %d", debt.Total)
	}

	if state.CheckError() != nil {
		t.Errorf("expected nil check error, got %v", state.CheckError())
	}

	v := state.Version()
	if v.Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", v.Version)
	}

	if state.Uptime().IsZero() {
		t.Error("expected non-zero uptime")
	}
}

func TestServerState_SetError(t *testing.T) {
	state := NewServerState(VersionInfo{})

	testErr := context.DeadlineExceeded
	state.SetError(testErr)

	if state.CheckError() != testErr {
		t.Errorf("expected error %v, got %v", testErr, state.CheckError())
	}

	if state.LastCheck().IsZero() {
		t.Error("expected lastCheck to be set after SetError")
	}
}

func TestServer_HandlerHealth(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
	}
}

func TestServer_HandlerStatus(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test-v1"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	srv.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Version != "test-v1" {
		t.Errorf("expected version test-v1, got %s", resp.Version)
	}
	if resp.Violations != 0 {
		t.Errorf("expected 0 violations, got %d", resp.Violations)
	}
}

func TestServer_HandlerStatusWithViolations(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
		{ID: "v2", RuleID: "r2", Severity: domain.SeverityWarning},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	srv.handleStatus(rec, req)

	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Violations != 2 {
		t.Errorf("expected 2 violations, got %d", resp.Violations)
	}
}

func TestServer_HandlerViolations(t *testing.T) {
	state := NewServerState(VersionInfo{})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Severity: domain.SeverityError},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/violations", nil)
	rec := httptest.NewRecorder()

	srv.handleViolations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result []domain.Violation
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result))
	}
	if result[0].ID != "v1" {
		t.Errorf("expected violation ID v1, got %s", result[0].ID)
	}
}

func TestServer_HandlerViolationsEmpty(t *testing.T) {
	state := NewServerState(VersionInfo{})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/violations", nil)
	rec := httptest.NewRecorder()

	srv.handleViolations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Should return empty array, not null
	var result []domain.Violation
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result == nil {
		t.Error("expected empty array, not null")
	}
}

func TestServer_HandlerCoupling(t *testing.T) {
	state := NewServerState(VersionInfo{})
	coupling := domain.NewCouplingMatrix()
	coupling.Add("app", "domain")
	coupling.Add("app", "domain")
	coupling.Add("domain", "infra")
	state.SetCheckResult(nil, coupling, domain.NewDebtScore(), nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/coupling", nil)
	rec := httptest.NewRecorder()

	srv.handleCoupling(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var entries []domain.CouplingEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 coupling entries, got %d", len(entries))
	}
}

func TestServer_HandlerCouplingEmpty(t *testing.T) {
	state := NewServerState(VersionInfo{})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/coupling", nil)
	rec := httptest.NewRecorder()

	srv.handleCoupling(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var entries []domain.CouplingEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if entries == nil {
		t.Error("expected empty array, not null")
	}
}

func TestServer_HandlerDebt(t *testing.T) {
	state := NewServerState(VersionInfo{})
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()
	state.SetCheckResult(nil, domain.NewCouplingMatrix(), debt, nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/debt", nil)
	rec := httptest.NewRecorder()

	srv.handleDebt(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result domain.DebtScore
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// 2 errors * 3 + 1 warning * 1 = 7
	if result.Total != 7 {
		t.Errorf("expected debt score 7, got %d", result.Total)
	}
}

func TestServer_HandlerStatusMethodNotAllowed(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	rec := httptest.NewRecorder()

	srv.handleStatus(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestServer_HandlerDashboard(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", ct)
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	state := NewServerState(VersionInfo{})
	srv := &Server{
		port:  0, // let OS pick a port
		bind:  "127.0.0.1",
		state: state,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify server stopped
	select {
	case err := <-errCh:
		if err != nil && err.Error() != "server failed: http: Server closed" {
			// http.ErrServerClosed is expected on graceful shutdown
			t.Logf("server exited with: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop within timeout")
	}
}

func TestServer_UnknownRouteReturns404(t *testing.T) {
	state := NewServerState(VersionInfo{})
	srv := &Server{state: state}

	// Use a mux without the catch-all "/" handler to test 404 behavior
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", srv.handleHealth)

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for unknown route, got %d", rec.Code)
	}
}

func TestServer_HandlerStatusWithSeverityBreakdown(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test-v2"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
		{ID: "v2", RuleID: "r2", Severity: domain.SeverityError},
		{ID: "v3", RuleID: "r3", Severity: domain.SeverityWarning},
		{ID: "v4", RuleID: "r4", Severity: domain.SeverityInfo},
	}
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), debt, nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	srv.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Violations != 4 {
		t.Errorf("expected 4 violations, got %d", resp.Violations)
	}
	if resp.ViolationsBySeverity["error"] != 2 {
		t.Errorf("expected 2 error violations, got %d", resp.ViolationsBySeverity["error"])
	}
	if resp.ViolationsBySeverity["warning"] != 1 {
		t.Errorf("expected 1 warning violation, got %d", resp.ViolationsBySeverity["warning"])
	}
	if resp.ViolationsBySeverity["info"] != 1 {
		t.Errorf("expected 1 info violation, got %d", resp.ViolationsBySeverity["info"])
	}
	if resp.DebtScore != 7 {
		t.Errorf("expected debt score 7, got %d", resp.DebtScore)
	}
}

func TestServer_HandlerStatusEmptyState(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test-v1"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	srv.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Violations != 0 {
		t.Errorf("expected 0 violations, got %d", resp.Violations)
	}
	if resp.ViolationsBySeverity == nil {
		t.Error("expected violations_by_severity map, got nil")
	}
	if resp.DebtScore != 0 {
		t.Errorf("expected debt score 0, got %d", resp.DebtScore)
	}
}

func TestServer_HandlerViolationsFullDataShape(t *testing.T) {
	state := NewServerState(VersionInfo{})
	violations := []domain.Violation{
		{
			ID:          "v-001",
			RuleID:      "no-infra-in-domain",
			Severity:    domain.SeverityError,
			File:        "internal/domain/service.go",
			Line:        42,
			Message:     "domain layer must not import infrastructure",
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/infra",
		},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/violations", nil)
	rec := httptest.NewRecorder()

	srv.handleViolations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result []domain.Violation
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result))
	}

	v := result[0]
	if v.ID != "v-001" {
		t.Errorf("expected ID v-001, got %s", v.ID)
	}
	if v.RuleID != "no-infra-in-domain" {
		t.Errorf("expected rule_id no-infra-in-domain, got %s", v.RuleID)
	}
	if v.Severity != domain.SeverityError {
		t.Errorf("expected severity error, got %s", v.Severity)
	}
	if v.File != "internal/domain/service.go" {
		t.Errorf("expected file internal/domain/service.go, got %s", v.File)
	}
	if v.Line != 42 {
		t.Errorf("expected line 42, got %d", v.Line)
	}
	if v.Message != "domain layer must not import infrastructure" {
		t.Errorf("expected message, got %s", v.Message)
	}
	if v.SourceLayer != "domain" {
		t.Errorf("expected source_layer domain, got %s", v.SourceLayer)
	}
	if v.TargetLayer != "infrastructure" {
		t.Errorf("expected target_layer infrastructure, got %s", v.TargetLayer)
	}
	if v.Import != "github.com/example/infra" {
		t.Errorf("expected import github.com/example/infra, got %s", v.Import)
	}
}

func TestServer_HandlerCouplingEntriesFormat(t *testing.T) {
	state := NewServerState(VersionInfo{})
	coupling := domain.NewCouplingMatrix()
	coupling.Add("application", "domain")
	coupling.Add("application", "domain")
	coupling.Add("application", "infrastructure")
	coupling.Add("domain", "infrastructure")
	state.SetCheckResult(nil, coupling, domain.NewDebtScore(), nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/coupling", nil)
	rec := httptest.NewRecorder()

	srv.handleCoupling(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var entries []domain.CouplingEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 coupling entries, got %d", len(entries))
	}

	// Find the application->domain entry
	var appDomain *domain.CouplingEntry
	for i := range entries {
		if entries[i].FromLayer == "application" && entries[i].ToLayer == "domain" {
			appDomain = &entries[i]
			break
		}
	}
	if appDomain == nil {
		t.Fatal("expected application->domain entry not found")
	}
	if appDomain.Count != 2 {
		t.Errorf("expected count 2 for application->domain, got %d", appDomain.Count)
	}
	// 2 out of 4 total = 50%
	if appDomain.Percentage != 50.0 {
		t.Errorf("expected percentage 50.0 for application->domain, got %.1f", appDomain.Percentage)
	}
}

func TestServer_HandlerDebtFullStructure(t *testing.T) {
	state := NewServerState(VersionInfo{})
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.AddViolation("warning")
	debt.AddViolation("info")
	debt.Calculate()
	debt.SetTrend(5)
	state.SetCheckResult(nil, domain.NewCouplingMatrix(), debt, nil, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/debt", nil)
	rec := httptest.NewRecorder()

	srv.handleDebt(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result domain.DebtScore
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// 3 errors * 3 + 2 warnings * 1 + 1 info * 0 = 11
	if result.Total != 11 {
		t.Errorf("expected debt score 11, got %d", result.Total)
	}
	if result.BySeverity["error"] != 3 {
		t.Errorf("expected 3 error violations, got %d", result.BySeverity["error"])
	}
	if result.BySeverity["warning"] != 2 {
		t.Errorf("expected 2 warning violations, got %d", result.BySeverity["warning"])
	}
	if result.BySeverity["info"] != 1 {
		t.Errorf("expected 1 info violation, got %d", result.BySeverity["info"])
	}
	if result.Trend != "up" {
		t.Errorf("expected trend 'up', got %s", result.Trend)
	}
	if result.TrendDelta != 5 {
		t.Errorf("expected trend_delta 5, got %d", result.TrendDelta)
	}
}

func TestServer_HandlerDebtEmptyState(t *testing.T) {
	state := NewServerState(VersionInfo{})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/debt", nil)
	rec := httptest.NewRecorder()

	srv.handleDebt(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result domain.DebtScore
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("expected debt score 0, got %d", result.Total)
	}
	// Empty state returns zero-value DebtScore (BySeverity may be nil)
}

func TestServer_AllEndpointsReturnValidJSON(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	cfg := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules: []domain.Rule{
			{
				ID:       "no-infra",
				Severity: domain.SeverityError,
				Check: domain.CheckExpr{
					Raw: "count(deps(domain, infra)) == 0",
				},
			},
		},
		Functions: map[string]string{
			"is_clean": "violations(no-infra) == 0",
		},
	}
	state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, cfg, Metrics{}, nil)

	endpoints := []string{
		"/api/health", "/api/status", "/api/violations",
		"/api/coupling", "/api/debt", "/api/metrics", "/api/config",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep, nil)
			rec := httptest.NewRecorder()

			switch ep {
			case "/api/health":
				srv.handleHealth(rec, req)
			case "/api/status":
				srv.handleStatus(rec, req)
			case "/api/violations":
				srv.handleViolations(rec, req)
			case "/api/coupling":
				srv.handleCoupling(rec, req)
			case "/api/debt":
				srv.handleDebt(rec, req)
			case "/api/metrics":
				srv.handleMetrics(rec, req)
			case "/api/config":
				srv.handleConfig(rec, req)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			// Verify body is valid JSON
			var raw json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("response body is not valid JSON: %v\nbody: %s", err, rec.Body.String())
			}
		})
	}
}

// TestServerState_ConfigGetter tests the Config() getter returns nil when unset
func TestServerState_ConfigGetter(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	if cfg := state.Config(); cfg != nil {
		t.Error("expected nil config before SetCheckResult")
	}
}

// TestServerState_ConfigAfterSet tests the Config() getter returns config after SetCheckResult
func TestServerState_ConfigAfterSet(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	cfg := &domain.Config{Version: domain.SchemaVersion{Major: 1, Minor: 0}}
	state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, cfg, Metrics{}, nil)
	if got := state.Config(); got == nil || got.Version.String() != "1.0" {
		t.Errorf("expected config with version 1.0, got %v", got)
	}
}

// TestNewServer verifies the New constructor
func TestNewServer(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := New(8080, "127.0.0.1", "/test", "", nil, state)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.port != 8080 {
		t.Errorf("expected port 8080, got %d", srv.port)
	}
	if srv.projectRoot != "/test" {
		t.Errorf("expected projectRoot /test, got %s", srv.projectRoot)
	}
}

// TestConfigPathFor verifies configPathFor returns correct path
func TestConfigPathFor(t *testing.T) {
	tests := []struct {
		root     string
		expected string
	}{
		{"/project", "/project/arx.yaml"},
		{"/project/sub", "/project/sub/arx.yaml"},
	}
	for _, tt := range tests {
		got := configPathFor(tt.root)
		if got != tt.expected {
			t.Errorf("configPathFor(%q) = %q, want %q", tt.root, got, tt.expected)
		}
	}
}

func TestServer_HandlerConfigEmpty(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()

	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if loaded, ok := result["loaded"].(bool); !ok || loaded != false {
		t.Errorf("expected loaded=false, got %v", result["loaded"])
	}
}

func TestServer_HandlerConfigWithData(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	cfg := &domain.Config{
		Version: domain.SchemaVersion{Major: 2, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules: []domain.Rule{
			{
				ID:       "r1",
				Severity: domain.SeverityError,
				Check:    domain.CheckExpr{Raw: "count(deps(domain, infra)) > 0"},
			},
		},
		Functions: map[string]string{"is_clean": "violations(r1) == 0"},
	}
	state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, cfg, Metrics{}, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()

	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if loaded, ok := result["loaded"].(bool); !ok || loaded != true {
		t.Errorf("expected loaded=true, got %v", result["loaded"])
	}
	if layers, ok := result["layers"].([]any); !ok || len(layers) != 1 || layers[0] != "domain" {
		t.Errorf("expected layers [domain], got %v", result["layers"])
	}
	if funcs, ok := result["functions"].([]any); !ok || len(funcs) != 1 || funcs[0] != "is_clean" {
		t.Errorf("expected functions [is_clean], got %v", result["functions"])
	}
}

func TestServer_HandlerConfigMethodNotAllowed(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	rec := httptest.NewRecorder()

	srv.handleConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestServer_HandlerReloadNoService(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodPost, "/api/reload", nil)
	rec := httptest.NewRecorder()

	srv.handleReload(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result["status"] != "error" {
		t.Errorf("expected status=error, got %s", result["status"])
	}
}

func TestServer_HandlerReloadWithService(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state, service: NewDefaultCheckService()}

	req := httptest.NewRequest(http.MethodPost, "/api/reload", nil)
	rec := httptest.NewRecorder()

	srv.handleReload(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestServer_HandlerReloadMethodNotAllowed(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodGet, "/api/reload", nil)
	rec := httptest.NewRecorder()

	srv.handleReload(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestIsConfigPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"arx.yaml", true},
		{"arx.yml", true},
		{"/home/user/project/arx.yaml", true},
		{"/home/user/project/arx.yml", true},
		{"/project/src/something.go", false},
		{"arx.txt", false},
		{"config.yaml", false},
		{"Arx.yaml", false},
		{"/home/user/project/.arx/arx.yaml", true},
	}
	for _, tt := range tests {
		got := isConfigPath(tt.path)
		if got != tt.expected {
			t.Errorf("isConfigPath(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

// TestNewDefaultCheckService verifies the factory creates a valid service
func TestNewDefaultCheckService(t *testing.T) {
	service := NewDefaultCheckService()
	if service == nil {
		t.Fatal("NewDefaultCheckService() returned nil")
	}
}

func TestServerState_SaveLoad(t *testing.T) {
	// Create state with violations, coupling, and debt
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Severity: domain.SeverityError},
		{ID: "v2", RuleID: "r2", File: "b.go", Severity: domain.SeverityWarning},
	}
	coupling := domain.NewCouplingMatrix()
	coupling.Add("app", "domain")
	coupling.Add("app", "domain")
	coupling.Add("domain", "infra")
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()
	state.SetCheckResult(violations, coupling, debt, nil, Metrics{}, nil)

	// Save to temp file
	tmp := t.TempDir()
	path := tmp + "/state.json"
	if err := state.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load into a fresh state
	loaded := NewServerState(VersionInfo{Version: "loaded"})
	if err := loaded.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify violations
	if loaded.ViolationCount() != 2 {
		t.Errorf("expected 2 violations, got %d", loaded.ViolationCount())
	}
	vs := loaded.Violations()
	if vs[0].ID != "v1" || vs[1].ID != "v2" {
		t.Errorf("violations mismatch: got %v", vs)
	}

	// Verify coupling
	c := loaded.Coupling()
	if c.Get("app", "domain") != 2 {
		t.Errorf("expected coupling app->domain=2, got %d", c.Get("app", "domain"))
	}
	if c.Get("domain", "infra") != 1 {
		t.Errorf("expected coupling domain->infra=1, got %d", c.Get("domain", "infra"))
	}

	// Verify debt
	d := loaded.Debt()
	if d.Total != 4 { // 1*3 + 1*1 = 4
		t.Errorf("expected debt score 4, got %d", d.Total)
	}

	// Verify lastCheck was restored
	if loaded.LastCheck().IsZero() {
		t.Error("expected lastCheck to be set after load")
	}
}

func TestServerState_SaveLoadCorrupted(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/corrupt.json"

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("{not valid json!!!}"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	state := NewServerState(VersionInfo{Version: "test"})
	err := state.LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error loading corrupt JSON, got nil")
	}
}

func TestServerState_LoadMissing(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})

	// Loading a non-existent file should return nil (graceful miss)
	err := state.LoadFromFile("/tmp/arx-nonexistent-cache-file-12345.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	// State should remain at defaults
	if state.ViolationCount() != 0 {
		t.Errorf("expected 0 violations after missing load, got %d", state.ViolationCount())
	}
	if state.CheckError() != nil {
		t.Errorf("expected nil error after missing load, got %v", state.CheckError())
	}
}

func TestMetrics_JSONRoundTrip(t *testing.T) {
	m := Metrics{
		CheckDurationMs: 342,
		FilesScanned:    156,
		TotalDeps:       892,
		DetectorsRun:    3,
		UptimeSeconds:   1800,
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal Metrics: %v", err)
	}

	var got Metrics
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal Metrics: %v", err)
	}

	if got.CheckDurationMs != m.CheckDurationMs {
		t.Errorf("check_duration_ms: got %d, want %d", got.CheckDurationMs, m.CheckDurationMs)
	}
	if got.FilesScanned != m.FilesScanned {
		t.Errorf("files_scanned: got %d, want %d", got.FilesScanned, m.FilesScanned)
	}
	if got.TotalDeps != m.TotalDeps {
		t.Errorf("total_deps: got %d, want %d", got.TotalDeps, m.TotalDeps)
	}
	if got.DetectorsRun != m.DetectorsRun {
		t.Errorf("detectors_run: got %d, want %d", got.DetectorsRun, m.DetectorsRun)
	}
}

func TestServerState_MetricsGetterSetter(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})

	// Default metrics should be zero
	m := state.Metrics()
	if m.CheckDurationMs != 0 {
		t.Errorf("expected default check_duration_ms 0, got %d", m.CheckDurationMs)
	}
	if m.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime, got %d", m.UptimeSeconds)
	}

	// Set metrics and verify
	expected := Metrics{
		CheckDurationMs: 500,
		FilesScanned:    42,
		TotalDeps:       100,
		DetectorsRun:    2,
	}
	state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, nil, expected, nil)

	got := state.Metrics()
	if got.CheckDurationMs != 500 {
		t.Errorf("check_duration_ms: got %d, want 500", got.CheckDurationMs)
	}
	if got.FilesScanned != 42 {
		t.Errorf("files_scanned: got %d, want 42", got.FilesScanned)
	}
	if got.TotalDeps != 100 {
		t.Errorf("total_deps: got %d, want 100", got.TotalDeps)
	}
	if got.DetectorsRun != 2 {
		t.Errorf("detectors_run: got %d, want 2", got.DetectorsRun)
	}
	// UptimeSeconds is computed dynamically, just verify it's non-negative
	if got.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime_seconds, got %d", got.UptimeSeconds)
	}
}

func TestServerState_MetricsThreadSafety(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m := Metrics{
				CheckDurationMs: int64(i),
				FilesScanned:    i,
				TotalDeps:       i * 2,
				DetectorsRun:    i % 5,
			}
			state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, nil, m, nil)
			time.Sleep(time.Microsecond)
		}
		close(done)
	}()

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = state.Metrics()
					time.Sleep(time.Microsecond)
				}
			}
		}()
	}

	wg.Wait()
}

func TestServer_HandlerMetrics(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	m := Metrics{
		CheckDurationMs: 342,
		FilesScanned:    156,
		TotalDeps:       892,
		DetectorsRun:    3,
	}
	state.SetCheckResult(nil, domain.CouplingMatrix{}, domain.DebtScore{}, nil, m, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rec := httptest.NewRecorder()

	srv.handleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result.CheckDurationMs != 342 {
		t.Errorf("check_duration_ms: got %d, want 342", result.CheckDurationMs)
	}
	if result.FilesScanned != 156 {
		t.Errorf("files_scanned: got %d, want 156", result.FilesScanned)
	}
	if result.TotalDeps != 892 {
		t.Errorf("total_deps: got %d, want 892", result.TotalDeps)
	}
	if result.DetectorsRun != 3 {
		t.Errorf("detectors_run: got %d, want 3", result.DetectorsRun)
	}
}

func TestServer_HandlerMetricsMethodNotAllowed(t *testing.T) {
	srv := &Server{state: NewServerState(VersionInfo{})}

	req := httptest.NewRequest(http.MethodPost, "/api/metrics", nil)
	rec := httptest.NewRecorder()

	srv.handleMetrics(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestServer_HandlerMetricsEmptyState(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rec := httptest.NewRecorder()

	srv.handleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Zero values for unset metrics (uptime is computed dynamically)
	if result.CheckDurationMs != 0 {
		t.Errorf("expected 0 check_duration_ms, got %d", result.CheckDurationMs)
	}
	if result.FilesScanned != 0 {
		t.Errorf("expected 0 files_scanned, got %d", result.FilesScanned)
	}
}

func TestHandleConfigSchema(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config/schema", srv.handleConfigSchema)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/config/schema")
	if err != nil {
		t.Fatalf("GET /api/config/schema error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var info SchemaInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.Current != "1.0" {
		t.Errorf("current = %q, want %q", info.Current, "1.0")
	}
	if len(info.Supported) == 0 {
		t.Error("supported is empty, expected at least one version")
	}
}
