import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError } from "../../api/client";
import type { DashboardMetrics } from "../../api/types";
import { ErrorState, Loading } from "../../components/states";
import { fetchDashboardMetrics } from "./dashboardApi";
import "./dashboard.css";

// Money is stored and transferred as a whole number of cents; the product
// shows it in German formatting with a comma decimal separator, a dot as the
// thousands separator and a trailing euro sign, e.g. 841255 -> "8.412,55 €".
export function formatEuroFromCents(cents: number): string {
  const rounded = Math.round(cents);
  const negative = rounded < 0;
  const absolute = Math.abs(rounded);
  const euros = Math.floor(absolute / 100)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const remainder = (absolute % 100).toString().padStart(2, "0");
  return `${negative ? "-" : ""}${euros},${remainder} €`;
}

interface KpiCardProps {
  testId: string;
  label: string;
  value: string;
  sub: string;
  tone: string;
  to?: string;
}

function KpiCard({ testId, label, value, sub, tone, to }: KpiCardProps) {
  const content = (
    <>
      <p className="kpi-card__label">{label}</p>
      <p className="kpi-card__value">{value}</p>
      <p className="kpi-card__sub">{sub}</p>
    </>
  );
  const cardProps = {
    "data-testid": testId,
    className: "card kpi-card",
    style: { borderTopColor: tone },
  } as const;

  if (to) {
    return (
      <Link {...cardProps} to={to}>
        {content}
      </Link>
    );
  }
  return <div {...cardProps}>{content}</div>;
}

export function DashboardPage() {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    let active = true;
    fetchDashboardMetrics()
      .then((data) => {
        if (active) {
          setMetrics(data);
          setLoading(false);
        }
      })
      .catch((cause: unknown) => {
        if (!active) return;
        const apiError =
          cause instanceof ApiError
            ? cause
            : new ApiError(0, "internal_error", "Die Dashboard-Daten konnten nicht geladen werden.");
        setError(apiError);
        setMetrics(null);
        setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => load(), [load]);

  return (
    <section className="page">
      <header className="page__header">
        <h1>Dashboard</h1>
        <p className="page__description">
          Überblick über Aufträge und Umsatz der Werkstatt.
        </p>
      </header>

      {loading ? (
        <Loading />
      ) : error ? (
        <ErrorState
          message={error.message}
          code={error.code}
          onRetry={load}
        />
      ) : metrics ? (
        <>
          <section className="kpi-grid" aria-label="Kennzahlen">
            <KpiCard
              testId="kpi-requested"
              label="angefragt"
              value={String(metrics.openCounts.requested)}
              sub="offene Anfragen"
              tone="var(--color-status-requested)"
              to="/werkstatt/auftraege?status=requested"
            />
            <KpiCard
              testId="kpi-confirmed"
              label="bestätigt"
              value={String(metrics.openCounts.confirmed)}
              sub="warten auf Arbeitsbeginn"
              tone="var(--color-status-confirmed)"
              to="/werkstatt/auftraege?status=confirmed"
            />
            <KpiCard
              testId="kpi-in-progress"
              label="in Arbeit"
              value={String(metrics.openCounts.in_progress)}
              sub="in der Werkstatt"
              tone="var(--color-status-in-progress)"
              to="/werkstatt/auftraege?status=in_progress"
            />
            <KpiCard
              testId="kpi-done"
              label="heute fertig"
              value={String(metrics.completedToday)}
              sub="abholbereit"
              tone="var(--color-status-done)"
              to="/werkstatt/auftraege?status=completed"
            />
            <KpiCard
              testId="kpi-revenue"
              label="Umsatz laufender Monat"
              value={formatEuroFromCents(metrics.monthRevenueGrossCents)}
              sub="aus gestellten Rechnungen"
              tone="var(--color-accent)"
            />
          </section>

          <section className="dashboard-lists" aria-label="Listen">
            <article className="card">
              <div className="card__header">
                <h2 className="dashboard-card__title">Nächste Wunschtermine</h2>
                <Link className="btn btn-ghost" to="/werkstatt/auftraege">
                  Alle Aufträge
                </Link>
              </div>
              <p className="dashboard-list-card__body">
                Diese Übersicht ist noch nicht verfügbar.
              </p>
            </article>

            <article className="card">
              <div className="card__header">
                <h2 className="dashboard-card__title">Heute fertig geworden</h2>
              </div>
              <p className="dashboard-list-card__body">
                Diese Übersicht ist noch nicht verfügbar.
              </p>
            </article>
          </section>
        </>
      ) : null}
    </section>
  );
}
