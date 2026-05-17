package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"golang.org/x/net/html"
)

func TestDashboard_RendersValidHTML(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test-v1"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", ct)
	}

	// Parse HTML to verify it's valid
	_, err := html.Parse(strings.NewReader(rec.Body.String()))
	if err != nil {
		t.Fatalf("HTML failed to parse: %v", err)
	}
}

func TestDashboard_ShowsViolationCount(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "no-infra-dep", File: "a.go", Line: 10, Severity: domain.SeverityError},
		{ID: "v2", RuleID: "layer-violation", File: "b.go", Line: 20, Severity: domain.SeverityWarning},
		{ID: "v3", RuleID: "info-rule", File: "c.go", Line: 30, Severity: domain.SeverityInfo},
	}
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.AddViolation("info")
	debt.Calculate()
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), debt, nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Verify violation counts appear in the HTML
	if !strings.Contains(body, "1</span>") {
		t.Error("expected error count '1' in dashboard HTML")
	}
	if !strings.Contains(body, "1</span>") {
		t.Error("expected warning count '1' in dashboard HTML")
	}

	// Verify violation rows are rendered
	if !strings.Contains(body, "no-infra-dep") {
		t.Error("expected rule ID 'no-infra-dep' in dashboard HTML")
	}
	if !strings.Contains(body, "layer-violation") {
		t.Error("expected rule ID 'layer-violation' in dashboard HTML")
	}

	// Verify severity badges
	if !strings.Contains(body, `severity-badge error`) {
		t.Error("expected error severity badge in dashboard HTML")
	}
	if !strings.Contains(body, `severity-badge warning`) {
		t.Error("expected warning severity badge in dashboard HTML")
	}
	if !strings.Contains(body, `severity-badge info`) {
		t.Error("expected info severity badge in dashboard HTML")
	}
}

func TestDashboard_EmptyState(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Verify empty state message
	if !strings.Contains(body, "No violations found") {
		t.Error("expected 'No violations found' message in empty dashboard")
	}

	// Verify zero counts
	if !strings.Contains(body, "0</span>") {
		t.Error("expected zero counts in empty dashboard")
	}

	// Verify valid HTML
	_, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("HTML failed to parse for empty state: %v", err)
	}
}

func TestDashboard_WithCouplingData(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	coupling := domain.NewCouplingMatrix()
	coupling.Add("app", "domain")
	coupling.Add("app", "domain")
	coupling.Add("app", "domain")
	coupling.Add("domain", "infra")
	state.SetCheckResult(nil, coupling, domain.NewDebtScore(), nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Verify coupling data appears
	if !strings.Contains(body, "app") {
		t.Error("expected 'app' layer in coupling section")
	}
	if !strings.Contains(body, "domain") {
		t.Error("expected 'domain' layer in coupling section")
	}
}

func TestDashboard_WithDebtScore(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()
	state.SetCheckResult(nil, domain.NewCouplingMatrix(), debt, nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Debt score: 2*3 + 1*1 = 7
	if !strings.Contains(body, "7</span>") {
		t.Error("expected debt score '7' in dashboard HTML")
	}
	if !strings.Contains(body, "2</span>") {
		t.Error("expected debt errors '2' in dashboard HTML")
	}
}

func TestDashboard_ContainsRequiredSections(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	requiredSections := []string{
		"id=\"violations-section\"",
		"id=\"coupling-section\"",
		"id=\"debt-section\"",
		"summary-card errors",
		"summary-card warnings",
		"summary-card info",
		"summary-card debt",
		"severity-badge",
		"poll-status",
		"Arx Dashboard",
	}

	for _, section := range requiredSections {
		if !strings.Contains(body, section) {
			t.Errorf("expected dashboard to contain %q", section)
		}
	}
}

func TestDashboard_ContainsPollingScript(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Verify polling script is present
	if !strings.Contains(body, "/api/violations") {
		t.Error("expected polling script to fetch /api/violations")
	}
	if !strings.Contains(body, "/api/coupling") {
		t.Error("expected polling script to fetch /api/coupling")
	}
	if !strings.Contains(body, "/api/debt") {
		t.Error("expected polling script to fetch /api/debt")
	}
	if !strings.Contains(body, "setInterval") {
		t.Error("expected polling script to use setInterval")
	}
}

func TestDashboard_PrintStyles(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "@media print") {
		t.Error("expected print-friendly CSS in dashboard")
	}
}

func TestDashboard_ResponsiveCSS(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "@media (max-width") {
		t.Error("expected responsive CSS in dashboard")
	}
}

func TestDashboard_CSSVariables(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	requiredVars := []string{
		"--color-error",
		"--color-warning",
		"--color-info",
	}

	for _, v := range requiredVars {
		if !strings.Contains(body, v) {
			t.Errorf("expected CSS variable %q in dashboard", v)
		}
	}
}

func TestDashboard_LastCheckTimestamp(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Should show "Last check:" label
	if !strings.Contains(body, "Last check:") {
		t.Error("expected 'Last check:' label in dashboard")
	}

	// Should NOT show "never" since we set a result
	if strings.Contains(body, "Last check: never") {
		t.Error("expected actual timestamp, not 'never', after check result")
	}
}

func TestDashboard_LastCheckNever(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Should show "never" since no check has been performed
	if !strings.Contains(body, "Last check: never") {
		t.Error("expected 'Last check: never' when no check has been performed")
	}
}

func TestBuildDashboardData_CountsViolations(t *testing.T) {
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
		{ID: "v2", RuleID: "r2", Severity: domain.SeverityError},
		{ID: "v3", RuleID: "r3", Severity: domain.SeverityWarning},
		{ID: "v4", RuleID: "r4", Severity: domain.SeverityInfo},
	}

	data := buildDashboardData(
		violations,
		domain.NewCouplingMatrix(),
		domain.NewDebtScore(),
		time.Now(),
		VersionInfo{Version: "test"},
		time.Now(),
	)

	if data.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", data.ErrorCount)
	}
	if data.WarningCount != 1 {
		t.Errorf("expected 1 warning, got %d", data.WarningCount)
	}
	if data.InfoCount != 1 {
		t.Errorf("expected 1 info, got %d", data.InfoCount)
	}
	if len(data.Violations) != 4 {
		t.Errorf("expected 4 violations, got %d", len(data.Violations))
	}
}

func TestBuildDashboardData_CouplingEntries(t *testing.T) {
	matrix := domain.NewCouplingMatrix()
	matrix.Add("app", "domain")
	matrix.Add("app", "domain")
	matrix.Add("domain", "infra")

	data := buildDashboardData(
		nil,
		matrix,
		domain.NewDebtScore(),
		time.Now(),
		VersionInfo{},
		time.Now(),
	)

	if len(data.CouplingEntries) != 2 {
		t.Errorf("expected 2 coupling entries, got %d", len(data.CouplingEntries))
	}
}

func TestBuildDashboardData_DebtBreakdown(t *testing.T) {
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()

	data := buildDashboardData(
		nil,
		domain.NewCouplingMatrix(),
		debt,
		time.Now(),
		VersionInfo{},
		time.Now(),
	)

	if data.DebtTotal != 7 {
		t.Errorf("expected debt total 7, got %d", data.DebtTotal)
	}
	if data.DebtErrors != 2 {
		t.Errorf("expected debt errors 2, got %d", data.DebtErrors)
	}
	if data.DebtWarnings != 1 {
		t.Errorf("expected debt warnings 1, got %d", data.DebtWarnings)
	}
	if !data.HasDebt {
		t.Error("expected HasDebt to be true")
	}
}

func TestBuildDashboardData_NoDebt(t *testing.T) {
	debt := domain.NewDebtScore()

	data := buildDashboardData(
		nil,
		domain.NewCouplingMatrix(),
		debt,
		time.Now(),
		VersionInfo{},
		time.Now(),
	)

	if data.DebtTotal != 0 {
		t.Errorf("expected debt total 0, got %d", data.DebtTotal)
	}
	if data.HasDebt {
		t.Error("expected HasDebt to be false when no debt")
	}
}

func TestDashboard_ContainsFilterBar(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Severity: domain.SeverityError, SourceLayer: "app"},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	// Severity checkboxes
	if !strings.Contains(body, `id="sev-error"`) {
		t.Error("expected severity error checkbox in filter bar")
	}
	if !strings.Contains(body, `id="sev-warning"`) {
		t.Error("expected severity warning checkbox in filter bar")
	}
	if !strings.Contains(body, `id="sev-info"`) {
		t.Error("expected severity info checkbox in filter bar")
	}

	// Layer dropdown
	if !strings.Contains(body, `id="layer-filter"`) {
		t.Error("expected layer filter <select> in filter bar")
	}
	if !strings.Contains(body, "All layers") {
		t.Error("expected 'All layers' default option in layer dropdown")
	}

	// Search input
	if !strings.Contains(body, `id="search-input"`) {
		t.Error("expected search input in filter bar")
	}
	if !strings.Contains(body, `type="text"`) {
		t.Error("expected search input to be type=text")
	}
}

func TestDashboard_ContainsSortableHeaders(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Severity: domain.SeverityError},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	sortableColumns := []string{
		`data-sortable="severity"`,
		`data-sortable="rule_id"`,
		`data-sortable="file"`,
		`data-sortable="line"`,
		`data-sortable="source_layer"`,
		`data-sortable="target_layer"`,
		`data-sortable="message"`,
	}

	for _, col := range sortableColumns {
		if !strings.Contains(body, col) {
			t.Errorf("expected violations table <th> to have %q", col)
		}
	}
}

func TestDashboard_ContainsFilterSummary(t *testing.T) {
	state := NewServerState(VersionInfo{Version: "test"})
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Severity: domain.SeverityError},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, nil)

	srv := &Server{state: state}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleDashboard(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `id="filter-summary"`) {
		t.Error("expected filter-summary element in dashboard")
	}
	if !strings.Contains(body, "Clear filters") {
		t.Error("expected 'Clear filters' button in dashboard")
	}
	if !strings.Contains(body, "No violations match the current filter") {
		t.Error("expected empty filter state message in dashboard")
	}
}
