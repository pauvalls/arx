require_relative '../domain/order'

class OrderRepository
  def find(id)
    Order.new
  end

  def save(order)
    # persist to database
  end
end
