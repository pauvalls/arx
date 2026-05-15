package domain

import (
	"fmt"
	"strings"
)

// CouplingCalculator provides methods to calculate coupling metrics
type CouplingCalculator struct{}

// NewCouplingCalculator creates a new coupling calculator
func NewCouplingCalculator() *CouplingCalculator {
	return &CouplingCalculator{}
}

// CalculateCouplingMatrix calculates the dependency counts between all layer pairs
// It processes all dependencies and builds a matrix showing how many dependencies
// exist from each source layer to each target layer
func (c *CouplingCalculator) CalculateCouplingMatrix(dependencies []Dependency, layers []Layer) CouplingMatrix {
	matrix := NewCouplingMatrix()

	// Build layer lookup map for quick resolution
	layerMap := make(map[string]*Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Process each dependency
	for _, dep := range dependencies {
		sourceLayer := resolveLayerForCoupling(dep.SourceFile, layerMap)
		targetLayer := dep.ResolvedLayer

		// Skip if we can't resolve both layers
		if sourceLayer == "" || targetLayer == "" {
			continue
		}

		// Add to matrix
		matrix.Add(sourceLayer, targetLayer)
	}

	return matrix
}

// resolveLayerForCoupling resolves a file path to its layer
// Similar to the function in circular.go but simplified for coupling calculation
func resolveLayerForCoupling(filePath string, layerMap map[string]*Layer) string {
	for _, layer := range layerMap {
		if layer.MatchesPath(filePath) {
			return layer.Name
		}
	}
	return ""
}

// CouplingEntry represents a single entry in the coupling matrix with percentage
type CouplingEntry struct {
	FromLayer  string
	ToLayer    string
	Count      int
	Percentage float64
}

// GetEntriesWithPercentage returns all coupling entries with their percentages
// Percentage is calculated as (count / total_dependencies) * 100
func (m *CouplingMatrix) GetEntriesWithPercentage() []CouplingEntry {
	total := m.Count()
	entries := make([]CouplingEntry, 0)

	if m.FromTo == nil {
		return entries
	}

	for fromLayer, targets := range m.FromTo {
		for toLayer, count := range targets {
			percentage := 0.0
			if total > 0 {
				percentage = (float64(count) / float64(total)) * 100
			}

			entries = append(entries, CouplingEntry{
				FromLayer:  fromLayer,
				ToLayer:    toLayer,
				Count:      count,
				Percentage: percentage,
			})
		}
	}

	return entries
}

// FindCircularPairs detects bidirectional dependencies (A->B and B->A)
// Returns pairs of layers that have circular dependencies
func (m *CouplingMatrix) FindCircularPairs() [][2]string {
	var circularPairs [][2]string

	if m.FromTo == nil {
		return circularPairs
	}

	// Check for each pair if both directions exist
	checked := make(map[string]bool)

	for fromLayer, targets := range m.FromTo {
		for toLayer := range targets {
			// Create a unique key for this pair (alphabetically ordered)
			pairKey := fromLayer + "->" + toLayer
			if checked[toLayer+"->"+fromLayer] {
				// Already checked this pair in reverse
				continue
			}

			// Check if reverse dependency exists
			if m.Get(toLayer, fromLayer) > 0 {
				// Found circular dependency
				// Order alphabetically for consistency
				if fromLayer < toLayer {
					circularPairs = append(circularPairs, [2]string{fromLayer, toLayer})
				} else {
					circularPairs = append(circularPairs, [2]string{toLayer, fromLayer})
				}
			}

			checked[pairKey] = true
		}
	}

	return circularPairs
}

// HasCircularDependencies returns true if any circular dependencies exist
func (m *CouplingMatrix) HasCircularDependencies() bool {
	return len(m.FindCircularPairs()) > 0
}

// CircularPairCount returns the number of circular dependency pairs
func (m *CouplingMatrix) CircularPairCount() int {
	return len(m.FindCircularPairs())
}

// ASCIIOptions controls the rendering of the ASCII table
type ASCIIOptions struct {
	ShowPercentage bool
	ShowCount      bool
	Colorize       bool // Ignored in ASCII mode, but kept for API compatibility
}

// DefaultASCIIOptions returns default options for ASCII rendering
func DefaultASCIIOptions() *ASCIIOptions {
	return &ASCIIOptions{
		ShowPercentage: true,
		ShowCount:      true,
		Colorize:       false,
	}
}

// ToASCII renders the coupling matrix as an ASCII table
// Format:
// +---------------+---------------+---------------+
// | From          | To            | Count  | Pct  |
// +---------------+---------------+---------------+
// | application   | domain        | 5      | 10%  |
// | domain        | infrastructure| 3      | 6%   |
// +---------------+---------------+---------------+
func (m *CouplingMatrix) ToASCII(opts *ASCIIOptions) string {
	if opts == nil {
		opts = DefaultASCIIOptions()
	}

	entries := m.GetEntriesWithPercentage()

	// Handle empty matrix
	if len(entries) == 0 {
		return "+---------------+---------------+-------+------+\n" +
			"| From          | To            | Count | Pct  |\n" +
			"+---------------+---------------+-------+------+\n" +
			"| (no dependencies detected)                              |\n" +
			"+---------------+---------------+-------+------+"
	}

	// Calculate column widths
	maxFrom := 15
	maxTo := 15
	for _, e := range entries {
		if len(e.FromLayer) > maxFrom {
			maxFrom = len(e.FromLayer)
		}
		if len(e.ToLayer) > maxTo {
			maxTo = len(e.ToLayer)
		}
	}

	// Build the table
	var sb strings.Builder

	// Top border
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxFrom+2))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxTo+2))
	sb.WriteString("+")
	if opts.ShowCount {
		sb.WriteString("-------+")
	}
	if opts.ShowPercentage {
		sb.WriteString("------+")
	}
	sb.WriteString("\n")

	// Header
	sb.WriteString("| ")
	sb.WriteString(padRight("From", maxFrom))
	sb.WriteString(" | ")
	sb.WriteString(padRight("To", maxTo))
	sb.WriteString(" |")
	if opts.ShowCount {
		sb.WriteString(" Count |")
	}
	if opts.ShowPercentage {
		sb.WriteString(" Pct  |")
	}
	sb.WriteString("\n")

	// Header separator
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxFrom+2))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxTo+2))
	sb.WriteString("+")
	if opts.ShowCount {
		sb.WriteString("-------+")
	}
	if opts.ShowPercentage {
		sb.WriteString("------+")
	}
	sb.WriteString("\n")

	// Data rows
	for _, e := range entries {
		sb.WriteString("| ")
		sb.WriteString(padRight(e.FromLayer, maxFrom))
		sb.WriteString(" | ")
		sb.WriteString(padRight(e.ToLayer, maxTo))
		sb.WriteString(" |")
		if opts.ShowCount {
			sb.WriteString(fmt.Sprintf(" %5d |", e.Count))
		}
		if opts.ShowPercentage {
			sb.WriteString(fmt.Sprintf(" %5.1f%% |", e.Percentage))
		}
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxFrom+2))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", maxTo+2))
	sb.WriteString("+")
	if opts.ShowCount {
		sb.WriteString("-------+")
	}
	if opts.ShowPercentage {
		sb.WriteString("------+")
	}

	return sb.String()
}

// padRight pads a string to the specified length with spaces on the right
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// GetLayerStats returns statistics for a specific layer
type LayerStats struct {
	LayerName       string
	OutgoingDeps    int // Dependencies this layer has to other layers
	IncomingDeps    int // Dependencies other layers have to this layer
	NetCoupling     int // Incoming - Outgoing (positive = depended upon, negative = depends on others)
	IsSourceHeavy   bool // true if Outgoing > Incoming (layer depends on others more than they depend on it)
	IsTargetHeavy   bool // true if Incoming > Outgoing (layer is depended upon more than it depends on others)
}

// GetLayerStats calculates statistics for a specific layer
func (m *CouplingMatrix) GetLayerStats(layerName string) LayerStats {
	stats := LayerStats{LayerName: layerName}

	if m.FromTo == nil {
		return stats
	}

	// Count outgoing dependencies
	for target, count := range m.FromTo[layerName] {
		if target != layerName { // Skip self-dependencies
			stats.OutgoingDeps += count
		}
	}

	// Count incoming dependencies
	for source, targets := range m.FromTo {
		if source != layerName { // Skip self-dependencies
			if count, ok := targets[layerName]; ok {
				stats.IncomingDeps += count
			}
		}
	}

	stats.NetCoupling = stats.IncomingDeps - stats.OutgoingDeps
	stats.IsSourceHeavy = stats.OutgoingDeps > stats.IncomingDeps
	stats.IsTargetHeavy = stats.IncomingDeps > stats.OutgoingDeps

	return stats
}

// GetAllLayerStats returns statistics for all layers in the matrix
func (m *CouplingMatrix) GetAllLayerStats() map[string]LayerStats {
	stats := make(map[string]LayerStats)
	layers := m.getAllLayers()

	for _, layer := range layers {
		stats[layer] = m.GetLayerStats(layer)
	}

	return stats
}

// getAllLayers returns all unique layer names in the matrix
func (m *CouplingMatrix) getAllLayers() []string {
	layerSet := make(map[string]bool)

	if m.FromTo == nil {
		return []string{}
	}

	for from := range m.FromTo {
		layerSet[from] = true
		for to := range m.FromTo[from] {
			layerSet[to] = true
		}
	}

	layers := make([]string, 0, len(layerSet))
	for layer := range layerSet {
		layers = append(layers, layer)
	}

	return layers
}
