// Order entity - core business logic
export interface Order {
  id: string;
  items: string[];
}

export function validateOrder(order: Order): boolean {
  return order.items.length > 0;
}
