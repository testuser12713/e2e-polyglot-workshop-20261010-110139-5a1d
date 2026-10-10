import type { ReactNode } from "react";

// Pass-through stub. The workshop login ticket replaces the body with a real
// guard that redirects unauthenticated visitors to /werkstatt/anmeldung.
export function RequireWorkshopAuth({ children }: { children: ReactNode }) {
  return <>{children}</>;
}
