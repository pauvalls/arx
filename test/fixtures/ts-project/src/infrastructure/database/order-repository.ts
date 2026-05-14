import type { Order } from '../domain/order';

// OrderRepository implements order persistence
export class OrderRepository {
  async findById(id: string): Promise<Order> {
    return { id, items: [] };
  }
}
