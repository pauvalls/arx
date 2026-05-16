# Infrastructure layer — imports from domain
from domain.model import Order, create_order


class PostgresOrderRepository:
    def save(self, order: Order) -> None:
        # Would persist to database
        pass

    def find_by_id(self, order_id: str) -> Order:
        return create_order([])


def save_order(order: Order) -> None:
    repo = PostgresOrderRepository()
    repo.save(order)
