package application

import (
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/baseline"
)

// DefaultBaselineFile is the default filename for the baseline.
const DefaultBaselineFile = ".arx-baseline.json"

// BaselineService handles baseline generation, loading, and filtering.
type BaselineService struct {
	storage *baseline.Storage
}

// NewBaselineService creates a new BaselineService.
func NewBaselineService() *BaselineService {
	return &BaselineService{
		storage: baseline.NewStorage(),
	}
}

// Generate creates a baseline from violations and config hash.
func (s *BaselineService) Generate(violations []domain.Violation, configHash string) *domain.Baseline {
	return domain.GenerateBaseline(violations, configHash)
}

// Load reads a baseline from the specified path.
// Returns nil, nil if the file does not exist.
func (s *BaselineService) Load(path string) (*domain.Baseline, error) {
	return s.storage.Load(path)
}

// Save writes a baseline to the specified path.
func (s *BaselineService) Save(b *domain.Baseline, path string) error {
	return s.storage.Save(b, path)
}

// Exists checks if a baseline file exists at the specified path.
func (s *BaselineService) Exists(path string) bool {
	return s.storage.Exists(path)
}

// FilterViolations returns only violations NOT in the baseline.
// Returns all violations unchanged when baseline is nil.
func (s *BaselineService) FilterViolations(violations []domain.Violation, b *domain.Baseline) []domain.Violation {
	if b == nil {
		return violations
	}
	return b.Filter(violations)
}

// DefaultPath returns the default baseline path for a project root.
func (s *BaselineService) DefaultPath(projectRoot string) string {
	return projectRoot + "/" + DefaultBaselineFile
}
