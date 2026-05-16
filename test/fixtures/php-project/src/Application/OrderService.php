<?php

namespace App\Application;

use App\Domain\Order;
use App\Infrastructure\OrderRepository;
use Symfony\Component\HttpFoundation\Request;

require_once __DIR__ . '/../Domain/Order.php';

class OrderService
{
    private OrderRepository $repository;

    public function __construct()
    {
        $this->repository = new OrderRepository();
    }

    public function process(Request $request): void
    {
        $order = new Order();
        $order->validate();
        $this->repository->save($order);
    }
}
