package com.example

import org.junit.Test
import com.example.domain.Entity

class EntityTest {
    @Test
    fun testCreate() {
        val entity = Entity(1L, "test")
        assert(entity.name == "test")
    }
}
