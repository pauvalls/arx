import Foundation
import Domain
import Infrastructure
@_exported import Domain

/// Order service (application layer) — orchestrates domain and infrastructure
public class OrderService {
    private let repository: OrderRepository

    public init(repository: OrderRepository) {
        self.repository = repository
    }

    public func createOrder(id: String, items: [OrderItem]) -> Order {
        let order = Order(id: id, items: items)
        repository.save(order)
        return order
    }

    public func getOrder(id: String) -> Order? {
        return repository.findById(id)
    }
}
