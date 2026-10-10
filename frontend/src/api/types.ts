// Shared types of the sprint contract. Every ticket imports these — do not
// invent a second spelling for a shape named here.

export type OrderStatus =
  | "requested"
  | "confirmed"
  | "in_progress"
  | "completed"
  | "picked_up";

export const ORDER_STATUS_ORDER: readonly OrderStatus[] = [
  "requested",
  "confirmed",
  "in_progress",
  "completed",
  "picked_up",
];

export const ORDER_STATUS_LABELS: Record<OrderStatus, string> = {
  requested: "angefragt",
  confirmed: "bestätigt",
  in_progress: "in Arbeit",
  completed: "fertig",
  picked_up: "abgeholt",
};

export type ApiErrorCode =
  | "validation_error"
  | "not_found"
  | "invalid_transition"
  | "duplicate_plate"
  | "unauthorized"
  | "rate_limited"
  | "internal_error";

export interface Customer {
  id: number;
  name: string;
  email: string;
  phone: string;
}

export interface Vehicle {
  id: number;
  customerId: number;
  plate: string;
  brand: string;
  model: string;
  mileageKm: number;
}

export type OrderItemKind = "labor" | "part";

export interface OrderItem {
  id: number;
  kind: OrderItemKind;
  description: string;
  hours: number | null;
  quantity: number | null;
  unitPriceCents: number | null;
  amountCents: number;
}

export interface StatusEntry {
  status: OrderStatus;
  changedAt: string;
  employeeName: string;
}

export interface StatusHistoryEntry {
  status: OrderStatus;
  changedAt: string;
}

export interface Order {
  id: number;
  orderNumber: string;
  vehicleId: number;
  customerId: number;
  plate: string;
  brand: string;
  model: string;
  status: OrderStatus;
  requestedDate: string;
  problemDescription: string;
  createdAt: string;
  items: OrderItem[];
  history: StatusEntry[];
}

export interface OrderSummary {
  id: number;
  orderNumber: string;
  plate: string;
  brand: string;
  model: string;
  status: OrderStatus;
  requestedDate: string;
  createdAt: string;
}

export interface InvoiceItem {
  kind: OrderItemKind;
  description: string;
  hours?: number;
  quantity?: number;
  unitPriceCents?: number;
  amountCents: number;
}

export interface Invoice {
  invoiceNumber: string;
  netCents: number;
  taxRatePercent: number;
  taxCents: number;
  grossCents: number;
  items: InvoiceItem[];
}

export interface InvoiceTotals {
  netCents: number;
  taxRatePercent: number;
  taxCents: number;
  grossCents: number;
}

export interface PublicOrder {
  orderNumber: string;
  plate: string;
  status: OrderStatus;
  requestedDate: string;
  history: StatusHistoryEntry[];
  invoice: Invoice | null;
}

export interface DashboardMetrics {
  openCounts: {
    requested: number;
    confirmed: number;
    in_progress: number;
  };
  completedToday: number;
  monthRevenueGrossCents: number;
}

export interface LoginResponse {
  token: string;
  expiresAt: string;
}

export interface AppointmentRequest {
  name: string;
  email: string;
  phone: string;
  plate: string;
  brand: string;
  model: string;
  mileageKm: number;
  requestedDate: string;
  problemDescription: string;
}

export interface AppointmentResponse {
  orderNumber: string;
  status: OrderStatus;
}

export interface OrderListResponse {
  orders: OrderSummary[];
}

export interface CreateOrderItemInput {
  kind: OrderItemKind;
  description: string;
  hours?: number;
  quantity?: number;
  unitPriceCents?: number;
}
