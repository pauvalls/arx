package com.example.domain.order;

import java.util.List;
import java.util.ArrayList;

public class Order {
    private List<OrderItem> items;

    public Order() {
        this.items = new ArrayList<>();
    }

    public void addItem(OrderItem item) {
        this.items.add(item);
    }

    public List<OrderItem> getItems() {
        return items;
    }
}
