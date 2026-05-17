package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// VersionInfo holds version information for the server.
// Re-exported from cmd/arx to avoid import cycles.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// ServerState holds the cached audit results shared between the refresh loop
// and HTTP handlers. All public access must go through the mutex-protected
// getters to ensure safe concurrent reads.
type ServerState struct {
	mu         sync.RWMutex
	uptime     time.Time
	lastCheck  time.Time
	violations []domain.Violation
	coupling   domain.CouplingMatrix
	debt       domain.DebtScore
	config     *domain.Config
	version    VersionInfo
	checkError error
}

// NewServerState creates a new ServerState with the given version info.
func NewServerState(version VersionInfo) *ServerState {
	return &ServerState{
		uptime:  time.Now(),
		version: version,
	}
}

// SetCheckResult atomically updates all check-related fields.
func (s *ServerState) SetCheckResult(violations []domain.Violation, coupling domain.CouplingMatrix, debt domain.DebtScore, cfg *domain.Config, checkErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.violations = violations
	s.coupling = coupling
	s.debt = debt
	s.config = cfg
	s.checkError = checkErr
	s.lastCheck = time.Now()
}

// SetError atomically records a check error without clearing previous results.
func (s *ServerState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkError = err
	s.lastCheck = time.Now()
}

// Uptime returns when the server started.
func (s *ServerState) Uptime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.uptime
}

// LastCheck returns the timestamp of the last successful check.
func (s *ServerState) LastCheck() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCheck
}

// Violations returns a copy of the current violations slice.
func (s *ServerState) Violations() []domain.Violation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent callers from mutating internal state
	result := make([]domain.Violation, len(s.violations))
	copy(result, s.violations)
	return result
}

// Coupling returns the current coupling matrix.
func (s *ServerState) Coupling() domain.CouplingMatrix {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.coupling
}

// Debt returns the current debt score.
func (s *ServerState) Debt() domain.DebtScore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.debt
}

// Config returns the loaded configuration.
func (s *ServerState) Config() *domain.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// Version returns the server version info.
func (s *ServerState) Version() VersionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// CheckError returns the last check error, if any.
func (s *ServerState) CheckError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkError
}

// ViolationCount returns the number of violations without allocating a slice copy.
func (s *ServerState) ViolationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.violations)
}

// CacheData represents the serializable state for persistence.
type CacheData struct {
	Violations []domain.Violation    `json:"violations"`
	Coupling   domain.CouplingMatrix `json:"coupling"`
	Debt       domain.DebtScore      `json:"debt"`
	LastCheck  time.Time             `json:"last_check"`
	Error      string                `json:"error,omitempty"`
}

// SaveToFile writes the current state to a JSON file.
func (s *ServerState) SaveToFile(path string) error {
	s.mu.RLock()
	data := CacheData{
		Violations: s.violations,
		Coupling:   s.coupling,
		Debt:       s.debt,
		LastCheck:  s.lastCheck,
	}
	if s.checkError != nil {
		data.Error = s.checkError.Error()
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, bytes, 0644)
}

// LoadFromFile reads state from a JSON file and updates the ServerState.
// Returns nil if the file does not exist (graceful miss).
// Returns an error on corrupt JSON or other read failures.
func (s *ServerState) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // graceful miss — start with empty state
		}
		return err
	}
	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}
	s.mu.Lock()
	s.violations = cache.Violations
	s.coupling = cache.Coupling
	s.debt = cache.Debt
	s.lastCheck = cache.LastCheck
	if cache.Error != "" {
		s.checkError = fmt.Errorf("%s", cache.Error)
	}
	s.mu.Unlock()
	return nil
}
