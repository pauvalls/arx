package domain

import (
	"strings"
	"testing"
)

// TestCalculateCoupling_Basic tests basic coupling matrix calculation
func TestCalculateCoupling_Basic(t *testing.T) {
	// Create test layers
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "application", Paths: []string{"application/"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/"}},
	}

	// Create test dependencies
	dependencies := []Dependency{
		{SourceFile: "application/service.go", ResolvedLayer: "domain"},
		{SourceFile: "application/service.go", ResolvedLayer: "domain"},
		{SourceFile: "application/repo.go", ResolvedLayer: "infrastructure"},
		{SourceFile: "infrastructure/db.go", ResolvedLayer: "domain"},
	}

	// Calculate coupling
	calc := NewCouplingCalculator()
	matrix := calc.CalculateCouplingMatrix(dependencies, layers)

	// Verify counts
	if matrix.Get("application", "domain") != 2 {
		t.Errorf("Expected application->domain count = 2, got %d", matrix.Get("application", "domain"))
	}

	if matrix.Get("application", "infrastructure") != 1 {
		t.Errorf("Expected application->infrastructure count = 1, got %d", matrix.Get("application", "infrastructure"))
	}

	if matrix.Get("infrastructure", "domain") != 1 {
		t.Errorf("Expected infrastructure->domain count = 1, got %d", matrix.Get("infrastructure", "domain"))
	}

	// Verify total count
	total := matrix.Count()
	if total != 4 {
		t.Errorf("Expected total count = 4, got %d", total)
	}
}

// TestCalculateCoupling_MultipleLayers tests coupling with many layers
func TestCalculateCoupling_MultipleLayers(t *testing.T) {
	// Create 4-layer architecture
	layers := []Layer{
		{Name: "presentation", Paths: []string{"presentation/"}},
		{Name: "application", Paths: []string{"application/"}},
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/"}},
	}

	// Create dependencies across all layers
	dependencies := []Dependency{
		// Presentation layer dependencies
		{SourceFile: "presentation/handler.go", ResolvedLayer: "application"},
		{SourceFile: "presentation/handler.go", ResolvedLayer: "application"},
		{SourceFile: "presentation/handler.go", ResolvedLayer: "domain"},

		// Application layer dependencies
		{SourceFile: "application/service.go", ResolvedLayer: "domain"},
		{SourceFile: "application/service.go", ResolvedLayer: "domain"},
		{SourceFile: "application/repo.go", ResolvedLayer: "infrastructure"},

		// Domain layer dependencies (should be minimal in clean architecture)
		{SourceFile: "domain/entity.go", ResolvedLayer: "domain"}, // Self-dependency

		// Infrastructure layer dependencies
		{SourceFile: "infrastructure/db.go", ResolvedLayer: "domain"},
		{SourceFile: "infrastructure/cache.go", ResolvedLayer: "domain"},
	}

	calc := NewCouplingCalculator()
	matrix := calc.CalculateCouplingMatrix(dependencies, layers)

	// Verify presentation layer coupling
	if matrix.Get("presentation", "application") != 2 {
		t.Errorf("Expected presentation->application = 2, got %d", matrix.Get("presentation", "application"))
	}
	if matrix.Get("presentation", "domain") != 1 {
		t.Errorf("Expected presentation->domain = 1, got %d", matrix.Get("presentation", "domain"))
	}

	// Verify application layer coupling
	if matrix.Get("application", "domain") != 2 {
		t.Errorf("Expected application->domain = 2, got %d", matrix.Get("application", "domain"))
	}
	if matrix.Get("application", "infrastructure") != 1 {
		t.Errorf("Expected application->infrastructure = 1, got %d", matrix.Get("application", "infrastructure"))
	}

	// Verify infrastructure layer coupling
	if matrix.Get("infrastructure", "domain") != 2 {
		t.Errorf("Expected infrastructure->domain = 2, got %d", matrix.Get("infrastructure", "domain"))
	}

	// Total should be 9
	if matrix.Count() != 9 {
		t.Errorf("Expected total = 9, got %d", matrix.Count())
	}
}

// TestCalculateCoupling_NoDependencies tests empty dependency list
func TestCalculateCoupling_NoDependencies(t *testing.T) {
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "application", Paths: []string{"application/"}},
	}

	dependencies := []Dependency{}

	calc := NewCouplingCalculator()
	matrix := calc.CalculateCouplingMatrix(dependencies, layers)

	// Matrix should be empty
	if matrix.Count() != 0 {
		t.Errorf("Expected count = 0 for empty dependencies, got %d", matrix.Count())
	}

	// GetEntriesWithPercentage should return empty slice
	entries := matrix.GetEntriesWithPercentage()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty matrix, got %d", len(entries))
	}
}

// TestCouplingMatrix_ASCII tests ASCII table rendering
func TestCouplingMatrix_ASCII(t *testing.T) {
	matrix := NewCouplingMatrix()
	matrix.Add("application", "domain")
	matrix.Add("application", "domain")
	matrix.Add("application", "infrastructure")
	matrix.Add("infrastructure", "domain")

	// Test with default options
	ascii := matrix.ToASCII(nil)

	// Verify table structure
	if !strings.Contains(ascii, "+") {
		t.Error("ASCII table should contain '+' border characters")
	}
	if !strings.Contains(ascii, "|") {
		t.Error("ASCII table should contain '|' column separators")
	}
	if !strings.Contains(ascii, "From") {
		t.Error("ASCII table should contain 'From' header")
	}
	if !strings.Contains(ascii, "To") {
		t.Error("ASCII table should contain 'To' header")
	}
	if !strings.Contains(ascii, "Count") {
		t.Error("ASCII table should contain 'Count' header")
	}
	if !strings.Contains(ascii, "Pct") {
		t.Error("ASCII table should contain 'Pct' header")
	}

	// Verify data is present
	if !strings.Contains(ascii, "application") {
		t.Error("ASCII table should contain 'application' layer")
	}
	if !strings.Contains(ascii, "domain") {
		t.Error("ASCII table should contain 'domain' layer")
	}

	// Test with custom options
	opts := &ASCIIOptions{
		ShowPercentage: true,
		ShowCount:      true,
		Colorize:       false,
	}
	asciiCustom := matrix.ToASCII(opts)

	if !strings.Contains(asciiCustom, "Count") {
		t.Error("Custom ASCII table should contain 'Count' header")
	}
}

// TestCouplingMatrix_ASCII_Empty tests ASCII table with no dependencies
func TestCouplingMatrix_ASCII_Empty(t *testing.T) {
	matrix := NewCouplingMatrix()
	ascii := matrix.ToASCII(nil)

	// Should show "no dependencies" message
	if !strings.Contains(ascii, "no dependencies detected") {
		t.Error("Empty matrix should show 'no dependencies detected' message")
	}
}

// TestCouplingMatrix_Percentage tests percentage calculation
func TestCouplingMatrix_Percentage(t *testing.T) {
	matrix := NewCouplingMatrix()
	matrix.Add("application", "domain")
	matrix.Add("application", "domain")
	matrix.Add("application", "domain")
	matrix.Add("application", "infrastructure")

	// Total = 4
	// application->domain = 3 (75%)
	// application->infrastructure = 1 (25%)

	entries := matrix.GetEntriesWithPercentage()

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Find the application->domain entry
	var appDomainEntry CouplingEntry
	for _, e := range entries {
		if e.FromLayer == "application" && e.ToLayer == "domain" {
			appDomainEntry = e
			break
		}
	}

	if appDomainEntry.Count != 3 {
		t.Errorf("Expected application->domain count = 3, got %d", appDomainEntry.Count)
	}

	if appDomainEntry.Percentage != 75.0 {
		t.Errorf("Expected application->domain percentage = 75.0, got %.2f", appDomainEntry.Percentage)
	}
}

// TestCouplingMatrix_CircularPairs tests circular dependency detection
func TestCouplingMatrix_CircularPairs(t *testing.T) {
	matrix := NewCouplingMatrix()

	// Create circular dependency: application <-> domain
	matrix.Add("application", "domain")
	matrix.Add("application", "domain")
	matrix.Add("domain", "application")

	// Non-circular: infrastructure -> domain (one-way)
	matrix.Add("infrastructure", "domain")

	circularPairs := matrix.FindCircularPairs()

	// Should detect exactly one circular pair: (application, domain)
	if len(circularPairs) != 1 {
		t.Fatalf("Expected 1 circular pair, got %d", len(circularPairs))
	}

	pair := circularPairs[0]
	// Pair should be alphabetically ordered
	if !(pair[0] == "application" && pair[1] == "domain") {
		t.Errorf("Expected circular pair [application, domain], got [%s, %s]", pair[0], pair[1])
	}
}

// TestCouplingMatrix_CircularPairs_None tests when no circular dependencies exist
func TestCouplingMatrix_CircularPairs_None(t *testing.T) {
	matrix := NewCouplingMatrix()

	// All one-way dependencies
	matrix.Add("presentation", "application")
	matrix.Add("application", "domain")
	matrix.Add("domain", "infrastructure")

	circularPairs := matrix.FindCircularPairs()

	if len(circularPairs) != 0 {
		t.Errorf("Expected 0 circular pairs, got %d: %v", len(circularPairs), circularPairs)
	}

	if matrix.HasCircularDependencies() {
		t.Error("HasCircularDependencies should return false")
	}

	if matrix.CircularPairCount() != 0 {
		t.Errorf("CircularPairCount should be 0, got %d", matrix.CircularPairCount())
	}
}

// TestCouplingMatrix_CircularPairs_Multiple tests multiple circular dependencies
func TestCouplingMatrix_CircularPairs_Multiple(t *testing.T) {
	matrix := NewCouplingMatrix()

	// Circular pair 1: application <-> domain
	matrix.Add("application", "domain")
	matrix.Add("domain", "application")

	// Circular pair 2: infrastructure <-> presentation
	matrix.Add("infrastructure", "presentation")
	matrix.Add("presentation", "infrastructure")

	// Non-circular
	matrix.Add("application", "infrastructure")

	circularPairs := matrix.FindCircularPairs()

	if len(circularPairs) != 2 {
		t.Fatalf("Expected 2 circular pairs, got %d", len(circularPairs))
	}
}

// TestCouplingMatrix_LayerStats tests layer statistics calculation
func TestCouplingMatrix_LayerStats(t *testing.T) {
	matrix := NewCouplingMatrix()

	// Create dependencies
	matrix.Add("application", "domain")
	matrix.Add("application", "domain")
	matrix.Add("application", "infrastructure")
	matrix.Add("infrastructure", "domain")
	matrix.Add("domain", "application")

	// Get stats for application layer
	stats := matrix.GetLayerStats("application")

	// application has:
	// Outgoing: 2 (to domain) + 1 (to infrastructure) = 3
	// Incoming: 1 (from domain)
	if stats.OutgoingDeps != 3 {
		t.Errorf("Expected application outgoing = 3, got %d", stats.OutgoingDeps)
	}

	if stats.IncomingDeps != 1 {
		t.Errorf("Expected application incoming = 1, got %d", stats.IncomingDeps)
	}

	if stats.NetCoupling != -2 {
		t.Errorf("Expected application net coupling = -2, got %d", stats.NetCoupling)
	}

	if !stats.IsSourceHeavy {
		t.Error("application should be source-heavy (outgoing > incoming)")
	}

	if stats.IsTargetHeavy {
		t.Error("application should not be target-heavy")
	}
}

// TestCouplingMatrix_LayerStats_Isolated tests stats for layer with no dependencies
func TestCouplingMatrix_LayerStats_Isolated(t *testing.T) {
	matrix := NewCouplingMatrix()
	matrix.Add("application", "domain")

	// Get stats for isolated layer
	stats := matrix.GetLayerStats("infrastructure")

	if stats.OutgoingDeps != 0 {
		t.Errorf("Expected isolated layer outgoing = 0, got %d", stats.OutgoingDeps)
	}

	if stats.IncomingDeps != 0 {
		t.Errorf("Expected isolated layer incoming = 0, got %d", stats.IncomingDeps)
	}

	if stats.NetCoupling != 0 {
		t.Errorf("Expected isolated layer net coupling = 0, got %d", stats.NetCoupling)
	}
}

// TestCouplingMatrix_GetAllLayerStats tests getting stats for all layers
func TestCouplingMatrix_GetAllLayerStats(t *testing.T) {
	matrix := NewCouplingMatrix()
	matrix.Add("application", "domain")
	matrix.Add("domain", "infrastructure")

	allStats := matrix.GetAllLayerStats()

	// Should have stats for all 3 layers
	if len(allStats) != 3 {
		t.Errorf("Expected stats for 3 layers, got %d", len(allStats))
	}

	// Check that all expected layers are present
	expectedLayers := []string{"application", "domain", "infrastructure"}
	for _, layer := range expectedLayers {
		if _, ok := allStats[layer]; !ok {
			t.Errorf("Expected stats for layer %q, not found", layer)
		}
	}
}

// TestCouplingMatrix_GetEntriesWithPercentage_ZeroTotal tests percentage when total is zero
func TestCouplingMatrix_GetEntriesWithPercentage_ZeroTotal(t *testing.T) {
	matrix := NewCouplingMatrix()
	// Don't add any entries

	entries := matrix.GetEntriesWithPercentage()

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty matrix, got %d", len(entries))
	}
}

// TestCouplingMatrix_ToASCII_CustomOptions tests ASCII rendering with different options
func TestCouplingMatrix_ToASCII_CustomOptions(t *testing.T) {
	matrix := NewCouplingMatrix()
	matrix.Add("application", "domain")

	// Test with only count
	opts := &ASCIIOptions{
		ShowPercentage: false,
		ShowCount:      true,
	}
	ascii := matrix.ToASCII(opts)

	if strings.Contains(ascii, "Pct") {
		t.Error("ASCII table should not contain 'Pct' header when ShowPercentage is false")
	}
	if !strings.Contains(ascii, "Count") {
		t.Error("ASCII table should contain 'Count' header when ShowCount is true")
	}

	// Test with only percentage
	opts2 := &ASCIIOptions{
		ShowPercentage: true,
		ShowCount:      false,
	}
	ascii2 := matrix.ToASCII(opts2)

	if strings.Contains(ascii2, "Count") {
		t.Error("ASCII table should not contain 'Count' header when ShowCount is false")
	}
	if !strings.Contains(ascii2, "Pct") {
		t.Error("ASCII table should contain 'Pct' header when ShowPercentage is true")
	}
}
