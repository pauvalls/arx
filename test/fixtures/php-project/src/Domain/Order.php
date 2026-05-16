<?php

namespace App\Domain;

class Order
{
    private string $id;
    private array $items;
    private string $status;

    public function __construct()
    {
        $this->items = [];
        $this->status = 'pending';
    }

    public function validate(): void
    {
        if (empty($this->items)) {
            throw new \InvalidArgumentException('Order must have items');
        }
    }

    public function save(): void
    {
        // persist order
    }
}
