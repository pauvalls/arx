package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// SSEvent represents a single Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ch  chan SSEEvent
	ctx context.Context
}

// SSERegistry manages connected SSE clients with thread-safe operations.
type SSERegistry struct {
	mu      sync.RWMutex
	clients map[*SSEClient]struct{}
}

// NewSSERegistry creates a new SSE client registry.
func NewSSERegistry() *SSERegistry {
	return &SSERegistry{
		clients: make(map[*SSEClient]struct{}),
	}
}

// Register creates and registers a new SSE client with a buffered channel.
func (r *SSERegistry) Register(ctx context.Context) *SSEClient {
	c := &SSEClient{
		ch:  make(chan SSEEvent, 8),
		ctx: ctx,
	}

	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()

	return c
}

// Unregister removes a client from the registry and closes its channel.
// It is safe to call multiple times with the same client or with nil.
func (r *SSERegistry) Unregister(c *SSEClient) {
	if c == nil {
		return
	}

	r.mu.Lock()
	_, exists := r.clients[c]
	if exists {
		delete(r.clients, c)
		close(c.ch)
	}
	r.mu.Unlock()
}

// Broadcast sends an event to all connected clients non-blocking.
// Slow clients that have a full buffer will miss the event.
func (r *SSERegistry) Broadcast(event SSEEvent) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for c := range r.clients {
		select {
		case c.ch <- event:
		default:
			// Slow client — skip to avoid blocking
		}
	}
}

// Clients returns the number of currently registered clients.
func (r *SSERegistry) Clients() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// handleSSE handles GET /api/events — SSE endpoint.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := s.registry.Register(r.Context())
	defer s.registry.Unregister(client)

	// Send initial heartbeat: connected
	fmt.Fprintf(w, "event: heartbeat\ndata: connected\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-client.ch:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, evt.Data)
			flusher.Flush()
			heartbeat.Reset(30 * time.Second)
		case <-heartbeat.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"status\":\"alive\"}\n\n")
			flusher.Flush()
		}
	}
}

// broadcastCheckComplete collects the current server state, builds a JSON payload,
// and broadcasts it as a check_complete event to all SSE clients.
func (s *Server) broadcastCheckComplete() {
	payload := s.buildStatePayload()
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal state payload: %v\n", err)
		return
	}

	s.registry.Broadcast(SSEEvent{
		Event: "check_complete",
		Data:  string(data),
	})
}

// broadcastConfigReload broadcasts a config_reload event to all SSE clients.
func (s *Server) broadcastConfigReload() {
	s.registry.Broadcast(SSEEvent{
		Event: "config_reload",
		Data:  "",
	})
}

// statePayload is the JSON payload sent in check_complete events.
type statePayload struct {
	Violations     []domain.Violation     `json:"violations"`
	Coupling       []domain.CouplingEntry `json:"coupling"`
	DebtMetrics    domain.DebtScore       `json:"debt_metrics"`
	Metrics        Metrics                `json:"metrics"`
	Config         *domain.Config         `json:"config"`
	SeverityCounts map[string]int         `json:"severity_counts"`
}

// buildStatePayload collects thread-safe snapshots from ServerState.
func (s *Server) buildStatePayload() statePayload {
	violations := s.state.Violations()
	coupling := s.state.Coupling()
	debt := s.state.Debt()
	metrics := s.state.Metrics()
	cfg := s.state.Config()

	severityCounts := map[string]int{}
	for _, v := range violations {
		severityCounts[string(v.Severity)]++
	}

	couplingEntries := coupling.GetEntriesWithPercentage()
	if couplingEntries == nil {
		couplingEntries = []domain.CouplingEntry{}
	}

	return statePayload{
		Violations:     violations,
		Coupling:       couplingEntries,
		DebtMetrics:    debt,
		Metrics:        metrics,
		Config:         cfg,
		SeverityCounts: severityCounts,
	}
}
