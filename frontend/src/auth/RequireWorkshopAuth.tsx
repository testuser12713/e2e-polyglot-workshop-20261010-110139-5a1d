import { Navigate, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { useSession, type Session } from "./session";

// Route guard for every workshop screen. Without a usable session the visitor
// is sent to the login screen and the intended target is remembered so the
// login can return there. The session is reactive (useSession), so clearing it
// — for example after a workshop call answered 401 — instantly re-renders this
// guard and redirects back to login.

function isUsable(session: Session | null): boolean {
  if (!session || !session.token) return false;
  if (!session.expiresAt) return true;
  const at = Date.parse(session.expiresAt);
  if (!Number.isFinite(at)) return true;
  return at > Date.now();
}

export function RequireWorkshopAuth({ children }: { children: ReactNode }) {
  const session = useSession();
  const location = useLocation();

  if (!isUsable(session)) {
    return (
      <Navigate
        to="/werkstatt/anmeldung"
        replace
        state={{ from: `${location.pathname}${location.search}` }}
      />
    );
  }

  return <>{children}</>;
}
