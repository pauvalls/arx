package domain

import (
	"testing"
)

func TestDetectCircularDependencies_SimpleCycle(t *testing.T) {
	t.Parallel()

	// Create a simple cycle: A -> B -> C -> A
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "B"},
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/c", ResolvedLayer: "C"},
		{SourceFile: "c.go", SourceLine: 7, ImportPath: "pkg/a", ResolvedLayer: "A"},
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
		{Name: "C", Paths: []string{"c.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle.Cycle) != 4 { // A -> B -> C -> A (4 elements, last repeats first)
		t.Errorf("Expected cycle length 4, got %d: %v", len(cycle.Cycle), cycle.Cycle)
	}

	// Verify cycle contains the right layers (order may vary due to normalization)
	hasA := false
	hasB := false
	hasC := false
	for _, layer := range cycle.Cycle[:3] {
		if layer == "A" {
			hasA = true
		}
		if layer == "B" {
			hasB = true
		}
		if layer == "C" {
			hasC = true
		}
	}

	if !hasA || !hasB || !hasC {
		t.Errorf("Cycle missing layers: A=%v, B=%v, C=%v", hasA, hasB, hasC)
	}
}

func TestDetectCircularDependencies_TwoLayerCycle(t *testing.T) {
	t.Parallel()

	// Create a two-layer cycle: A -> B -> A
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "B"},
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/a", ResolvedLayer: "A"},
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle.Cycle) != 3 { // A -> B -> A
		t.Errorf("Expected cycle length 3, got %d: %v", len(cycle.Cycle), cycle.Cycle)
	}
}

func TestDetectCircularDependencies_NoCycle(t *testing.T) {
	t.Parallel()

	// Linear dependency: A -> B -> C (no cycle)
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "B"},
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/c", ResolvedLayer: "C"},
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
		{Name: "C", Paths: []string{"c.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 0 {
		t.Errorf("Expected 0 cycles, got %d: %v", len(cycles), cycles)
	}
}

func TestDetectCircularDependencies_MultipleCycles(t *testing.T) {
	t.Parallel()

	// Create two independent cycles: A -> B -> A and C -> D -> C
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "B"},
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/a", ResolvedLayer: "A"},
		{SourceFile: "c.go", SourceLine: 7, ImportPath: "pkg/d", ResolvedLayer: "D"},
		{SourceFile: "d.go", SourceLine: 9, ImportPath: "pkg/c", ResolvedLayer: "C"},
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
		{Name: "C", Paths: []string{"c.go"}},
		{Name: "D", Paths: []string{"d.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 2 {
		t.Fatalf("Expected 2 cycles, got %d: %v", len(cycles), cycles)
	}
}

func TestDetectCircularDependencies_ComplexGraph(t *testing.T) {
	t.Parallel()

	// Create a more complex graph with one cycle
	// A -> B -> C -> B (cycle), A -> D (no cycle)
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "B"},
		{SourceFile: "a.go", SourceLine: 6, ImportPath: "pkg/d", ResolvedLayer: "D"},
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/c", ResolvedLayer: "C"},
		{SourceFile: "c.go", SourceLine: 7, ImportPath: "pkg/b", ResolvedLayer: "B"},
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
		{Name: "C", Paths: []string{"c.go"}},
		{Name: "D", Paths: []string{"d.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, got %d: %v", len(cycles), cycles)
	}

	// Verify the cycle involves B and C
	cycle := cycles[0]
	hasB := false
	hasC := false
	for _, layer := range cycle.Cycle {
		if layer == "B" {
			hasB = true
		}
		if layer == "C" {
			hasC = true
		}
	}

	if !hasB || !hasC {
		t.Errorf("Cycle should involve B and C: %v", cycle.Cycle)
	}
}

func TestDetectCircularDependencies_EmptyDependencies(t *testing.T) {
	t.Parallel()

	deps := []Dependency{}
	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 0 {
		t.Errorf("Expected 0 cycles for empty dependencies, got %d", len(cycles))
	}
}

func TestDetectCircularDependencies_UnresolvableLayers(t *testing.T) {
	t.Parallel()

	// Dependencies that can't be resolved to layers
	deps := []Dependency{
		{SourceFile: "a.go", SourceLine: 5, ImportPath: "pkg/b", ResolvedLayer: "X"}, // X doesn't exist
		{SourceFile: "b.go", SourceLine: 3, ImportPath: "pkg/a", ResolvedLayer: "Y"}, // Y doesn't exist
	}

	layers := []Layer{
		{Name: "A", Paths: []string{"a.go"}},
		{Name: "B", Paths: []string{"b.go"}},
	}

	cycles := DetectCircularDependencies(deps, layers)

	if len(cycles) != 0 {
		t.Errorf("Expected 0 cycles for unresolvable layers, got %d", len(cycles))
	}
}

func TestCreateCircularViolations(t *testing.T) {
	t.Parallel()

	cycles := []CircularDependency{
		{
			Cycle: []string{"domain", "application", "infrastructure", "domain"},
			Path:  []string{"domain/order.go:5", "application/service.go:10", "infrastructure/repo.go:15", "domain"},
		},
	}

	rule := Rule{
		ID:       "no-circular-dependencies",
		From:     "*",
		To:       []string{"*"},
		Type:     RuleTypeMustNotCircular,
		Severity: SeverityError,
	}

	violations := CreateCircularViolations(cycles, rule)

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.ID != "C-01" {
		t.Errorf("Expected violation ID C-01, got %s", v.ID)
	}

	if v.RuleID != "no-circular-dependencies" {
		t.Errorf("Expected rule ID no-circular-dependencies, got %s", v.RuleID)
	}

	if v.Severity != SeverityError {
		t.Errorf("Expected severity Error, got %s", v.Severity)
	}

	if v.SourceLayer != "domain" {
		t.Errorf("Expected source layer domain, got %s", v.SourceLayer)
	}
}

func TestNormalizeCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cycle    []string
		expected []string
	}{
		{
			name:     "simple cycle",
			cycle:    []string{"B", "C", "A", "B"},
			expected: []string{"A", "B", "C", "A"},
		},
		{
			name:     "already normalized",
			cycle:    []string{"A", "B", "C", "A"},
			expected: []string{"A", "B", "C", "A"},
		},
		{
			name:     "two elements",
			cycle:    []string{"B", "A", "B"},
			expected: []string{"A", "B", "A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normalizeCycle(tt.cycle)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Position %d: expected %s, got %s", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestSameCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "same cycle different rotation",
			a:        []string{"A", "B", "C", "A"},
			b:        []string{"B", "C", "A", "B"},
			expected: true,
		},
		{
			name:     "identical cycles",
			a:        []string{"A", "B", "C", "A"},
			b:        []string{"A", "B", "C", "A"},
			expected: true,
		},
		{
			name:     "different cycles",
			a:        []string{"A", "B", "C", "A"},
			b:        []string{"A", "C", "B", "A"},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []string{"A", "B", "A"},
			b:        []string{"A", "B", "C", "A"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sameCycle(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatCycle(t *testing.T) {
	t.Parallel()

	cycle := []string{"domain", "application", "infrastructure", "domain"}
	result := formatCycle(cycle)
	expected := "domain → application → infrastructure → domain"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestBuildCircularDependencyMessage(t *testing.T) {
	t.Parallel()

	cycle := CircularDependency{
		Cycle: []string{"domain", "application", "infrastructure", "domain"},
		Path:  []string{"domain/order.go", "application/service.go", "infrastructure/repo.go", "domain"},
	}

	message := buildCircularDependencyMessage(cycle)

	// Verify message contains key information
	if !containsString(message, "Circular dependency detected") {
		t.Error("Message should mention circular dependency")
	}

	if !containsString(message, "domain → application → infrastructure → domain") {
		t.Error("Message should show the cycle path")
	}

	if !containsString(message, "tightly coupled") {
		t.Error("Message should explain the problem")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
