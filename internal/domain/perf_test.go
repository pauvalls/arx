package domain

import (
	"sync"
	"testing"
	"time"
)

func TestPerfTimer_RecordsElapsed(t *testing.T) {
	timer := NewPerfTimer()
	time.Sleep(5 * time.Millisecond)
	elapsed := timer.Elapsed()
	if elapsed < 5*time.Millisecond {
		t.Errorf("Elapsed() = %v, want >= 5ms", elapsed)
	}
}

func TestPerfTimer_Reset(t *testing.T) {
	timer := NewPerfTimer()
	time.Sleep(5 * time.Millisecond)
	_ = timer.Elapsed()

	timer.Reset()
	time.Sleep(2 * time.Millisecond)
	elapsed := timer.Elapsed()

	if elapsed < 2*time.Millisecond {
		t.Errorf("Elapsed() after Reset = %v, want >= 2ms", elapsed)
	}
	if elapsed > 20*time.Millisecond {
		t.Errorf("Elapsed() after Reset = %v, want < 20ms (should not include pre-reset time)", elapsed)
	}
}

func TestPerfTimer_ElapsedWithoutStop(t *testing.T) {
	timer := NewPerfTimer()
	time.Sleep(3 * time.Millisecond)
	e1 := timer.Elapsed()
	time.Sleep(3 * time.Millisecond)
	e2 := timer.Elapsed()

	if e2 <= e1 {
		t.Errorf("Elapsed() should increase over time: e1=%v, e2=%v", e1, e2)
	}
}

func TestPerfTimer_ImmediateElapsed(t *testing.T) {
	timer := NewPerfTimer()
	elapsed := timer.Elapsed()
	// Should be near-zero (within a few microseconds)
	if elapsed < 0 {
		t.Error("Elapsed() should not be negative")
	}
}

func TestPerfCollector_AddPhaseAndReport(t *testing.T) {
	pc := NewPerfCollector()
	pc.AddPhase("go", 45*time.Millisecond)
	pc.AddPhase("typescript", 12*time.Millisecond)

	report := pc.Report()

	if report.Total != 57*time.Millisecond {
		t.Errorf("Report().Total = %v, want %v", report.Total, 57*time.Millisecond)
	}
	if len(report.Phases) != 2 {
		t.Fatalf("Report().Phases length = %d, want 2", len(report.Phases))
	}
	if report.Phases[0].Name != "go" || report.Phases[0].Duration != 45*time.Millisecond {
		t.Errorf("Phase[0] = %+v, want {go, 45ms}", report.Phases[0])
	}
	if report.Phases[1].Name != "typescript" || report.Phases[1].Duration != 12*time.Millisecond {
		t.Errorf("Phase[1] = %+v, want {typescript, 12ms}", report.Phases[1])
	}
}

func TestPerfCollector_ConcurrentSafety(t *testing.T) {
	pc := NewPerfCollector()
	var wg sync.WaitGroup
	n := 50

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pc.AddPhase("detector", time.Duration(i)*time.Millisecond)
		}(i)
	}
	wg.Wait()

	report := pc.Report()
	if len(report.Phases) != n {
		t.Errorf("Report().Phases length = %d, want %d", len(report.Phases), n)
	}
}

func TestPerfCollector_ReportImmutability(t *testing.T) {
	pc := NewPerfCollector()
	pc.AddPhase("go", 45*time.Millisecond)

	report1 := pc.Report()
	pc.AddPhase("typescript", 12*time.Millisecond)
	report2 := pc.Report()

	// report1 should be a snapshot captured before the second AddPhase
	if len(report1.Phases) != 1 {
		t.Errorf("Report() snapshot 1 has %d phases, want 1 (should not include later additions)", len(report1.Phases))
	}
	if len(report2.Phases) != 2 {
		t.Errorf("Report() snapshot 2 has %d phases, want 2", len(report2.Phases))
	}
}

func TestPerfCollector_EmptyReport(t *testing.T) {
	pc := NewPerfCollector()
	report := pc.Report()

	if report.Total != 0 {
		t.Errorf("Empty report Total = %v, want 0", report.Total)
	}
	if len(report.Phases) != 0 {
		t.Errorf("Empty report Phases = %d, want 0", len(report.Phases))
	}
}

func TestPerformanceReport_TotalCalculation(t *testing.T) {
	report := PerformanceReport{
		Phases: []PhaseTiming{
			{Name: "a", Duration: 10 * time.Millisecond},
			{Name: "b", Duration: 20 * time.Millisecond},
			{Name: "c", Duration: 30 * time.Millisecond},
		},
	}
	report.Total = report.calculateTotal()

	if report.Total != 60*time.Millisecond {
		t.Errorf("Total = %v, want %v", report.Total, 60*time.Millisecond)
	}
}
