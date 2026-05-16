import Foundation

/// Order domain entity
public struct Order {
    public let id: String
    public let items: [OrderItem]
    public var status: OrderStatus

    public init(id: String, items: [OrderItem], status: OrderStatus = .pending) {
        self.id = id
        self.items = items
        self.status = status
    }
}

/// Order item within an order
public struct OrderItem {
    public let productId: String
    public let quantity: Int
}

/// Order lifecycle status
public enum OrderStatus: String {
    case pending
    case confirmed
    case shipped
    case delivered
    case cancelled
}
