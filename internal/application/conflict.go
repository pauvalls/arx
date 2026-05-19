package application

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pauvalls/arx/internal/domain"
)

// hunkHeaderRe matches unified diff hunk headers like @@ -5,3 +5,3 @@
// Captures: old_start, old_count, new_start, new_count
var hunkHeaderRe = regexp.MustCompile(`@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@`)

// DetectConflicts detects overlapping fix suggestions in the same file.
func DetectConflicts(suggestions []domain.FixSuggestion) []domain.Conflict {
	if len(suggestions) < 2 {
		return nil
	}

	// Group suggestions by file
	byFile := make(map[string][]domain.FixSuggestion)
	for _, s := range suggestions {
		if s.Diff == "" {
			continue
		}
		byFile[s.File] = append(byFile[s.File], s)
	}

	var conflicts []domain.Conflict
	for file, fileSugs := range byFile {
		if len(fileSugs) < 2 {
			continue
		}

		// Parse hunk ranges for each suggestion
		type sugWithRange struct {
			sug   domain.FixSuggestion
			start int
			end   int
		}

		var parsed []sugWithRange
		for _, s := range fileSugs {
			start, end := parseHunkRange(s.Diff)
			if start == 0 && end == 0 {
				continue
			}
			parsed = append(parsed, sugWithRange{sug: s, start: start, end: end})
		}

		// Compare each pair for overlap (3-line tolerance)
		for i := 0; i < len(parsed); i++ {
			for j := i + 1; j < len(parsed); j++ {
				if rangesOverlap(parsed[i].start, parsed[i].end, parsed[j].start, parsed[j].end, 3) {
					conflicts = append(conflicts, domain.Conflict{
						File: file,
						Suggestions: [2]domain.FixSuggestion{
							parsed[i].sug,
							parsed[j].sug,
						},
						Description: fmt.Sprintf("Overlapping fixes for %s: %s and %s affect adjacent or overlapping lines",
							file, parsed[i].sug.ViolationID, parsed[j].sug.ViolationID),
					})
				}
			}
		}
	}

	return conflicts
}

// parseHunkRange extracts the line range from a unified diff hunk header.
// Returns (startLine, endLine) for the new file (+) side, or (0, 0) if parsing fails.
func parseHunkRange(diff string) (int, int) {
	matches := hunkHeaderRe.FindStringSubmatch(diff)
	// matches: [full, old_start, old_count, new_start, new_count]
	if len(matches) < 5 {
		return 0, 0
	}
	start, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, 0
	}
	countStr := matches[4]
	count := 1
	if countStr != "" {
		count, err = strconv.Atoi(countStr)
		if err != nil {
			count = 1
		}
	}
	return start, start + count - 1
}

// rangesOverlap checks if two line ranges overlap within a tolerance.
func rangesOverlap(start1, end1, start2, end2, tolerance int) bool {
	// Extend ranges by tolerance
	return start1-tolerance <= end2+tolerance && start2-tolerance <= end1+tolerance
}
