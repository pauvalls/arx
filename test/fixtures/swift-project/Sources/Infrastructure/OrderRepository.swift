import Foundation
import Domain

/// In-memory order repository (infrastructure layer)
public class OrderRepository {
    private var storage: [String: Order] = [:]

    public func save(_ order: Order) {
        storage[order.id] = order
    }

    public func findById(_ id: String) -> Order? {
        return storage[id]
    }

    public func findAll() -> [Order] {
        return Array(storage.values)
    }
}
