<?php

namespace App\Infrastructure;

use App\Domain\Order;

class OrderRepository
{
    public function find(string $id): Order
    {
        return new Order();
    }

    public function save(Order $order): void
    {
        // persist to database
    }
}
