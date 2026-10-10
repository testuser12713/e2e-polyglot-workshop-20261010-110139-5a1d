import { NavLink, useLocation } from "react-router-dom";
import { clearSession, useSession } from "../auth/session";

interface NavEntry {
  to: string;
  label: string;
  end?: boolean;
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
          <NavLink
            to="/termin"
            className={inWorkshop ? "area-switch__segment" : "area-switch__segment is-active"}
          >
            Kundenbereich
          </NavLink>
          <NavLink
            to="/werkstatt/dashboard"
            className={inWorkshop ? "area-switch__segment is-active" : "area-switch__segment"}
          >
            Werkstatt
          </NavLink>
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
