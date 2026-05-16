require_relative '../domain/order'
require_relative '../infrastructure/order_repo'

class OrderService
  def initialize
    @order = Order.new
    @repo = OrderRepository.new
  end

  def process
    @order.validate
    @repo.save(@order)
  end
end
