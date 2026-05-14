// BAD: Domain should not depend on infrastructure
// This is a deliberate violation for testing
import { OrderRepository } from '@infrastructure/database/order-repository';
import type { Order } from './order';

export async function getOrderFromDB(id: string): Promise<Order> {
  const repo = new OrderRepository();
  return repo.findById(id);
}
