import type { Order } from '@domain/order';
import { validateOrder } from '@domain/order';

// OrderService orchestrates order operations
export class OrderService {
  createOrder(items: string[]): Order | null {
    const order: Order = { id: crypto.randomUUID(), items };
    if (!validateOrder(order)) {
      return null;
    }
    return order;
  }
}
