package com.example.infrastructure.repository;

import com.example.domain.order.Order;
import com.example.domain.order.OrderItem;
import java.util.HashMap;
import java.util.Map;

public class OrderRepository {
    private Map<String, Order> orders = new HashMap<>();

    public void save(Order order) {
        orders.put(order.toString(), order);
    }

    public Order findById(String id) {
        return orders.get(id);
    }
}
