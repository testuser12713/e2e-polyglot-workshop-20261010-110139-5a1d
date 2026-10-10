import { clearSession, getToken } from "../auth/session";

// The one fetch wrapper for the whole app. Every page module uses apiFetch so
// that JSON handling, the bearer token and the API's uniform error body live in
// exactly one place.

// VITE_API_BASE_URL is filled in by the runner from ${service:api.origin}. The
// sibling api service is declared by another ticket and may not exist yet, so
// the variable can be unset or empty here: fall back to the local dev port.
const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim();

export const API_BASE_URL: string = (
  configuredBaseUrl ? configuredBaseUrl : "http://localhost:8000"
).replace(/\/+$/, "");

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body !== undefined && init.body !== null && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const url = path.startsWith("http") ? path : `${API_BASE_URL}${path}`;

  let response: Response;
  try {
    response = await fetch(url, { ...init, headers });
  } catch {
    throw new ApiError(0, "network_error", "Der Server ist nicht erreichbar.");
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = null;
    }
  }

  if (!response.ok) {
    // A 401 means the stored token is missing, invalid or expired: drop the
    // session so the workshop guard re-renders and sends the user to login.
    if (response.status === 401) {
      clearSession();
    }
    const error = (body as { error?: { code?: unknown; message?: unknown } } | null)?.error;
    const code = typeof error?.code === "string" ? error.code : "internal_error";
    const message =
      typeof error?.message === "string" && error.message
        ? error.message
        : response.statusText || "Unbekannter Fehler";
    throw new ApiError(response.status, code, message);
  }

  return body as T;
}
