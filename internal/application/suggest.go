package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// Fix represents a single suggested code fix.
type Fix struct {
	ViolationID string // e.g., "D-01"
	RuleID      string
	File        string // file to modify
	Line        int    // line where the violation is
	Description string // human-readable description of the fix
	Diff        string // unified diff of the change
	Original    string // original file content
	Suggested   string // suggested file content
}

// FixTemplateFunc generates a Fix for a violation, or nil if no fix applies.
type FixTemplateFunc func(violation domain.Violation, projectRoot string) *Fix

// FixEngine generates fix suggestions for violations.
type FixEngine struct {
	templates map[string]FixTemplateFunc
}

// NewFixEngine creates a new FixEngine with built-in templates.
func NewFixEngine() *FixEngine {
	return &FixEngine{
		templates: map[string]FixTemplateFunc{
			"domain-imports-infrastructure":    fixDomainImportsInfrastructure,
			"application-imports-infrastructure": fixAppImportsInfrastructure,
		},
	}
}

// SuggestFix generates a fix suggestion for a single violation.
func (e *FixEngine) SuggestFix(v domain.Violation) *Fix {
	if fn, ok := e.templates[v.RuleID]; ok {
		return fn(v, ".")
	}
	// Generic advice for unknown patterns
	return &Fix{
		ViolationID: v.ID,
		RuleID:      v.RuleID,
		File:        v.File,
		Description: fmt.Sprintf("Review the dependency from %s to %s. Consider extracting an interface.", v.SourceLayer, v.TargetLayer),
	}
}

// SuggestAll generates fix suggestions for all violations.
func (e *FixEngine) SuggestAll(violations []domain.Violation) []*Fix {
	var fixes []*Fix
	for _, v := range violations {
		if fix := e.SuggestFix(v); fix != nil {
			fixes = append(fixes, fix)
		}
	}
	return fixes
}

// fixDomainImportsInfrastructure generates a fix for domain layer importing infrastructure.
func fixDomainImportsInfrastructure(v domain.Violation, projectRoot string) *Fix {
	return &Fix{
		ViolationID: v.ID,
		RuleID:      v.RuleID,
		File:        v.File,
		Line:        v.Line,
		Description: fmt.Sprintf("Extract an interface from %s and inject it via constructor", v.Import),
		Diff:        fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ -%d +%d @@\n-// TODO: replace direct import with interface\n+// Define interface in domain layer, inject via constructor", v.File, v.File, v.Line, v.Line),
	}
}

// fixAppImportsInfrastructure generates a fix for application layer importing infrastructure.
func fixAppImportsInfrastructure(v domain.Violation, projectRoot string) *Fix {
	return &Fix{
		ViolationID: v.ID,
		RuleID:      v.RuleID,
		File:        v.File,
		Line:        v.Line,
		Description: "Move the infrastructure dependency behind a port interface",
	}
}

// UnifiedDiff returns the fix as a unified diff string.
// If Diff is already populated, returns it. Otherwise computes from Original/Suggested.
func (f Fix) UnifiedDiff() string {
	if f.Diff != "" {
		return f.Diff
	}
	if f.Original == "" && f.Suggested == "" {
		return ""
	}
	return simpleUnifiedDiff(f.File, f.Original, f.Suggested)
}

// simpleUnifiedDiff produces a minimal unified diff without external dependencies.
func simpleUnifiedDiff(path, original, suggested string) string {
	origLines := strings.Split(original, "\n")
	suggLines := strings.Split(suggested, "\n")

	// Find first differing line
	firstDiff := 0
	for firstDiff < len(origLines) && firstDiff < len(suggLines) {
		if origLines[firstDiff] != suggLines[firstDiff] {
			break
		}
		firstDiff++
	}

	// Find last differing line (from end)
	origEnd := len(origLines) - 1
	suggEnd := len(suggLines) - 1
	for origEnd > firstDiff && suggEnd > firstDiff {
		if origLines[origEnd] != suggLines[suggEnd] {
			break
		}
		origEnd--
		suggEnd--
	}

	// Handle trailing empty element from split
	if origEnd >= len(origLines) {
		origEnd = len(origLines) - 1
	}
	if suggEnd >= len(suggLines) {
		suggEnd = len(suggLines) - 1
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- a/%s\n", path))
	b.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	origCount := origEnd - firstDiff + 1
	suggCount := suggEnd - firstDiff + 1
	if origEnd < firstDiff {
		origCount = 0
	}
	if suggEnd < firstDiff {
		suggCount = 0
	}

	b.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", firstDiff+1, origCount, firstDiff+1, suggCount))

	for i := firstDiff; i <= origEnd; i++ {
		if i < len(origLines) {
			b.WriteString(fmt.Sprintf("-%s\n", origLines[i]))
		}
	}
	for i := firstDiff; i <= suggEnd; i++ {
		if i < len(suggLines) {
			b.WriteString(fmt.Sprintf("+%s\n", suggLines[i]))
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// Apply writes the fix to disk, creating a timestamped backup first.
// backupDir is the root backup directory (e.g., ".arx-backup").
func (e *FixEngine) Apply(fix Fix, backupDir string) error {
	if fix.File == "" {
		return fmt.Errorf("cannot apply fix: no file specified")
	}

	// Read current file content for backup
	current, err := os.ReadFile(fix.File)
	if err != nil {
		return fmt.Errorf("cannot read file %q: %w", fix.File, err)
	}

	// Create timestamped backup directory
	timestamp := time.Now().Format("20060102T150405")
	backupPath := filepath.Join(backupDir, timestamp)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("cannot create backup directory: %w", err)
	}

	// Write backup file (preserve relative path with .bak suffix)
	backupFile := filepath.Join(backupPath, fix.File+".bak")
	backupDirPath := filepath.Dir(backupFile)
	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		return fmt.Errorf("cannot create backup subdirectory: %w", err)
	}
	if err := os.WriteFile(backupFile, current, 0644); err != nil {
		return fmt.Errorf("cannot create backup: %w", err)
	}

	// Write suggested content
	if err := os.WriteFile(fix.File, []byte(fix.Suggested), 0644); err != nil {
		// Attempt rollback on failure
		_ = e.Rollback(fix.File, backupDir)
		return fmt.Errorf("failed to write fix to %q: %w (restored from backup)", fix.File, err)
	}

	return nil
}

// Rollback restores a file from the latest backup in the backup directory.
func (e *FixEngine) Rollback(file string, backupDir string) error {
	// Find latest backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("cannot read backup directory: %w", err)
	}

	var latest string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() > latest {
			latest = entry.Name()
		}
	}
	if latest == "" {
		return fmt.Errorf("no backups found in %s", backupDir)
	}

	backupFile := filepath.Join(backupDir, latest, file+".bak")
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("cannot read backup file %q: %w", backupFile, err)
	}

	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Errorf("cannot restore file %q: %w", file, err)
	}

	return nil
}
