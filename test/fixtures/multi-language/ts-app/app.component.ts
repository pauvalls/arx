import { Component } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { OrderService } from '../domain/order.service';

@Injectable({ providedIn: 'root' })
export class AppComponent {
  constructor(private service: OrderService) {}
}
