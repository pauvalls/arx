package application

import "github.com/example/test-project/internal/domain"

// OrderService orchestrates order operations
type OrderService struct{}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(items []string) *domain.Order {
	order := domain.Order{Items: items}
	if !domain.ValidateOrder(order) {
		return nil
	}
	return &order
}
