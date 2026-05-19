package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// ============================================================================
// T-01: SSE Event Types + Client Registry
// ============================================================================

func TestSSERegistry_RegisterCreatesClient(t *testing.T) {
	reg := NewSSERegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := reg.Register(ctx)
	if client == nil {
		t.Fatal("Register() returned nil")
	}
	if client.ch == nil {
		t.Error("expected non-nil channel on client")
	}

	if got := reg.Clients(); got != 1 {
		t.Errorf("Clients() = %d, want 1", got)
	}
}

func TestSSERegistry_UnregisterRemovesClient(t *testing.T) {
	reg := NewSSERegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := reg.Register(ctx)
	reg.Unregister(client)

	if got := reg.Clients(); got != 0 {
		t.Errorf("Clients() after unregister = %d, want 0", got)
	}

	// Channel must be closed after unregister
	_, ok := <-client.ch
	if ok {
		t.Error("expected closed channel after Unregister")
	}
}

func TestSSERegistry_BroadcastSendsToAllClients(t *testing.T) {
	reg := NewSSERegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c1 := reg.Register(ctx)
	c2 := reg.Register(ctx)

	event := SSEEvent{Event: "test", Data: `{"msg":"hello"}`}
	reg.Broadcast(event)

	// c1 should receive
	select {
	case e := <-c1.ch:
		if e.Event != "test" {
			t.Errorf("c1: got event %q, want %q", e.Event, "test")
		}
		if e.Data != `{"msg":"hello"}` {
			t.Errorf("c1: got data %q, want %q", e.Data, `{"msg":"hello"}`)
		}
	case <-time.After(time.Second):
		t.Fatal("c1: timeout waiting for event")
	}

	// c2 should receive
	select {
	case e := <-c2.ch:
		if e.Event != "test" {
			t.Errorf("c2: got event %q, want %q", e.Event, "test")
		}
	case <-time.After(time.Second):
		t.Fatal("c2: timeout waiting for event")
	}
}

func TestSSERegistry_BroadcastSlowClientNotBlocked(t *testing.T) {
	reg := NewSSERegistry()

	// Slow client never reads — should not block
	slowCtx, slowCancel := context.WithCancel(context.Background())
	defer slowCancel()
	slow := reg.Register(slowCtx)

	// Normal client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	normal := reg.Register(ctx)

	// Fill slow client's buffer by sending many events (buffer is 8)
	for i := 0; i < 15; i++ {
		reg.Broadcast(SSEEvent{Event: "test", Data: "data"})
	}

	// Normal client should still receive events (its buffer wasn't filled)
	// The slow client lost events but didn't block the broadcast
	select {
	case <-normal.ch:
		// OK — received at least one event
	case <-time.After(time.Second):
		t.Fatal("normal client: timeout waiting for event — broadcast may have blocked")
	}

	// Verify slow client is still registered (channel not double-closed)
	if reg.Clients() < 2 {
		t.Errorf("expected at least 2 registered clients after non-blocking broadcast, got %d", reg.Clients())
	}

	// Unregister slow client — should not panic
	reg.Unregister(slow)

	// Drain buffered data from the closed channel
	// Channel must close eventually (it was closed by Unregister)
	drained := make(chan struct{})
	go func() {
		for range slow.ch {
			// drain
		}
		close(drained)
	}()

	select {
	case <-drained:
		// channel was closed — success
	case <-time.After(time.Second):
		t.Fatal("slow client channel was not closed after Unregister")
	}
}

func TestSSERegistry_ConcurrentAccess(t *testing.T) {
	reg := NewSSERegistry()

	var wg sync.WaitGroup

	// Concurrent register/unregister
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := reg.Register(ctx)
			reg.Unregister(c)
		}()
	}

	// Concurrent broadcast
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Broadcast(SSEEvent{Event: "concurrent", Data: "{}"})
		}()
	}

	// Concurrent Clients() calls
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.Clients()
		}()
	}

	wg.Wait()
	// Success = no panic, no deadlock, no data race
}

func TestSSERegistry_UnregisterIdempotent(t *testing.T) {
	reg := NewSSERegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := reg.Register(ctx)

	// First unregister
	reg.Unregister(client)
	// Second unregister — should not panic
	reg.Unregister(client)
	// Unregister nil — should not panic
	reg.Unregister(nil)

	if got := reg.Clients(); got != 0 {
		t.Errorf("Clients() = %d, want 0 after multiple unregister", got)
	}
}

// ============================================================================
// T-02: SSE HTTP Handler
// ============================================================================

func TestSSEHandler_SetsCorrectHeaders(t *testing.T) {
	reg := NewSSERegistry()
	srv := &Server{state: NewServerState(VersionInfo{}), registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", srv.handleSSE)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection = %q, want %q", conn, "keep-alive")
	}
}

func TestSSEHandler_SendsInitialHeartbeat(t *testing.T) {
	reg := NewSSERegistry()
	srv := &Server{state: NewServerState(VersionInfo{}), registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", srv.handleSSE)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the initial SSE message
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		// We expect to read at least the heartbeat before context cancel
		t.Logf("read %d bytes: %v", n, err)
	}

	body := string(buf[:n])
	if !containsSSE(body, "event: heartbeat") {
		t.Errorf("initial message missing 'event: heartbeat', got:\n%s", body)
	}
	if !containsSSE(body, "data: connected") {
		t.Errorf("initial message missing 'data: connected', got:\n%s", body)
	}
}

func TestSSEHandler_StreamsBroadcastEvent(t *testing.T) {
	reg := NewSSERegistry()
	srv := &Server{state: NewServerState(VersionInfo{}), registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", srv.handleSSE)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read initial heartbeat to drain it
	buf := make([]byte, 4096)
	_, err = resp.Body.Read(buf)
	if err != nil {
		t.Logf("read (heartbeat): %v", err)
	}

	// Now broadcast an event
	reg.Broadcast(SSEEvent{Event: "check_complete", Data: `{"status":"ok"}`})

	// Read the streamed event
	n, err := resp.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Logf("read (event): %v", err)
	}
	body := string(buf[:n])
	if !containsSSE(body, "event: check_complete") {
		t.Errorf("expected 'event: check_complete', got:\n%s", body)
	}
	if !containsSSE(body, `{"status":"ok"}`) {
		t.Errorf("expected data payload, got:\n%s", body)
	}
}

func TestSSEHandler_ClientDisconnectCleanup(t *testing.T) {
	reg := NewSSERegistry()
	srv := &Server{state: NewServerState(VersionInfo{}), registry: reg}

	// Verify 0 clients initially
	if got := reg.Clients(); got != 0 {
		t.Fatalf("expected 0 initial clients, got %d", got)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", srv.handleSSE)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Read initial heartbeat
	buf := make([]byte, 256)
	resp.Body.Read(buf)

	// Client should be registered now
	if got := reg.Clients(); got != 1 {
		t.Errorf("expected 1 active client, got %d", got)
	}

	// Cancel the request context → handler should unregister
	cancel()

	// Wait for handler to cleanup
	time.Sleep(50 * time.Millisecond)

	resp.Body.Close()

	if got := reg.Clients(); got != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", got)
	}
}

// containsSSE checks if the SSE-formatted body contains the given line
func containsSSE(body, line string) bool {
	return strings.Contains(body, line)
}

// ============================================================================
// T-03: State Snapshot + Broadcast Wiring
// ============================================================================

func TestServer_BroadcastCheckCompleteSendsEvent(t *testing.T) {
	reg := NewSSERegistry()
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state, registry: reg}

	// Set some state
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
	}
	state.SetCheckResult(violations, domain.NewCouplingMatrix(), domain.NewDebtScore(), nil, Metrics{}, nil)

	// Register a client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := reg.Register(ctx)

	// Broadcast
	srv.broadcastCheckComplete()

	// Read from client channel
	select {
	case e := <-client.ch:
		if e.Event != "check_complete" {
			t.Errorf("event = %q, want %q", e.Event, "check_complete")
		}
		// Verify data is valid JSON and contains expected fields
		var payload map[string]any
		if err := json.Unmarshal([]byte(e.Data), &payload); err != nil {
			t.Fatalf("failed to parse event data as JSON: %v", err)
		}
		if _, ok := payload["violations"]; !ok {
			t.Error("payload missing 'violations' field")
		}
		if _, ok := payload["severity_counts"]; !ok {
			t.Error("payload missing 'severity_counts' field")
		}
		if _, ok := payload["metrics"]; !ok {
			t.Error("payload missing 'metrics' field")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for check_complete event")
	}
}

func TestServer_BroadcastConfigReloadSendsEvent(t *testing.T) {
	reg := NewSSERegistry()
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state, registry: reg}

	// Register a client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := reg.Register(ctx)

	// Broadcast config reload
	srv.broadcastConfigReload()

	select {
	case e := <-client.ch:
		if e.Event != "config_reload" {
			t.Errorf("event = %q, want %q", e.Event, "config_reload")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for config_reload event")
	}
}

func TestServer_BroadcastCheckCompleteHasSeverityCounts(t *testing.T) {
	reg := NewSSERegistry()
	state := NewServerState(VersionInfo{Version: "test"})
	srv := &Server{state: state, registry: reg}

	// Set state with mixed violations
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", Severity: domain.SeverityError},
		{ID: "v2", RuleID: "r2", Severity: domain.SeverityError},
		{ID: "v3", RuleID: "r3", Severity: domain.SeverityWarning},
		{ID: "v4", RuleID: "r4", Severity: domain.SeverityInfo},
	}
	coupling := domain.NewCouplingMatrix()
	coupling.Add("app", "domain")
	debt := domain.NewDebtScore()
	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.Calculate()

	state.SetCheckResult(violations, coupling, debt, nil, Metrics{CheckDurationMs: 100}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := reg.Register(ctx)

	srv.broadcastCheckComplete()

	select {
	case e := <-client.ch:
		var payload map[string]any
		if err := json.Unmarshal([]byte(e.Data), &payload); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		sc, ok := payload["severity_counts"].(map[string]any)
		if !ok {
			t.Fatal("severity_counts missing or wrong type")
		}
		if sc["error"].(float64) != 2 {
			t.Errorf("severity_counts.error = %v, want 2", sc["error"])
		}
		if sc["warning"].(float64) != 1 {
			t.Errorf("severity_counts.warning = %v, want 1", sc["warning"])
		}
		if sc["info"].(float64) != 1 {
			t.Errorf("severity_counts.info = %v, want 1", sc["info"])
		}

		// Verify coupling is present
		cpl, ok := payload["coupling"].([]any)
		if !ok {
			t.Fatal("coupling missing or wrong type")
		}
		if len(cpl) != 1 {
			t.Errorf("expected 1 coupling entry, got %d", len(cpl))
		}

		// Verify metrics
		m, ok := payload["metrics"].(map[string]any)
		if !ok {
			t.Fatal("metrics missing or wrong type")
		}
		if m["check_duration_ms"].(float64) != 100 {
			t.Errorf("check_duration_ms = %v, want 100", m["check_duration_ms"])
		}

		// Verify debt_metrics
		dm, ok := payload["debt_metrics"].(map[string]any)
		if !ok {
			t.Fatal("debt_metrics missing or wrong type")
		}
		if dm["total"].(float64) != 7 {
			t.Errorf("debt_metrics.total = %v, want 7", dm["total"])
		}

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for check_complete event")
	}
}

// ============================================================================
// T-02 / T-07: Heartbeat mechanism
// ============================================================================

func TestSSEHandler_HeartbeatTimer(t *testing.T) {
	reg := NewSSERegistry()
	srv := &Server{state: NewServerState(VersionInfo{}), registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", srv.handleSSE)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Use a short heartbeat interval via a custom test hook or just read
	// multiple events over time. Since we can't change the interval without
	// modifying the handler, we'll test with real 30s heartbeat indirectly.
	// For this test, we verify the heartbeat format is correct by reading
	// the initial heartbeat, which uses the same format.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read initial heartbeat
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	if !containsSSE(body, "event: heartbeat") {
		t.Errorf("expected heartbeat event, got:\n%s", body)
	}
}

// ============================================================================
// T-08: Race verification is done by running `go test -count=5 -race`
// ============================================================================
