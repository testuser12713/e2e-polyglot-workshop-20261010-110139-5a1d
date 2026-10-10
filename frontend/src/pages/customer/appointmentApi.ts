import { apiFetch } from "../../api/client";
import type { AppointmentRequest, AppointmentResponse } from "../../api/types";

// The page's only network entry point. The sprint contract fixes the request
// and response shape of POST /api/public/appointments, so this small module
// exists next to the page — the shared apiFetch client does JSON, error bodies
// and the (absent) bearer token.
export function submitAppointment(
  payload: AppointmentRequest,
): Promise<AppointmentResponse> {
  return apiFetch<AppointmentResponse>("/api/public/appointments", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
