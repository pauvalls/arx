require_relative '../lib/domain/order'

RSpec.describe Order do
  it 'has items' do
    order = Order.new
    expect(order.items).to eq([])
  end
end
