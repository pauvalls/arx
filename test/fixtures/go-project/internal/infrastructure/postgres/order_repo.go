package postgres

import "github.com/example/test-project/internal/domain"

// OrderRepository implements order persistence
type OrderRepository struct{}

// NewOrderRepository creates a new repository
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

// FindByID finds an order by ID
func (r *OrderRepository) FindByID(id string) (*domain.Order, error) {
	return &domain.Order{}, nil
}
