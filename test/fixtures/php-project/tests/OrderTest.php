<?php

namespace App\Tests;

use App\Domain\Order;
use PHPUnit\Framework\TestCase;

class OrderTest extends TestCase
{
    public function testOrderCreation(): void
    {
        $order = new Order();
        $this->assertEquals('pending', $order->status);
    }
}
