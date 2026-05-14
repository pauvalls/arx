package domain

import (
	"fmt"
	"sort"
)

// CircularDependency represents a detected circular dependency cycle
type CircularDependency struct {
	Cycle []string // Ordered list of layers in the cycle (e.g., ["domain", "application", "infrastructure", "domain"])
	Path  []string // Full path showing the imports that create the cycle
}

// DetectCircularDependencies finds all circular dependencies in the dependency graph
// Uses DFS-based cycle detection algorithm
func DetectCircularDependencies(dependencies []Dependency, layers []Layer) []CircularDependency {
	// Build adjacency list from dependencies
	layerMap := make(map[string]*Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Build graph: source layer -> target layers
	graph := make(map[string][]string)
	edgeDetails := make(map[string]map[string]Dependency) // source -> target -> dependency details

	for _, dep := range dependencies {
		sourceLayer := resolveLayer(dep.SourceFile, layerMap)
		targetLayer := dep.ResolvedLayer

		if sourceLayer == "" || targetLayer == "" {
			continue
		}

		// Add edge if not already present
		if !contains(graph[sourceLayer], targetLayer) {
			graph[sourceLayer] = append(graph[sourceLayer], targetLayer)
		}

		// Store dependency details
		if edgeDetails[sourceLayer] == nil {
			edgeDetails[sourceLayer] = make(map[string]Dependency)
		}
		if _, exists := edgeDetails[sourceLayer][targetLayer]; !exists {
			edgeDetails[sourceLayer][targetLayer] = dep
		}
	}

	// Find all cycles using DFS
	var cycles []CircularDependency
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle - extract it
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}

				if cycleStart != -1 {
					cycle := path[cycleStart:]
					cycle = append(cycle, neighbor) // Close the cycle

					// Build full path with import details
					var fullPath []string
					for i := 0; i < len(cycle)-1; i++ {
						from := cycle[i]
						to := cycle[i+1]
						if dep, ok := edgeDetails[from][to]; ok {
							fullPath = append(fullPath, fmt.Sprintf("%s (%s:%d)", from, dep.SourceFile, dep.SourceLine))
						} else {
							fullPath = append(fullPath, from)
						}
					}
					fullPath = append(fullPath, neighbor)

					// Check if this cycle already exists (avoid duplicates)
					if !cycleExists(cycles, cycle) {
						cycles = append(cycles, CircularDependency{
							Cycle: cycle,
							Path:  fullPath,
						})
					}
				}
			}
		}

		// Backtrack
		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	// Run DFS from each unvisited node
	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	// Sort cycles for deterministic output
	sortCycles(cycles)

	return cycles
}

// Contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// cycleExists checks if a cycle already exists in the list (ignoring rotation)
func cycleExists(cycles []CircularDependency, newCycle []string) bool {
	for _, existing := range cycles {
		if sameCycle(existing.Cycle, newCycle) {
			return true
		}
	}
	return false
}

// sameCycle checks if two cycles are the same (ignoring rotation)
func sameCycle(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Normalize both cycles (start from smallest element)
	aNorm := normalizeCycle(a)
	bNorm := normalizeCycle(b)

	for i := range aNorm {
		if aNorm[i] != bNorm[i] {
			return false
		}
	}

	return true
}

// normalizeCycle rotates a cycle to start from its smallest element
func normalizeCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	// The last element repeats the first, so we work with len-1 elements
	n := len(cycle) - 1

	// Find index of smallest element (excluding the repeated last element)
	minIdx := 0
	for i := 1; i < n; i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}

	// Rotate to start from minIdx (only the unique elements)
	normalized := make([]string, 0, n+1)
	for i := 0; i < n; i++ {
		idx := (minIdx + i) % n
		normalized = append(normalized, cycle[idx])
	}
	// Add the first element again to close the cycle
	normalized = append(normalized, normalized[0])

	return normalized
}

// sortCycles sorts cycles for deterministic output
func sortCycles(cycles []CircularDependency) {
	sort.Slice(cycles, func(i, j int) bool {
		if len(cycles[i].Cycle) != len(cycles[j].Cycle) {
			return len(cycles[i].Cycle) < len(cycles[j].Cycle)
		}
		return cycles[i].Cycle[0] < cycles[j].Cycle[0]
	})
}

// CreateCircularViolations generates Violation objects from detected circular dependencies
func CreateCircularViolations(cycles []CircularDependency, rule Rule) []Violation {
	var violations []Violation

	for i, cycle := range cycles {
		// Create a violation for the cycle
		violation := Violation{
			ID:          fmt.Sprintf("C-%02d", i+1),
			RuleID:      rule.ID,
			Severity:    SeverityError,
			File:        cycle.Path[0], // First file in the cycle
			Line:        1,             // Could be improved to show actual line
			SourceLayer: cycle.Cycle[0],
			TargetLayer: cycle.Cycle[len(cycle.Cycle)-2], // Layer before the cycle closes
			Import:      fmt.Sprintf("circular: %s", formatCycle(cycle.Cycle)),
			Message:     buildCircularDependencyMessage(cycle),
		}
		violations = append(violations, violation)
	}

	return violations
}

// formatCycle formats a cycle for display
func formatCycle(cycle []string) string {
	return joinStrings(cycle, " → ")
}

// buildCircularDependencyMessage creates a human-readable message for circular dependencies
func buildCircularDependencyMessage(cycle CircularDependency) string {
	return fmt.Sprintf(
		"Circular dependency detected: %s. "+
			"This creates tightly coupled code that is hard to test, modify, and deploy. "+
			"Break the cycle by extracting shared abstractions or using dependency injection.",
		formatCycle(cycle.Cycle),
	)
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
