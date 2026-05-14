package domain

// Order represents a business entity
type Order struct {
	ID    string
 Items []string
}

// ValidateOrder checks if an order is valid
func ValidateOrder(order Order) bool {
	return len(order.Items) > 0
}
