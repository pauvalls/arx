package com.example.domain;

import java.util.List;

public class Product {
    private String id;
    private String name;
    private List<String> categories;

    public Product(String id, String name, List<String> categories) {
        this.id = id;
        this.name = name;
        this.categories = categories;
    }

    public String getId() { return id; }
    public String getName() { return name; }
    public List<String> getCategories() { return categories; }
}
