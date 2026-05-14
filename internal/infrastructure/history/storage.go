package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pauvalls/arx/internal/domain"
)

// FileSystemStorage implements HistoryStorage using local JSON files
type FileSystemStorage struct {
	baseDir string
}

// NewFileSystemStorage creates a new FileSystemStorage
func NewFileSystemStorage(baseDir string) *FileSystemStorage {
	return &FileSystemStorage{
		baseDir: baseDir,
	}
}

// Save persists an audit report to a JSON file
func (s *FileSystemStorage) Save(report *domain.AuditReport) error {
	// Ensure directory exists
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("audit-%s.json", report.Timestamp.Format("2006-01-02"))
	filepath := filepath.Join(s.baseDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load retrieves an audit report by filename
func (s *FileSystemStorage) Load(filename string) (*domain.AuditReport, error) {
	filepath := filepath.Join(s.baseDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("audit file not found: %s", filename)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var report domain.AuditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

// LoadLatest retrieves the most recent audit report
func (s *FileSystemStorage) LoadLatest() (*domain.AuditReport, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil // No history available
	}

	// Files are sorted by date (newest first), so take the first one
	return s.Load(files[0])
}

// List returns all audit filenames sorted by date (newest first)
func (s *FileSystemStorage) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			files = append(files, entry.Name())
		}
	}

	// Sort by filename (date-based, so newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	return files, nil
}

// DeleteOld removes audits older than the retention limit
func (s *FileSystemStorage) DeleteOld(retention int) error {
	files, err := s.List()
	if err != nil {
		return err
	}

	// Delete files beyond retention limit
	if len(files) > retention {
		for i := retention; i < len(files); i++ {
			filepath := filepath.Join(s.baseDir, files[i])
			if err := os.Remove(filepath); err != nil {
				return fmt.Errorf("failed to delete old audit %s: %w", files[i], err)
			}
		}
	}

	return nil
}

// GetRetentionLimit returns the default retention limit
func (s *FileSystemStorage) GetRetentionLimit() int {
	return 10 // Keep last 10 audits
}

// Ensure FileSystemStorage implements the interface
var _ interface {
	Save(*domain.AuditReport) error
	Load(string) (*domain.AuditReport, error)
	LoadLatest() (*domain.AuditReport, error)
	List() ([]string, error)
	DeleteOld(int) error
	GetRetentionLimit() int
} = (*FileSystemStorage)(nil)
