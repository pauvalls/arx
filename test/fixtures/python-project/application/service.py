# Application layer — imports from domain and infrastructure
from domain.model import Order, create_order
from infrastructure.repository import save_order


class OrderService:
    def place_order(self, item_ids: list) -> Order:
        order = create_order(item_ids)
        save_order(order)
        return order
