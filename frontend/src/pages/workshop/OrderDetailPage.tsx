import { useCallback, useEffect, useState, type FormEvent } from "react";
import { useParams } from "react-router-dom";
import { ApiError } from "../../api/client";
import {
  ORDER_STATUS_LABELS,
  ORDER_STATUS_ORDER,
  type CreateOrderItemInput,
  type Order,
  type OrderItem,
  type OrderStatus,
  type StatusEntry,
} from "../../api/types";
import { Empty, ErrorState, Loading } from "../../components/states";
import { addOrderItems, changeOrderStatus, fetchOrder } from "./orderDetailApi";
import "./orderDetail.css";

// The order-detail screen of the workshop area. It shows one order with its
// vehicle, the positions and the status history, lets the staff member confirm
// the order, record positions (labor time and parts) and advance the status.
// Only the single allowed forward transition is ever offered as a button; a
// forbidden one is not shown as clickable. Every call goes through the shared
// api client, and a rejected step (e.g. 409 invalid_transition) is shown as a
// visible message while the shown status stays unchanged.

// Keyed by the CURRENT status: the action that moves it to the next one.
const ACTION_LABEL: Partial<Record<OrderStatus, string>> = {
  requested: "Auftrag bestätigen",
  confirmed: "In Arbeit setzen",
  in_progress: "Fertig melden",
  completed: "Abholung bestätigen",
};

const LABOR_DESCRIPTION = "Arbeitszeit: Werkstattleistung";

function formatDate(value: string): string {
  const [year, month, day] = value.slice(0, 10).split("-");
  if (!year || !month || !day) return value;
  return `${day}.${month}.${year}`;
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Europe/Berlin",
  }).format(date);
}

function formatMoney(cents: number): string {
  return `${(cents / 100).toLocaleString("de-DE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} €`;
}

function formatHours(hours: number): string {
  return `${hours.toLocaleString("de-DE", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })} h`;
}

function formatQuantity(quantity: number): string {
  return quantity.toLocaleString("de-DE");
}

function nextStatusOf(status: OrderStatus): OrderStatus | null {
  const index = ORDER_STATUS_ORDER.indexOf(status);
  if (index < 0 || index >= ORDER_STATUS_ORDER.length - 1) return null;
  return ORDER_STATUS_ORDER[index + 1];
}

function StatusBadge({ status, dimmed }: { status: OrderStatus; dimmed?: boolean }) {
  return (
    <span
      className={dimmed ? "status-badge status-badge--dimmed" : "status-badge"}
      data-status={status}
    >
      {ORDER_STATUS_LABELS[status]}
    </span>
  );
}

interface Notice {
  kind: "success" | "error";
  message: string;
  code?: string;
}

function NoticeBar({ notice }: { notice: Notice }) {
  return (
    <div
      className={`notice notice--${notice.kind}`}
      role={notice.kind === "error" ? "alert" : "status"}
    >
      <div>
        <p>{notice.message}</p>
        {notice.code ? <p className="notice__code">Code: {notice.code}</p> : null}
      </div>
    </div>
  );
}

function PositionsTable({ items }: { items: OrderItem[] }) {
  return (
    <table className="od-positions">
      <thead>
        <tr>
          <th>Bezeichnung</th>
          <th className="od-num">Menge</th>
          <th className="od-num">Einzelpreis</th>
          <th className="od-num">Gesamt</th>
        </tr>
      </thead>
      <tbody>
        {items.map((item) => (
          <tr key={item.id}>
            <td>{item.description}</td>
            <td className="od-num">
              {item.kind === "labor"
                ? item.hours !== null
                  ? formatHours(item.hours)
                  : "—"
                : item.quantity !== null
                  ? formatQuantity(item.quantity)
                  : "—"}
            </td>
            <td className="od-num">
              {item.unitPriceCents !== null ? formatMoney(item.unitPriceCents) : "—"}
            </td>
            <td className="od-num">{formatMoney(item.amountCents)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

interface LaborCaptureProps {
  disabled: boolean;
  onAdd: (input: CreateOrderItemInput) => Promise<boolean>;
}

function LaborCapture({ disabled, onAdd }: LaborCaptureProps) {
  const [hours, setHours] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const parsed = Number.parseFloat(hours.replace(",", "."));
  const valid = Number.isFinite(parsed) && parsed > 0;
  const showError = submitted && !valid;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSubmitted(true);
    if (!valid) return;
    const ok = await onAdd({ kind: "labor", description: LABOR_DESCRIPTION, hours: parsed });
    if (ok) {
      setHours("");
      setSubmitted(false);
    }
  }

  return (
    <form className="od-capture__row" onSubmit={handleSubmit} noValidate>
      <div className={`field${showError ? " field--invalid" : ""}`}>
        <label className="field__label" htmlFor="od-labor-hours">
          Arbeitszeit
        </label>
        <div className="field__control">
          <input
            id="od-labor-hours"
            type="number"
            min="0"
            step="0.1"
            inputMode="decimal"
            placeholder="1,5"
            value={hours}
            disabled={disabled}
            onChange={(event) => setHours(event.target.value)}
          />
        </div>
        <p className="field__error" role="alert">
          {showError ? "Bitte eine Arbeitszeit größer 0 angeben." : ""}
        </p>
      </div>
      <button type="submit" className="btn btn-secondary" disabled={disabled}>
        Arbeitszeit hinzufügen
      </button>
    </form>
  );
}

interface PartCaptureProps {
  disabled: boolean;
  onAdd: (input: CreateOrderItemInput) => Promise<boolean>;
}

function PartCapture({ disabled, onAdd }: PartCaptureProps) {
  const [description, setDescription] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [unitPrice, setUnitPrice] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const parsedQuantity = Number.parseInt(quantity, 10);
  const parsedPrice = Number.parseFloat(unitPrice.replace(",", "."));
  const descriptionValid = description.trim().length > 0;
  const quantityValid = Number.isFinite(parsedQuantity) && parsedQuantity > 0;
  const priceValid = Number.isFinite(parsedPrice) && parsedPrice >= 0;
  const valid = descriptionValid && quantityValid && priceValid;
  const showError = submitted && !valid;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSubmitted(true);
    if (!valid) return;
    const ok = await onAdd({
      kind: "part",
      description: description.trim(),
      quantity: parsedQuantity,
      unitPriceCents: Math.round(parsedPrice * 100),
    });
    if (ok) {
      setDescription("");
      setQuantity("1");
      setUnitPrice("");
      setSubmitted(false);
    }
  }

  return (
    <form className="od-capture" onSubmit={handleSubmit} noValidate>
      <div className={`field${submitted && !descriptionValid ? " field--invalid" : ""}`}>
        <label className="field__label" htmlFor="od-part-description">
          Bezeichnung
        </label>
        <div className="field__control">
          <input
            id="od-part-description"
            type="text"
            placeholder="z. B. Luftfilter"
            value={description}
            disabled={disabled}
            onChange={(event) => setDescription(event.target.value)}
          />
        </div>
      </div>
      <div className="od-capture__grid">
        <div className={`field${submitted && !quantityValid ? " field--invalid" : ""}`}>
          <label className="field__label" htmlFor="od-part-quantity">
            Menge
          </label>
          <div className="field__control">
            <input
              id="od-part-quantity"
              type="number"
              min="0"
              step="1"
              inputMode="numeric"
              value={quantity}
              disabled={disabled}
              onChange={(event) => setQuantity(event.target.value)}
            />
          </div>
        </div>
        <div className={`field${submitted && !priceValid ? " field--invalid" : ""}`}>
          <label className="field__label" htmlFor="od-part-price">
            Einzelpreis
          </label>
          <div className="field__control">
            <input
              id="od-part-price"
              type="number"
              min="0"
              step="0.01"
              inputMode="decimal"
              placeholder="24,90"
              value={unitPrice}
              disabled={disabled}
              onChange={(event) => setUnitPrice(event.target.value)}
            />
          </div>
        </div>
      </div>
      <p className="field__error" role="alert">
        {showError ? "Bitte Bezeichnung, Menge und Einzelpreis ausfüllen." : ""}
      </p>
      <div>
        <button type="submit" className="btn btn-secondary" disabled={disabled}>
          Teil hinzufügen
        </button>
      </div>
    </form>
  );
}

function Timeline({ history }: { history: StatusEntry[] }) {
  if (history.length === 0) {
    return <Empty title="Noch keine Statuswechsel" />;
  }
  const entries = [...history].reverse();
  return (
    <ol className="od-timeline" aria-label="Statusverlauf">
      {entries.map((entry, index) => {
        const previous = entries[index + 1]?.status ?? null;
        return (
          <li key={`${entry.status}-${entry.changedAt}-${index}`} className="od-timeline__item">
            <span className="od-timeline__dot" aria-hidden="true" data-status={entry.status} />
            <p className="od-timeline__time">{formatDateTime(entry.changedAt)}</p>
            <div className="od-timeline__change">
              {previous ? (
                <StatusBadge status={previous} dimmed />
              ) : (
                <span className="muted">angelegt</span>
              )}
              <span aria-hidden="true">→</span>
              <StatusBadge status={entry.status} />
              {entry.employeeName ? (
                <span className="muted">· {entry.employeeName}</span>
              ) : null}
            </div>
          </li>
        );
      })}
    </ol>
  );
}

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setLoadError(null);
    try {
      setOrder(await fetchOrder(id));
    } catch (error) {
      setLoadError(error instanceof ApiError ? error : new ApiError(0, "internal_error", "Der Auftrag konnte nicht geladen werden."));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  async function handleAdvance(next: OrderStatus) {
    if (!id || busy) return;
    setBusy(true);
    setNotice(null);
    try {
      const updated = await changeOrderStatus(id, next);
      setOrder(updated);
      setNotice({
        kind: "success",
        message: `Status geändert auf „${ORDER_STATUS_LABELS[updated.status]}“.`,
      });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : null;
      setNotice({
        kind: "error",
        message: apiError?.message ?? "Der Status konnte nicht geändert werden.",
        code: apiError?.code,
      });
    } finally {
      setBusy(false);
    }
  }

  async function handleAddItem(input: CreateOrderItemInput): Promise<boolean> {
    if (!id || busy) return false;
    setBusy(true);
    setNotice(null);
    try {
      const updated = await addOrderItems(id, [input]);
      setOrder(updated);
      setNotice({ kind: "success", message: "Position hinzugefügt." });
      return true;
    } catch (error) {
      const apiError = error instanceof ApiError ? error : null;
      setNotice({
        kind: "error",
        message: apiError?.message ?? "Die Position konnte nicht gespeichert werden.",
        code: apiError?.code,
      });
      return false;
    } finally {
      setBusy(false);
    }
  }

  function renderContent() {
    if (loading) {
      return <Loading title="Auftrag wird geladen …" />;
    }
    if (loadError) {
      return (
        <ErrorState
          title="Auftrag konnte nicht geladen werden"
          message={loadError.message}
          code={loadError.code !== "internal_error" ? loadError.code : undefined}
          onRetry={() => void load()}
        />
      );
    }
    if (!order) {
      return <Empty title="Auftrag nicht gefunden" />;
    }

    const next = nextStatusOf(order.status);
    const canCapture = order.status === "confirmed" || order.status === "in_progress";
    const currentIndex = ORDER_STATUS_ORDER.indexOf(order.status);

    return (
      <>
        <header className="page__header od-detail__head">
          <div>
            <p className="od-order-number">{order.orderNumber}</p>
            <p className="od-subtitle">
              {order.brand} {order.model} · <span className="mono">{order.plate}</span>
            </p>
          </div>
          <StatusBadge status={order.status} />
        </header>

        {notice ? <NoticeBar notice={notice} /> : null}

        <div className="card">
          <dl className="od-meta">
            <div className="od-meta__item">
              <dt className="od-meta__term">Fahrzeug</dt>
              <dd className="od-meta__value">
                {order.brand} {order.model}
              </dd>
            </div>
            <div className="od-meta__item">
              <dt className="od-meta__term">Kennzeichen</dt>
              <dd className="od-meta__value mono">{order.plate}</dd>
            </div>
            <div className="od-meta__item">
              <dt className="od-meta__term">Wunschtermin</dt>
              <dd className="od-meta__value">{formatDate(order.requestedDate)}</dd>
            </div>
            <div className="od-meta__item">
              <dt className="od-meta__term">Angelegt</dt>
              <dd className="od-meta__value">{formatDateTime(order.createdAt)}</dd>
            </div>
            <div className="od-meta__item od-meta__item--wide">
              <dt className="od-meta__term">Anliegen</dt>
              <dd className="od-meta__value">{order.problemDescription}</dd>
            </div>
          </dl>
        </div>

        <div className="card">
          <h2 className="card__header">Status</h2>
          <ol className="od-stepper" aria-label="Status">
            {ORDER_STATUS_ORDER.map((status, index) => {
              const classes = ["od-stepper__step"];
              if (index <= currentIndex) classes.push("is-reached");
              if (index === currentIndex) classes.push("is-current");
              return (
                <li
                  key={status}
                  className={classes.join(" ")}
                  data-status={status}
                  aria-current={index === currentIndex ? "step" : undefined}
                >
                  <span className="od-stepper__dot" aria-hidden="true" />
                  <span className="od-stepper__label">{ORDER_STATUS_LABELS[status]}</span>
                </li>
              );
            })}
          </ol>
        </div>

        <div className="card">
          <h2 className="card__header">Positionen</h2>
          {order.items.length > 0 ? (
            <PositionsTable items={order.items} />
          ) : (
            <p className="muted">Noch keine Positionen erfasst.</p>
          )}
          {canCapture ? (
            <div className="od-capture">
              <h3 className="od-capture__title">Arbeitszeit erfassen</h3>
              <LaborCapture disabled={busy} onAdd={handleAddItem} />
              <h3 className="od-capture__title">Teil erfassen</h3>
              <PartCapture disabled={busy} onAdd={handleAddItem} />
            </div>
          ) : null}
        </div>

        <div className="card">
          <h2 className="card__header">Statusverlauf</h2>
          <Timeline history={order.history} />
        </div>

        <div className="od-actions">
          {next ? (
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void handleAdvance(next)}
              disabled={busy}
            >
              {ACTION_LABEL[order.status] ?? ORDER_STATUS_LABELS[next]}
            </button>
          ) : (
            <span className="btn btn-secondary" aria-disabled="true">
              Auftrag abgeschlossen
            </span>
          )}
        </div>
      </>
    );
  }

  return (
    <section className="page od-detail">
      <header className="page__header">
        <h1>Auftragsdetail</h1>
        <p className="page__description">
          Positionen, Statusverlauf und Statuswechsel eines Auftrags.
        </p>
      </header>
      {renderContent()}
    </section>
  );
}
