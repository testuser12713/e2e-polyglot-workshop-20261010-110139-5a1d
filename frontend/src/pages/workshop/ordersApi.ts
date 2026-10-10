import { apiFetch } from "../../api/client";
import type { OrderListResponse, OrderStatus, OrderSummary } from "../../api/types";

// Small api module for the order list screen. It maps the page's filter state
// onto the declared query parameters of GET /api/workshop/orders and unwraps
// the {"orders":[...]} envelope into the plain list the table renders.

export interface OrderFilters {
  status?: OrderStatus | "";
  plate?: string;
}

// buildOrdersPath turns the filters into the exact request path. Empty values
// are omitted, so an untouched list requests the bare endpoint.
export function buildOrdersPath(filters: OrderFilters = {}): string {
  const params = new URLSearchParams();
  const status = filters.status?.trim();
  const plate = filters.plate?.trim();
  if (status) params.set("status", status);
  if (plate) params.set("plate", plate);
  const query = params.toString();
  return query ? `/api/workshop/orders?${query}` : "/api/workshop/orders";
}

export async function fetchOrders(filters: OrderFilters = {}): Promise<OrderSummary[]> {
  const data = await apiFetch<OrderListResponse>(buildOrdersPath(filters));
  return data.orders ?? [];
}
