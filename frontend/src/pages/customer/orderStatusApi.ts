import { apiFetch } from "../../api/client";
import type { PublicOrder } from "../../api/types";

// The one call this page owns: the public order-status lookup. It goes through
// the shared api client so JSON handling and the API's uniform error body live
// in one place, and it matches the contract path exactly —
// GET /api/public/orders/{orderNumber}?plate=<plate>.
export function fetchOrderStatus(orderNumber: string, plate: string): Promise<PublicOrder> {
  const query = new URLSearchParams({ plate });
  return apiFetch<PublicOrder>(
    `/api/public/orders/${encodeURIComponent(orderNumber)}?${query.toString()}`,
  );
}
