import { NavLink } from "react-router-dom";

export function Footer() {
  return (
    <footer className="app-footer">
      <div className="container app-footer__inner">
        <span className="app-footer__copy">
          © {new Date().getFullYear()} Kfz-Werkstatt Kundenportal
        </span>
        <nav className="app-footer__links" aria-label="Rechtliches">
          <NavLink to="/impressum">Impressum</NavLink>
          <NavLink to="/datenschutz">Datenschutzerklärung</NavLink>
        </nav>
      </div>
    </footer>
  );
}
