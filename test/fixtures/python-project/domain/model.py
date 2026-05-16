from dataclasses import dataclass
from typing import List


@dataclass
class Order:
    id: str
    items: List[str]
    paid: bool = False


def create_order(item_ids: List[str]) -> Order:
    return Order(id="ORD-001", items=item_ids)
