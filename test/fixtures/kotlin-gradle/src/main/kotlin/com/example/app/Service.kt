package com.example.app

import com.example.domain.Entity as DomainEntity
import com.example.infrastructure.Database
import java.util.UUID

class Service {
    private val db = Database()

    fun findEntity(id: Long): DomainEntity? {
        return db.query(id)
    }
}
