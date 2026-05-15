package com.example.app;

import com.example.domain.Product;
import com.example.infrastructure.ProductDatabase;
import java.util.List;

public class ProductService {
    private ProductDatabase database;

    public ProductService(ProductDatabase database) {
        this.database = database;
    }

    public List<Product> listAll() {
        return database.findAll();
    }
}
