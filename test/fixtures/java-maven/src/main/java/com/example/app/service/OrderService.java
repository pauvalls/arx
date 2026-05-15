package com.example.app.service;

import com.example.domain.order.Order;
import com.example.domain.order.OrderItem;
import com.example.infrastructure.repository.OrderRepository;

public class OrderService {
    private OrderRepository repository;

    public OrderService(OrderRepository repository) {
        this.repository = repository;
    }

    public void createOrder(Order order) {
        repository.save(order);
    }

    public Order getOrder(String id) {
        return repository.findById(id);
    }
}
