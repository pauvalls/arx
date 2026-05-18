package ports

import "github.com/pauvalls/arx/internal/domain"

// BaselineStorage defines the interface for baseline persistence.
// Application layer depends on this interface, not on concrete implementations.
type BaselineStorage interface {
	Load(path string) (*domain.Baseline, error)
	Save(b *domain.Baseline, path string) error
	Exists(path string) bool
}
