import XCTest
@testable import Domain
@testable import Application

/// Order tests — should be skipped by detector
final class OrderTests: XCTestCase {
    func testCreateOrder() {
        let repo = OrderRepository()
        let service = OrderService(repository: repo)
        let order = service.createOrder(id: "1", items: [])
        XCTAssertEqual(order.id, "1")
    }
}
