class Order
  attr_accessor :id, :items, :status

  def initialize
    @items = []
    @status = :pending
  end

  def validate
    raise "Order must have items" if @items.empty?
  end

  def save
    # persist order
  end
end
