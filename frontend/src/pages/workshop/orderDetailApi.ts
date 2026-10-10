import { apiFetch } from "../../api/client";
import type { CreateOrderItemInput, Order, OrderStatus } from "../../api/types";

// The small API module of the order-detail screen. It calls the three order
// endpoints of the contract through the shared api client; every path is the
// one the backend registers, with no trailing slash.

export function fetchOrder(id: number | string): Promise<Order> {
  return apiFetch<Order>(`/api/workshop/orders/${id}`);
}

export function addOrderItems(
  id: number | string,
  items: CreateOrderItemInput[],
): Promise<Order> {
  return apiFetch<Order>(`/api/workshop/orders/${id}/items`, {
    method: "POST",
    body: JSON.stringify({ items }),
  });
}

export function changeOrderStatus(id: number | string, status: OrderStatus): Promise<Order> {
  return apiFetch<Order>(`/api/workshop/orders/${id}/status`, {
    method: "POST",
    body: JSON.stringify({ status }),
  });
}
