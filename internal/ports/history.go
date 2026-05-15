package ports

import (
	"context"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// HistoryStorage defines the interface for persisting audit history
type HistoryStorage interface {
	// Save stores an audit report with automatic retention policy enforcement
	// Returns the path where the report was saved
	Save(ctx context.Context, report *domain.AuditReport) (string, error)

	// Load retrieves a specific audit report by date
	// Returns nil if no audit exists for the given date
	Load(ctx context.Context, date time.Time) (*domain.AuditReport, error)

	// LoadLatest retrieves the most recent audit report
	// Returns nil if no history exists
	LoadLatest(ctx context.Context) (*domain.AuditReport, error)

	// List returns all stored audit dates, sorted newest first
	// Limited to maxAudits entries per retention policy
	List(ctx context.Context) ([]time.Time, error)

	// DeleteOld removes audits older than the retention limit
	// Returns the number of deleted audits
	DeleteOld(ctx context.Context, maxAudits int) (int, error)
}
