import { NavLink, useLocation } from "react-router-dom";
import { clearSession, useSession } from "../auth/session";

interface NavEntry {
  to: string;
  label: string;
  end?: boolean;
}

// The area switch is the single control that toggles between the two areas
// (DESIGN.md, AppShell / Navigation). Each segment navigates to the area's
// entry route: the customer portal at "/" and the workshop area at
// "/werkstatt", which App.tsx redirects to its default landing
// (/werkstatt/dashboard) and RequireWorkshopAuth guards, so a visitor without a
// session lands on the login screen instead of staying put.
const CUSTOMER_AREA = "/";
const WORKSHOP_AREA = "/werkstatt";

interface AreaSegment {
  to: string;
  label: string;
}

const CUSTOMER_LINKS: NavEntry[] = [
  { to: "/termin", label: "Termin anfragen" },
  { to: "/auftragsstatus", label: "Auftragsstatus" },
];

const WORKSHOP_LINKS: NavEntry[] = [
  { to: "/werkstatt/dashboard", label: "Dashboard" },
  { to: "/werkstatt/auftraege", label: "Auftragsliste" },
  { to: "/werkstatt/anmeldung", label: "Anmeldung" },
];

const AREA_SEGMENTS: AreaSegment[] = [
  { to: CUSTOMER_AREA, label: "Kundenbereich" },
  { to: WORKSHOP_AREA, label: "Werkstatt" },
];

function segmentClass(active: boolean): string {
  return active ? "area-switch__segment is-active" : "area-switch__segment";
}

function tabClass({ isActive }: { isActive: boolean }): string {
  return isActive ? "tab is-active" : "tab";
}

export function Nav() {
  const session = useSession();
  const { pathname } = useLocation();
  const inWorkshop = pathname.startsWith("/werkstatt");

  return (
    <header className="app-header">
      <div className="container app-header__bar">
        <NavLink to="/termin" className="app-header__brand">
          <span className="app-header__mark" aria-hidden="true" />
          <span>Kfz-Werkstatt Portal</span>
        </NavLink>

        <nav className="area-switch" aria-label="Bereich wählen">
          {AREA_SEGMENTS.map((segment) => {
            const active = segment.to === WORKSHOP_AREA ? inWorkshop : !inWorkshop;
            return (
              <NavLink key={segment.to} to={segment.to} className={segmentClass(active)}>
                {segment.label}
              </NavLink>
            );
          })}
        </nav>

        <div className="app-header__session">
          {session ? (
            <>
              {session.email ? (
                <span className="app-header__email">{session.email}</span>
              ) : null}
              <button type="button" className="btn btn-ghost btn--sm" onClick={clearSession}>
                Abmelden
              </button>
            </>
          ) : (
            <NavLink to="/werkstatt/anmeldung" className="btn btn-ghost btn--sm">
              Anmelden
            </NavLink>
          )}
        </div>
      </div>

      <div className="container">
        <nav className="primary-tabs" aria-label="Hauptnavigation">
          {CUSTOMER_LINKS.map((entry) => (
            <NavLink key={entry.to} to={entry.to} end={entry.end} className={tabClass}>
              {entry.label}
            </NavLink>
          ))}
          {WORKSHOP_LINKS.map((entry) => (
            <NavLink key={entry.to} to={entry.to} end={entry.end} className={tabClass}>
              {entry.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  );
}
