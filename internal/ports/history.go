package ports

import "github.com/pauvalls/arx/internal/domain"

// HistoryStorage defines the interface for audit history persistence
type HistoryStorage interface {
	// Save persists an audit report
	Save(report *domain.AuditReport) error

	// Load retrieves an audit report by filename
	Load(filename string) (*domain.AuditReport, error)

	// LoadLatest retrieves the most recent audit report
	LoadLatest() (*domain.AuditReport, error)

	// List returns all audit filenames sorted by date (newest first)
	List() ([]string, error)

	// DeleteOld removes audits older than the retention limit
	DeleteOld(retention int) error

	// GetRetentionLimit returns the default retention limit
	GetRetentionLimit() int
}
