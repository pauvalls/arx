package domain

import "github.com/example/test-project/internal/infrastructure/postgres"

// BAD: Domain should not depend on infrastructure
// This is a deliberate violation for testing
func GetOrderFromDB(id string) (*Order, error) {
	repo := postgres.NewOrderRepository()
	return repo.FindByID(id)
}
