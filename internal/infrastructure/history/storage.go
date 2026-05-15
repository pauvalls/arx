package history

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

const (
	defaultMaxAudits = 10
	dateFormat       = "2006-01-02"
)

// FileSystemHistory implements HistoryStorage using JSON files
type FileSystemHistory struct {
	historyPath string
	maxAudits   int
}

// NewFileSystemHistory creates a new filesystem-based history storage
func NewFileSystemHistory(historyPath string) *FileSystemHistory {
	return &FileSystemHistory{
		historyPath: historyPath,
		maxAudits:   defaultMaxAudits,
	}
}

// Save stores an audit report to a JSON file
// File naming: audit-YYYY-MM-DD.json
// Does NOT enforce retention automatically - call DeleteOld() separately if needed
func (s *FileSystemHistory) Save(ctx context.Context, report *domain.AuditReport) (string, error) {
	// Ensure history directory exists
	if err := os.MkdirAll(s.historyPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create history directory: %w", err)
	}

	// Generate filename based on timestamp
	filename := fmt.Sprintf("audit-%s.json", report.Timestamp.Format(dateFormat))
	filePath := filepath.Join(s.historyPath, filename)

	// Marshal report to JSON with indentation
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal audit report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write audit file: %w", err)
	}

	return filePath, nil
}

// Load retrieves an audit report for a specific date
func (s *FileSystemHistory) Load(ctx context.Context, date time.Time) (*domain.AuditReport, error) {
	filename := fmt.Sprintf("audit-%s.json", date.Format(dateFormat))
	filePath := filepath.Join(s.historyPath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Return nil, nil if file doesn't exist
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit file: %w", err)
	}

	// Unmarshal JSON
	var report domain.AuditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audit report: %w", err)
	}

	return &report, nil
}

// LoadLatest retrieves the most recent audit report
func (s *FileSystemHistory) LoadLatest(ctx context.Context) (*domain.AuditReport, error) {
	dates, err := s.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list audits: %w", err)
	}

	if len(dates) == 0 {
		return nil, nil // No history exists
	}

	// Get the most recent date (first in sorted list)
	return s.Load(ctx, dates[0])
}

// List returns all stored audit dates, sorted newest first
// Does NOT enforce retention limit - use DeleteOld() for that
func (s *FileSystemHistory) List(ctx context.Context) ([]time.Time, error) {
	// Check if history directory exists
	if _, err := os.Stat(s.historyPath); os.IsNotExist(err) {
		return []time.Time{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(s.historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var dates []time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if filename matches audit-YYYY-MM-DD.json pattern
		name := entry.Name()
		if !strings.HasPrefix(name, "audit-") || !strings.HasSuffix(name, ".json") {
			continue
		}

		// Extract date from filename
		dateStr := strings.TrimSuffix(strings.TrimPrefix(name, "audit-"), ".json")
		date, err := time.Parse(dateFormat, dateStr)
		if err != nil {
			// Skip files with invalid date format
			continue
		}

		dates = append(dates, date)
	}

	// Sort by date (newest first)
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].After(dates[j])
	})

	return dates, nil
}

// DeleteOld removes audits older than the retention limit
// Returns the number of deleted audits
func (s *FileSystemHistory) DeleteOld(ctx context.Context, maxAudits int) (int, error) {
	dates, err := s.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list audits: %w", err)
	}

	deletedCount := 0

	// Delete audits beyond the retention limit
	if len(dates) > maxAudits {
		toDelete := dates[maxAudits:]
		for _, date := range toDelete {
			filename := fmt.Sprintf("audit-%s.json", date.Format(dateFormat))
			filePath := filepath.Join(s.historyPath, filename)

			if err := os.Remove(filePath); err != nil {
				// Log warning but continue with other deletions
				continue
			}
			deletedCount++
		}
	}

	return deletedCount, nil
}
