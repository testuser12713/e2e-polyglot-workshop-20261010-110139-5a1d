import { ApiError, apiFetch } from "../../api/client";
import type { DashboardMetrics } from "../../api/types";

// The dashboard's own small api module. The contract fixes the response shape
// of GET /api/workshop/dashboard, so the raw body is validated here: a body
// that does not carry every figure is an error, never a silent zero.

function isFigure(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0;
}

function invalidResponse(): ApiError {
  return new ApiError(
    0,
    "invalid_response",
    "Die Dashboard-Daten konnten nicht gelesen werden.",
  );
}

export function parseDashboardMetrics(input: unknown): DashboardMetrics {
  if (typeof input !== "object" || input === null) {
    throw invalidResponse();
  }
  const record = input as Record<string, unknown>;
  const counts = record.openCounts;
  if (typeof counts !== "object" || counts === null) {
    throw invalidResponse();
  }
  const openCounts = counts as Record<string, unknown>;
  if (
    !isFigure(openCounts.requested) ||
    !isFigure(openCounts.confirmed) ||
    !isFigure(openCounts.in_progress) ||
    !isFigure(record.completedToday) ||
    !isFigure(record.monthRevenueGrossCents)
  ) {
    throw invalidResponse();
  }
  return {
    openCounts: {
      requested: openCounts.requested,
      confirmed: openCounts.confirmed,
      in_progress: openCounts.in_progress,
    },
    completedToday: record.completedToday,
    monthRevenueGrossCents: record.monthRevenueGrossCents,
  };
}

export async function fetchDashboardMetrics(): Promise<DashboardMetrics> {
  const data = await apiFetch<unknown>("/api/workshop/dashboard");
  return parseDashboardMetrics(data);
}
