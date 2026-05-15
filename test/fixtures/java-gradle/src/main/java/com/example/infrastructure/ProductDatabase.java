package com.example.infrastructure;

import com.example.domain.Product;
import java.util.ArrayList;
import java.util.List;

public class ProductDatabase {
    private List<Product> products = new ArrayList<>();

    public void add(Product product) {
        products.add(product);
    }

    public List<Product> findAll() {
        return products;
    }
}
