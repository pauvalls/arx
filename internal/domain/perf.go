package domain

import (
	"sync"
	"time"
)

// PerfTimer is a zero-allocation timing wrapper using time.Now().
// It provides elapsed time measurement without heap allocations.
type PerfTimer struct {
	start time.Time
}

// NewPerfTimer creates a new PerfTimer and starts it.
func NewPerfTimer() PerfTimer {
	return PerfTimer{start: time.Now()}
}

// Elapsed returns the time elapsed since the timer was created or last reset.
func (t *PerfTimer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// Reset resets the timer to the current time.
func (t *PerfTimer) Reset() {
	t.start = time.Now()
}

// PhaseTiming holds timing data for a single phase of a check run.
type PhaseTiming struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration_ns"`
}

// PerformanceReport holds per-phase timing data for a check run.
type PerformanceReport struct {
	Total  time.Duration `json:"total_duration_ns"`
	Phases []PhaseTiming `json:"phases"`
}

// calculateTotal sums all phase durations.
func (p PerformanceReport) calculateTotal() time.Duration {
	var total time.Duration
	for _, pt := range p.Phases {
		total += pt.Duration
	}
	return total
}

// PerfCollector collects timing data during a check run.
// It is thread-safe and safe for concurrent use.
type PerfCollector struct {
	mu     sync.Mutex
	phases []PhaseTiming
}

// NewPerfCollector creates a new PerfCollector.
func NewPerfCollector() *PerfCollector {
	return &PerfCollector{}
}

// AddPhase adds a phase timing to the collector.
// It is thread-safe and can be called from concurrent goroutines.
func (pc *PerfCollector) AddPhase(name string, duration time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.phases = append(pc.phases, PhaseTiming{Name: name, Duration: duration})
}

// Report returns a snapshot of the collected timings.
// It is thread-safe. The returned snapshot is immutable to subsequent additions.
func (pc *PerfCollector) Report() PerformanceReport {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	phases := make([]PhaseTiming, len(pc.phases))
	copy(phases, pc.phases)

	report := PerformanceReport{Phases: phases}
	report.Total = report.calculateTotal()
	return report
}
