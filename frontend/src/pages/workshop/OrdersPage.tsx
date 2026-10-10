import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ApiError } from "../../api/client";
import {
  ORDER_STATUS_LABELS,
  ORDER_STATUS_ORDER,
  type OrderStatus,
  type OrderSummary,
} from "../../api/types";
import { Empty, ErrorState, Loading } from "../../components/states";
import { fetchOrders } from "./ordersApi";
import "./OrdersPage.css";

// The order list screen of the workshop area. It shows every order with plate,
// vehicle, status, requested date and creation time, and lets the team filter
// by status and search by (partial) plate. Both controls refetch the list from
// GET /api/workshop/orders with the status/plate query parameters; loading,
// empty and error states stay visible while the data is fetched.

const DATE_TIME_FORMATTER = new Intl.DateTimeFormat("de-DE", {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Berlin",
});

// requestedDate is a plain calendar date ("YYYY-MM-DD"); formatting it through
// a Date would shift it by the viewer's timezone, so it is formatted directly.
function formatDate(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return value || "—";
  return `${match[3]}.${match[2]}.${match[1]}`;
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value || "—";
  return DATE_TIME_FORMATTER.format(parsed);
}

function formatVehicle(order: OrderSummary): string {
  return [order.brand, order.model].filter(Boolean).join(" ") || "—";
}

const REFETCH_DELAY_MS = 200;

export function OrdersPage() {
  const navigate = useNavigate();

  const [status, setStatus] = useState<OrderStatus | "">("");
  const [plate, setPlate] = useState("");
  const [orders, setOrders] = useState<OrderSummary[] | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    setError(null);

    const handle = window.setTimeout(() => {
      fetchOrders({ status, plate })
        .then((result) => {
          if (cancelled) return;
          setOrders(result);
          setBusy(false);
        })
        .catch((cause: unknown) => {
          if (cancelled) return;
          setError(
            cause instanceof ApiError
              ? cause
              : new ApiError(0, "internal_error", "Unbekannter Fehler"),
          );
          setBusy(false);
        });
    }, REFETCH_DELAY_MS);

    return () => {
      cancelled = true;
      window.clearTimeout(handle);
    };
  }, [status, plate, reloadKey]);

  const hasFilters = status !== "" || plate.trim() !== "";

  const resetFilters = useCallback(() => {
    setStatus("");
    setPlate("");
  }, []);

  const retry = useCallback(() => {
    setReloadKey((key) => key + 1);
  }, []);

  const openOrder = (id: number) => {
    navigate(`/werkstatt/auftraege/${id}`);
  };

  return (
    <section className="page">
      <header className="page__header">
        <h1>Auftragsliste</h1>
        <p className="page__description">
          Alle Aufträge durchsuchen, filtern und bearbeiten.
        </p>
      </header>

      <section className="card orders-filters" aria-label="Filter">
        <div className="filter-bar">
          <div className="field filter-bar__status">
            <label className="field__label" htmlFor="status-filter">
              Status
            </label>
            <select
              id="status-filter"
              value={status}
              onChange={(event) => setStatus(event.target.value as OrderStatus | "")}
            >
              <option value="">Alle Status</option>
              {ORDER_STATUS_ORDER.map((value) => (
                <option key={value} value={value}>
                  {ORDER_STATUS_LABELS[value]}
                </option>
              ))}
            </select>
          </div>

          <div className="field filter-bar__search">
            <label className="field__label" htmlFor="plate-search">
              Kennzeichen
            </label>
            <input
              id="plate-search"
              type="search"
              value={plate}
              placeholder="Kennzeichen suchen, z. B. B-AB"
              onChange={(event) => setPlate(event.target.value)}
            />
          </div>
        </div>

        {hasFilters ? (
          <div className="filter-pills">
            {status ? (
              <span className="filter-pill">
                Status: {ORDER_STATUS_LABELS[status]}
                <button
                  type="button"
                  className="filter-pill__remove"
                  aria-label="Statusfilter entfernen"
                  onClick={() => setStatus("")}
                >
                  ×
                </button>
              </span>
            ) : null}
            {plate.trim() ? (
              <span className="filter-pill">
                Kennzeichen: {plate.trim()}
                <button
                  type="button"
                  className="filter-pill__remove"
                  aria-label="Suchfilter entfernen"
                  onClick={() => setPlate("")}
                >
                  ×
                </button>
              </span>
            ) : null}
            <button type="button" className="btn btn-ghost btn--sm" onClick={resetFilters}>
              Filter zurücksetzen
            </button>
          </div>
        ) : null}
      </section>

      {error ? (
        <ErrorState
          message={error.message}
          code={error.code}
          onRetry={retry}
        />
      ) : orders === null ? (
        <Loading title="Aufträge werden geladen …" />
      ) : orders.length === 0 ? (
        <Empty
          title={hasFilters ? "Keine Aufträge gefunden" : "Noch keine Aufträge"}
          description={
            hasFilters
              ? "Für die aktuelle Suche und Filterung gibt es keine Treffer. Setzen Sie die Filter zurück, um alle Aufträge zu sehen."
              : "Hier erscheint der erste Auftrag, sobald Sie einen anlegen."
          }
          action={
            hasFilters ? (
              <button type="button" className="btn btn-secondary" onClick={resetFilters}>
                Filter zurücksetzen
              </button>
            ) : undefined
          }
        />
      ) : (
        <>
          {busy ? (
            <p className="orders-loading-note" role="status" aria-live="polite">
              Aufträge werden aktualisiert …
            </p>
          ) : null}
          <div className="orders-table-wrap" aria-busy={busy}>
            <table className="data-table orders-table">
              <thead>
                <tr>
                  <th>Auftragsnummer</th>
                  <th>Kennzeichen</th>
                  <th>Fahrzeug</th>
                  <th>Status</th>
                  <th className="numeric">Wunschtermin</th>
                  <th className="numeric">Erstellt</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => (
                  <tr
                    key={order.id}
                    tabIndex={0}
                    onClick={() => openOrder(order.id)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        openOrder(order.id);
                      }
                    }}
                  >
                    <td className="mono" data-label="Auftragsnummer">
                      <Link
                        to={`/werkstatt/auftraege/${order.id}`}
                        onClick={(event) => event.stopPropagation()}
                      >
                        {order.orderNumber}
                      </Link>
                    </td>
                    <td className="mono" data-label="Kennzeichen">
                      {order.plate || "—"}
                    </td>
                    <td data-label="Fahrzeug">{formatVehicle(order)}</td>
                    <td data-label="Status">
                      <span className="status-badge" data-status={order.status}>
                        {ORDER_STATUS_LABELS[order.status] ?? order.status}
                      </span>
                    </td>
                    <td className="numeric" data-label="Wunschtermin">
                      {formatDate(order.requestedDate)}
                    </td>
                    <td className="numeric" data-label="Erstellt">
                      {formatDateTime(order.createdAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  );
}
