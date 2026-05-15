package com.example.app;

import org.junit.Assert;
import com.example.domain.order.Order;
import com.example.app.service.OrderService;

public class OrderServiceTest {

    @org.junit.Test
    public void testCreateOrder() {
        Order order = new Order();
        Assert.assertNotNull(order);
    }
}
