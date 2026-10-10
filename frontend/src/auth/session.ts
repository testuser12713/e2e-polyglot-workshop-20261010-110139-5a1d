import { useSyncExternalStore } from "react";

// The workshop session is persisted so a signed-in session survives a page
// reload (F5, a bookmarked URL, a link opened directly). The token lives in
// localStorage and is restored on start-up; nothing here ever contacts the
// network.

export interface Session {
  token: string;
  expiresAt: string | null;
  email: string | null;
}

const STORAGE_KEY = "workshop.session.v1";

let current: Session | null = null;
const listeners = new Set<() => void>();

function hasStorage(): boolean {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}

function decodeEmail(token: string): string | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
    const json = JSON.parse(atob(padded)) as Record<string, unknown>;
    return typeof json.email === "string" ? json.email : null;
  } catch {
    return null;
  }
}

function normalize(raw: unknown): Session | null {
  if (!raw || typeof raw !== "object") return null;
  const value = raw as Record<string, unknown>;
  if (typeof value.token !== "string" || value.token.length === 0) return null;
  const expiresAt = typeof value.expiresAt === "string" ? value.expiresAt : null;
  const email =
    typeof value.email === "string" ? value.email : decodeEmail(value.token);
  return { token: value.token, expiresAt, email };
}

function load(): Session | null {
  if (!hasStorage()) return null;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return normalize(JSON.parse(raw));
  } catch {
    return null;
  }
}

function persist(session: Session | null): void {
  if (!hasStorage()) return;
  try {
    if (session) {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Storage may be unavailable (private mode); the session stays in memory.
  }
}

function isExpired(session: Session): boolean {
  if (!session.expiresAt) return false;
  const at = Date.parse(session.expiresAt);
  return Number.isFinite(at) && at <= Date.now();
}

function emit(): void {
  for (const listener of listeners) listener();
}

current = load();

export function getToken(): string | null {
  if (!current || isExpired(current)) return null;
  return current.token;
}

export function setSession(token: string, expiresAt: string): void {
  current = {
    token,
    expiresAt: expiresAt || null,
    email: decodeEmail(token),
  };
  persist(current);
  emit();
}

export function clearSession(): void {
  current = null;
  persist(null);
  emit();
}

function subscribe(callback: () => void): () => void {
  listeners.add(callback);
  const onStorage = (): void => {
    current = load();
    emit();
  };
  if (hasStorage()) window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(callback);
    if (hasStorage()) window.removeEventListener("storage", onStorage);
  };
}

function getSnapshot(): Session | null {
  return current;
}

export function useSession(): Session | null {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
