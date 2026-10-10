import { Link } from "react-router-dom";
import "./portal.css";

interface Step {
  badgeStatus: string;
  badgeLabel: string;
  title: string;
  description: React.ReactNode;
}

const STEPS: Step[] = [
  {
    badgeStatus: "picked_up",
    badgeLabel: "Schritt 1",
    title: "Anfrage senden",
    description:
      "Kunden-, Fahrzeug- und Termindaten ausfüllen — das dauert nur wenige Minuten.",
  },
  {
    badgeStatus: "confirmed",
    badgeLabel: "Schritt 2",
    title: "Auftragsnummer erhalten",
    description: (
      <>
        Nach dem Absenden zeigt das Portal Ihre Auftragsnummer im Format{" "}
        <span className="mono">A-2026-0046</span>.
      </>
    ),
  },
  {
    badgeStatus: "completed",
    badgeLabel: "Schritt 3",
    title: "Status verfolgen",
    description: "Mit Auftragsnummer und Kennzeichen jederzeit Status und Rechnung einsehen.",
  },
];

interface OverviewColumn {
  label: string;
  entries: { to: string; label: string }[];
}

const OVERVIEW: OverviewColumn[] = [
  {
    label: "Kundenbereich",
    entries: [
      { to: "/", label: "Start" },
      { to: "/termin", label: "Termin anfragen" },
      { to: "/auftragsstatus", label: "Status abrufen" },
    ],
  },
  {
    label: "Werkstattbereich",
    entries: [
      { to: "/werkstatt/anmeldung", label: "Anmeldung" },
      { to: "/werkstatt/dashboard", label: "Dashboard" },
      { to: "/werkstatt/auftraege", label: "Aufträge" },
    ],
  },
  {
    label: "Rechtliches",
    entries: [
      { to: "/impressum", label: "Impressum" },
      { to: "/datenschutz", label: "Datenschutzerklärung" },
    ],
  },
];

// The customer portal landing page: the app's start screen. It shows the
// primary actions, the three-step flow and a page overview; every entry links
// to the page it names (AC-21: no dead controls).
export function PortalPage() {
  return (
    <section className="page" aria-labelledby="portal-heading">
      <header className="page__header">
        <h1 id="portal-heading">Kundenportal</h1>
        <p className="page__description">
          Termine anfragen und den Stand Ihrer Reparatur jederzeit verfolgen.
        </p>
      </header>

      <section aria-label="Hauptaktionen" className="portal-grid portal-grid--2">
        <article className="card">
          <div className="portal-card-body">
            <h2 className="portal-card-title">Termin anfragen</h2>
            <p className="portal-lead">
              Beschreiben Sie Ihr Anliegen und wählen Sie einen Wunschtermin. Sie erhalten sofort
              eine Auftragsnummer.
            </p>
            <div>
              <Link to="/termin" className="btn btn-primary">
                Termin anfragen
              </Link>
            </div>
          </div>
        </article>
        <article className="card">
          <div className="portal-card-body">
            <h2 className="portal-card-title">Auftragsstatus abrufen</h2>
            <p className="portal-lead">
              Geben Sie Auftragsnummer und Kennzeichen ein und sehen Sie Status, Verlauf und
              Rechnung.
            </p>
            <div>
              <Link to="/auftragsstatus" className="btn btn-secondary">
                Status abrufen
              </Link>
            </div>
          </div>
        </article>
      </section>

      <section aria-labelledby="portal-ablauf-heading">
        <h2 id="portal-ablauf-heading" className="portal-section-title">
          So funktioniert es
        </h2>
        <div className="portal-grid portal-grid--3">
          {STEPS.map((step) => (
            <article key={step.title} className="card">
              <div className="portal-card-body">
                <span className="status-badge" data-status={step.badgeStatus}>
                  {step.badgeLabel}
                </span>
                <h3 className="portal-card-title">{step.title}</h3>
                <p className="portal-lead">{step.description}</p>
              </div>
            </article>
          ))}
        </div>
      </section>

      <section className="card" aria-labelledby="portal-uebersicht-heading">
        <div className="card__header">
          <h2 id="portal-uebersicht-heading" className="portal-card-title">
            Seitenübersicht
          </h2>
        </div>
        <div className="portal-grid portal-grid--3">
          {OVERVIEW.map((column) => (
            <div key={column.label}>
              <p className="portal-overview-label">{column.label}</p>
              <ul className="portal-list">
                {column.entries.map((entry) => (
                  <li key={entry.to}>
                    <Link to={entry.to}>{entry.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>
    </section>
  );
}
