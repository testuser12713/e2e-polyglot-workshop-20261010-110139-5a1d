import { useState, type FormEvent } from "react";
import { ApiError } from "../../api/client";
import {
  ORDER_STATUS_LABELS,
  ORDER_STATUS_ORDER,
  type Invoice,
  type InvoiceItem,
  type OrderStatus,
  type PublicOrder,
} from "../../api/types";
import { Empty, ErrorState, Loading } from "../../components/states";
import { fetchOrderStatus } from "./orderStatusApi";

// Value formatting follows the single rule of DESIGN.md: money is cents-derived
// with German separators and an € suffix, dates are TT.MM.JJJJ and timestamps
// TT.MM.JJJJ, HH:MM in local (Europe/Berlin) time.

export function formatMoney(cents: number): string {
  const value = (cents / 100).toLocaleString("de-DE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  return `${value} €`;
}

export function formatDate(iso: string): string {
  const [year, month, day] = iso.split("-");
  if (!year || !month || !day) return iso;
  return `${day}.${month}.${year}`;
}

export function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("de-DE", {
    timeZone: "Europe/Berlin",
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function statusLabel(status: OrderStatus): string {
  return ORDER_STATUS_LABELS[status] ?? status;
}

function formatPlate(plate: string): string {
  return plate.toUpperCase();
}

function formatHours(hours: number): string {
  return hours.toLocaleString("de-DE", { maximumFractionDigits: 1 });
}

function formatItemQuantity(item: InvoiceItem): string {
  if (item.kind === "labor" && item.hours != null) {
    return `${formatHours(item.hours)} h`;
  }
  if (item.quantity != null) {
    return item.quantity.toLocaleString("de-DE");
  }
  return "—";
}

interface StatusBadgeProps {
  status: OrderStatus;
  dimmed?: boolean;
}

function StatusBadge({ status, dimmed = false }: StatusBadgeProps) {
  return (
    <span
      className={`status-badge${dimmed ? " status-badge--dimmed" : ""}`}
      data-status={status}
    >
      {statusLabel(status)}
    </span>
  );
}

function OrderProgress({ current }: { current: OrderStatus }) {
  const currentIndex = ORDER_STATUS_ORDER.indexOf(current);
  return (
    <ol className="os-stepper" aria-label="Fortschritt">
      {ORDER_STATUS_ORDER.map((status, index) => {
        const reached = currentIndex >= 0 && index <= currentIndex;
        const isCurrent = index === currentIndex;
        return (
          <li
            key={status}
            className={`os-stepper__step${reached ? " is-reached" : ""}${
              isCurrent ? " is-current" : ""
            }`}
            data-status={status}
            aria-current={isCurrent ? "step" : undefined}
          >
            <span className="os-stepper__dot" aria-hidden="true" />
            <span className="os-stepper__label">{statusLabel(status)}</span>
          </li>
        );
      })}
    </ol>
  );
}

function StatusTimeline({ order }: { order: PublicOrder }) {
  if (order.history.length === 0) {
    return <p className="muted">Noch keine Statuswechsel.</p>;
  }
  const newestFirst = order.history
    .map((entry, index) => ({
      ...entry,
      previous: index > 0 ? order.history[index - 1].status : null,
    }))
    .reverse();

  return (
    <ol className="os-timeline">
      {newestFirst.map((entry, index) => (
        <li
          key={`${entry.status}-${entry.changedAt}-${index}`}
          className="os-timeline__item"
          data-status={entry.status}
        >
          <span className="os-timeline__dot" aria-hidden="true" />
          <p className="os-timeline__time">{formatDateTime(entry.changedAt)}</p>
          <p className="os-timeline__change">
            {entry.previous ? (
              <StatusBadge status={entry.previous} dimmed />
            ) : (
              <span className="muted">angelegt</span>
            )}
            <span className="os-timeline__arrow" aria-hidden="true">
              →
            </span>
            <StatusBadge status={entry.status} />
          </p>
        </li>
      ))}
    </ol>
  );
}

function InvoiceSummary({ invoice }: { invoice: Invoice }) {
  return (
    <div className="os-invoice">
      <p className="muted">
        Rechnungsnummer <span className="mono">{invoice.invoiceNumber}</span>
      </p>
      <div className="os-table-wrap">
        <table className="os-invoice-table">
          <thead>
            <tr>
              <th>Bezeichnung</th>
              <th className="numeric">Menge</th>
              <th className="numeric">Einzelpreis</th>
              <th className="numeric">Gesamt</th>
            </tr>
          </thead>
          <tbody>
            {invoice.items.map((item, index) => (
              <tr key={`${item.description}-${index}`}>
                <td>{item.description}</td>
                <td className="numeric">{formatItemQuantity(item)}</td>
                <td className="numeric">
                  {item.unitPriceCents != null ? formatMoney(item.unitPriceCents) : "—"}
                </td>
                <td className="numeric">{formatMoney(item.amountCents)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <dl className="os-amounts">
        <div className="os-amounts__row">
          <dt>Nettosumme</dt>
          <dd className="numeric">{formatMoney(invoice.netCents)}</dd>
        </div>
        <div className="os-amounts__row">
          <dt>MwSt. ({invoice.taxRatePercent} %)</dt>
          <dd className="numeric">{formatMoney(invoice.taxCents)}</dd>
        </div>
        <div className="os-amounts__total">
          <dt>Bruttosumme</dt>
          <dd className="numeric">{formatMoney(invoice.grossCents)}</dd>
        </div>
      </dl>
    </div>
  );
}

function OrderResult({ order }: { order: PublicOrder }) {
  return (
    <>
      <section className="card" aria-label="Auftragsstatus">
        <header className="card__header">
          <h2 className="mono">{order.orderNumber}</h2>
          <StatusBadge status={order.status} />
        </header>
        <OrderProgress current={order.status} />
        <dl className="os-detail">
          <dt>Kennzeichen</dt>
          <dd className="mono">{formatPlate(order.plate)}</dd>
          <dt>Wunschtermin</dt>
          <dd className="numeric">{formatDate(order.requestedDate)}</dd>
        </dl>
      </section>

      <section className="card" aria-label="Statusverlauf">
        <header className="card__header">
          <h2>Statusverlauf</h2>
        </header>
        <StatusTimeline order={order} />
      </section>

      <section className="card" aria-label="Rechnung">
        <header className="card__header">
          <h2>Rechnung</h2>
        </header>
        {order.invoice ? (
          <InvoiceSummary invoice={order.invoice} />
        ) : (
          <p className="muted">Rechnung noch nicht verfügbar.</p>
        )}
      </section>
    </>
  );
}

type LookupState =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "success"; order: PublicOrder }
  | { kind: "error"; mismatch: boolean; message: string; code?: string; orderNumber: string; plate: string };

interface FieldErrors {
  orderNumber?: string;
  plate?: string;
}

// Page-scoped styles for the components DESIGN.md specifies but the shared
// theme does not carry yet (stepper, timeline, invoice summary). All values come
// from the design tokens; class names are prefixed to stay out of the way of
// other pages.
const PAGE_STYLES = `
.os-detail { display: grid; grid-template-columns: 1fr; gap: 8px 24px; margin: 0; }
.os-detail dt { margin-top: 8px; font-size: 12px; font-weight: 500; letter-spacing: 0.02em; text-transform: uppercase; color: var(--color-muted); }
.os-detail dd { margin: 0; font-size: 15px; }
.os-hint { margin-top: var(--space-3); }
.os-form__actions { margin-top: var(--space-4); }
.os-stepper { display: flex; list-style: none; margin: 0; padding: 0; }
.os-stepper__step { position: relative; flex: 1 1 0; display: flex; flex-direction: column; align-items: center; gap: 8px; text-align: center; min-height: 44px; }
.os-stepper__step::before { content: ""; position: absolute; top: 5px; left: 0; right: 50%; height: 2px; background: var(--color-border); }
.os-stepper__step:first-child::before { display: none; }
.os-stepper__step.is-reached::before { background: var(--step-color, var(--color-accent)); }
.os-stepper__dot { position: relative; z-index: 1; width: 10px; height: 10px; border-radius: 50%; border: 1px solid var(--color-border-strong); background: var(--color-surface); }
.os-stepper__step.is-reached .os-stepper__dot { background: var(--step-color, var(--color-accent)); border-color: var(--step-color, var(--color-accent)); }
.os-stepper__step.is-current .os-stepper__dot { box-shadow: 0 0 0 2px var(--color-surface), 0 0 0 4px var(--color-accent-soft); }
.os-stepper__label { font-size: 13px; line-height: 20px; font-weight: 500; color: var(--color-muted); }
.os-stepper__step.is-reached .os-stepper__label { color: var(--color-fg); }
.os-stepper__step.is-current .os-stepper__label { font-weight: 600; color: var(--color-fg); }
.os-stepper__step[data-status="requested"] { --step-color: var(--color-status-requested); }
.os-stepper__step[data-status="confirmed"] { --step-color: var(--color-status-confirmed); }
.os-stepper__step[data-status="in_progress"] { --step-color: var(--color-status-in-progress); }
.os-stepper__step[data-status="completed"] { --step-color: var(--color-status-done); }
.os-stepper__step[data-status="picked_up"] { --step-color: var(--color-status-picked-up); }
.os-timeline { list-style: none; margin: 0; padding: 0; }
.os-timeline__item { position: relative; padding: 0 0 0 16px; margin-bottom: 16px; }
.os-timeline__item:last-child { margin-bottom: 0; }
.os-timeline__item::before { content: ""; position: absolute; left: 3px; top: 6px; bottom: -16px; width: 2px; background: var(--color-border); }
.os-timeline__item:last-child::before { display: none; }
.os-timeline__dot { position: absolute; left: 0; top: 6px; width: 8px; height: 8px; border-radius: 50%; background: var(--color-muted); }
.os-timeline__item[data-status="requested"] .os-timeline__dot { background: var(--color-status-requested); }
.os-timeline__item[data-status="confirmed"] .os-timeline__dot { background: var(--color-status-confirmed); }
.os-timeline__item[data-status="in_progress"] .os-timeline__dot { background: var(--color-status-in-progress); }
.os-timeline__item[data-status="completed"] .os-timeline__dot { background: var(--color-status-done); }
.os-timeline__item[data-status="picked_up"] .os-timeline__dot { background: var(--color-status-picked-up); }
.os-timeline__time { font-size: 13px; line-height: 20px; color: var(--color-muted); font-variant-numeric: tabular-nums; }
.os-timeline__change { margin-top: 2px; display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.os-timeline__arrow { color: var(--color-muted); }
.os-invoice { display: flex; flex-direction: column; gap: var(--space-3); }
.os-table-wrap { overflow-x: auto; border: 1px solid var(--color-border); border-radius: var(--radius-md); }
.os-invoice-table { width: 100%; border-collapse: collapse; }
.os-invoice-table th { padding: 8px 12px; background: var(--color-surface-sunken); border-bottom: 1px solid var(--color-border); color: var(--color-muted); font-size: 12px; font-weight: 500; letter-spacing: 0.02em; text-transform: uppercase; text-align: left; }
.os-invoice-table td { padding: 12px; border-bottom: 1px solid var(--color-border); font-size: 15px; }
.os-invoice-table tr:last-child td { border-bottom: none; }
.os-invoice-table td.numeric { font-variant-numeric: tabular-nums; }
.os-amounts { display: flex; flex-direction: column; gap: 16px; width: 100%; max-width: 320px; margin: 0 0 0 auto; }
.os-amounts__row, .os-amounts__total { display: flex; justify-content: space-between; font-size: 15px; }
.os-amounts__total { border-top: 1px solid var(--color-border); padding-top: 12px; font-size: 20px; line-height: 28px; font-weight: 600; }
.os-amounts dt, .os-amounts dd { margin: 0; }
@media (min-width: 640px) {
  .os-detail { grid-template-columns: max-content 1fr; }
}
@media (max-width: 639px) {
  .os-stepper { flex-direction: column; }
  .os-stepper__step { flex-direction: row; align-items: center; justify-content: flex-start; gap: 12px; text-align: left; }
  .os-stepper__step::before { top: auto; bottom: 50%; left: 4px; right: auto; width: 2px; height: 50%; }
  .os-invoice-table thead { display: none; }
  .os-invoice-table, .os-invoice-table tbody, .os-invoice-table tr, .os-invoice-table td { display: block; width: 100%; }
  .os-invoice-table tr { padding: 12px 0; border-bottom: 1px solid var(--color-border); }
  .os-invoice-table tr:last-child { border-bottom: none; }
  .os-invoice-table td { padding: 0; border: none; }
  .os-invoice-table td.numeric { text-align: left; }
  .os-invoice-table td.numeric::before { content: attr(data-label) ": "; color: var(--color-muted); font-size: 13px; }
  .os-amounts { max-width: none; margin-left: 0; }
}
`;

export function OrderStatusPage() {
  const [orderNumber, setOrderNumber] = useState("");
  const [plate, setPlate] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [state, setState] = useState<LookupState>({ kind: "idle" });

  async function runLookup(rawOrderNumber: string, rawPlate: string) {
    const queryOrderNumber = rawOrderNumber.trim();
    const queryPlate = rawPlate.trim();
    setState({ kind: "loading" });
    try {
      const order = await fetchOrderStatus(queryOrderNumber, queryPlate);
      setState({ kind: "success", order });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : null;
      const mismatch = apiError?.status === 404 || apiError?.code === "not_found";
      setState({
        kind: "error",
        mismatch,
        orderNumber: queryOrderNumber,
        plate: queryPlate,
        code: apiError?.code,
        message: mismatch
          ? "Bitte prüfen Sie Ihre Eingaben. Aus Datenschutzgründen geben wir keine Auftragsdaten preis."
          : (apiError?.message ?? "Der Status konnte nicht abgerufen werden."),
      });
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors: FieldErrors = {};
    if (!orderNumber.trim()) errors.orderNumber = "Bitte geben Sie die Auftragsnummer ein.";
    if (!plate.trim()) errors.plate = "Bitte geben Sie das Kennzeichen ein.";
    setFieldErrors(errors);
    if (errors.orderNumber || errors.plate) {
      setState({ kind: "idle" });
      return;
    }
    void runLookup(orderNumber, plate);
  }

  return (
    <section className="page">
      <style>{PAGE_STYLES}</style>
      <header className="page__header">
        <h1>Auftragsstatus</h1>
        <p className="page__description">
          Geben Sie Ihre Auftragsnummer und das Kennzeichen ein.
        </p>
      </header>

      <section className="card" aria-label="Statusabfrage">
        <form className="os-form" onSubmit={handleSubmit} noValidate>
          <div className={`field${fieldErrors.orderNumber ? " field--invalid" : ""}`}>
            <label className="field__label" htmlFor="order-status-number">
              Auftragsnummer
            </label>
            <div className="field__control">
              <input
                id="order-status-number"
                className="mono"
                type="text"
                value={orderNumber}
                placeholder="A-2026-0042"
                autoComplete="off"
                onChange={(event) => setOrderNumber(event.target.value)}
              />
            </div>
            {fieldErrors.orderNumber ? (
              <p className="field__error">{fieldErrors.orderNumber}</p>
            ) : null}
          </div>

          <div className={`field${fieldErrors.plate ? " field--invalid" : ""}`}>
            <label className="field__label" htmlFor="order-status-plate">
              Kennzeichen
            </label>
            <div className="field__control">
              <input
                id="order-status-plate"
                className="mono"
                type="text"
                value={plate}
                placeholder="B-AB 1234"
                autoComplete="off"
                onChange={(event) => setPlate(event.target.value)}
              />
            </div>
            {fieldErrors.plate ? <p className="field__error">{fieldErrors.plate}</p> : null}
          </div>

          <div className="os-form__actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={state.kind === "loading"}
            >
              {state.kind === "loading" ? "Wird abgerufen …" : "Status abrufen"}
            </button>
          </div>
        </form>
        <p className="muted os-hint">
          Beispiel zum Ausprobieren: <span className="mono">A-2026-0042</span> mit Kennzeichen{" "}
          <span className="mono">B-AB 1234</span>.
        </p>
      </section>

      {state.kind === "idle" ? (
        <Empty
          title="Noch keine Abfrage"
          description="Geben Sie Auftragsnummer und Kennzeichen ein, um den Status Ihrer Reparatur zu sehen."
        />
      ) : null}

      {state.kind === "loading" ? <Loading title="Status wird abgerufen …" /> : null}

      {state.kind === "error" ? (
        <ErrorState
          title={
            state.mismatch
              ? "Auftragsnummer und Kennzeichen passen nicht zusammen."
              : "Status konnte nicht abgerufen werden"
          }
          message={state.message}
          code={state.code}
          onRetry={() => void runLookup(state.orderNumber, state.plate)}
          retryLabel="Erneut versuchen"
        />
      ) : null}

      {state.kind === "success" ? <OrderResult order={state.order} /> : null}
    </section>
  );
}
