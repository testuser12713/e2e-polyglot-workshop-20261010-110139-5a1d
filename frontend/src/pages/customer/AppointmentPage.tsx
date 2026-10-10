import { useState, type FormEvent, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ApiError } from "../../api/client";
import type { AppointmentRequest } from "../../api/types";
import { ErrorState } from "../../components/states";
import { submitAppointment } from "./appointmentApi";

// The public appointment form (screen: design/mockups/termin-anfragen.html).
// One form for customer data, vehicle data, the requested date and the problem
// description. Field errors appear only once a field was typed into or the form
// was submitted — the untouched form starts neutral.

type FieldName =
  | "name"
  | "email"
  | "phone"
  | "plate"
  | "brand"
  | "model"
  | "mileageKm"
  | "requestedDate"
  | "problemDescription";

type Values = Record<FieldName, string>;
type Touched = Partial<Record<FieldName, boolean>>;

interface ServerError {
  title: string;
  message: string;
  code: string;
}

const EMPTY: Values = {
  name: "",
  email: "",
  phone: "",
  plate: "",
  brand: "",
  model: "",
  mileageKm: "",
  requestedDate: "",
  problemDescription: "",
};

const FIELDS: readonly FieldName[] = [
  "name",
  "email",
  "phone",
  "plate",
  "brand",
  "model",
  "mileageKm",
  "requestedDate",
  "problemDescription",
];

const EMAIL_RE = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;
// A German plate: 1–3 letter region, then 1–2 letters and up to 4 digits,
// optionally separated by a space or dash. Validation is lenient on purpose.
const PLATE_RE = /^[A-ZÄÖÜ]{1,3}[ -]?[A-ZÄÖÜ]{1,2}[ -]?\d{1,4}$/i;

function validate(field: FieldName, value: string): string {
  const trimmed = value.trim();
  switch (field) {
    case "name":
      return trimmed ? "" : "Dieses Feld ist erforderlich.";
    case "email":
      if (!trimmed) return "Dieses Feld ist erforderlich.";
      return EMAIL_RE.test(trimmed)
        ? ""
        : "Bitte eine gültige E-Mail-Adresse eingeben.";
    case "phone":
      return trimmed ? "" : "Dieses Feld ist erforderlich.";
    case "plate":
      if (!trimmed) return "Dieses Feld ist erforderlich.";
      return PLATE_RE.test(trimmed)
        ? ""
        : "Bitte ein gültiges Kennzeichen eingeben.";
    case "brand":
    case "model":
      return trimmed ? "" : "Dieses Feld ist erforderlich.";
    case "mileageKm": {
      if (!trimmed) return "Dieses Feld ist erforderlich.";
      const km = Number(trimmed);
      return Number.isFinite(km) && km >= 0
        ? ""
        : "Bitte einen gültigen Kilometerstand eingeben.";
    }
    case "requestedDate":
      return trimmed ? "" : "Dieses Feld ist erforderlich.";
    case "problemDescription":
      return trimmed ? "" : "Dieses Feld ist erforderlich.";
  }
}

function errorTitle(code: string): string {
  switch (code) {
    case "duplicate_plate":
      return "Anfrage nicht möglich";
    case "validation_error":
      return "Bitte prüfen Sie Ihre Eingaben";
    default:
      return "Die Anfrage konnte nicht gesendet werden";
  }
}

function errorMessage(err: ApiError): string {
  if (err.status === 0 || !err.message) {
    return "Der Server ist nicht erreichbar. Bitte versuchen Sie es erneut.";
  }
  if (err.code === "duplicate_plate") {
    return err.message || "Für dieses Kennzeichen besteht bereits eine Anfrage.";
  }
  return err.message;
}

export function AppointmentPage() {
  const [values, setValues] = useState<Values>(EMPTY);
  const [touched, setTouched] = useState<Touched>({});
  const [submitting, setSubmitting] = useState(false);
  const [serverError, setServerError] = useState<ServerError | null>(null);
  const [orderNumber, setOrderNumber] = useState<string | null>(null);

  const fieldError = (field: FieldName): string =>
    touched[field] ? validate(field, values[field]) : "";

  function handleChange(field: FieldName, value: string): void {
    setValues((prev) => ({ ...prev, [field]: value }));
    setTouched((prev) => (prev[field] ? prev : { ...prev, [field]: true }));
  }

  function handleBlur(field: FieldName): void {
    setTouched((prev) => ({ ...prev, [field]: true }));
  }

  function handleReset(): void {
    setValues(EMPTY);
    setTouched({});
    setServerError(null);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setServerError(null);

    const allTouched: Touched = {};
    for (const field of FIELDS) allTouched[field] = true;
    setTouched(allTouched);

    const invalid = FIELDS.filter((field) => validate(field, values[field]) !== "");
    if (invalid.length > 0) {
      document.getElementById(invalid[0])?.focus();
      return;
    }

    const payload: AppointmentRequest = {
      name: values.name.trim(),
      email: values.email.trim(),
      phone: values.phone.trim(),
      plate: values.plate.trim().toUpperCase(),
      brand: values.brand.trim(),
      model: values.model.trim(),
      mileageKm: Number(values.mileageKm),
      requestedDate: values.requestedDate,
      problemDescription: values.problemDescription.trim(),
    };

    setSubmitting(true);
    try {
      const response = await submitAppointment(payload);
      setOrderNumber(response.orderNumber);
    } catch (error) {
      if (error instanceof ApiError) {
        setServerError({
          title: errorTitle(error.code),
          message: errorMessage(error),
          code: error.code,
        });
      } else {
        setServerError({
          title: errorTitle("internal_error"),
          message: "Die Anfrage konnte nicht gesendet werden. Bitte versuchen Sie es erneut.",
          code: "internal_error",
        });
      }
    } finally {
      setSubmitting(false);
    }
  }

  const pageHeader = (
    <header className="page__header">
      <h1>Termin anfragen</h1>
      <p className="page__description">
        Füllen Sie die Angaben aus. Nach dem Absenden erhalten Sie eine Auftragsnummer.
      </p>
    </header>
  );

  if (orderNumber) {
    return (
      <section className="page">
        {pageHeader}
        <div className="notice notice--success" role="status">
          <span aria-hidden="true">✓</span>
          <div>
            <p>
              Ihre Anfrage wurde gesendet. Auftragsnummer:{" "}
              <strong className="mono">{orderNumber}</strong>
            </p>
            <p style={{ marginTop: "var(--space-1)" }}>
              <Link className="btn btn-secondary" to="/auftragsstatus">
                Status abrufen
              </Link>
            </p>
          </div>
        </div>
      </section>
    );
  }

  function renderField(
    field: FieldName,
    label: string,
    control: ReactNode,
  ): ReactNode {
    const error = fieldError(field);
    return (
      <div
        className={error ? "field field--invalid" : "field"}
        id={`field-${field}`}
      >
        <label className="field__label" htmlFor={field}>
          {label}
        </label>
        {control}
        <p className="field__error">{error}</p>
      </div>
    );
  }

  return (
    <section className="page">
      {pageHeader}

      {serverError ? (
        <ErrorState
          title={serverError.title}
          message={serverError.message}
          code={serverError.code}
        />
      ) : null}

      <section className="card" aria-label="Anfrageformular">
        <form onSubmit={handleSubmit} noValidate style={{ maxWidth: "560px" }}>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: "var(--space-3)",
              marginBottom: "var(--space-4)",
            }}
          >
            <h3>Kundendaten</h3>
            {renderField(
              "name",
              "Name",
              <input
                id="name"
                name="name"
                type="text"
                autoComplete="name"
                value={values.name}
                disabled={submitting}
                onChange={(e) => handleChange("name", e.target.value)}
                onBlur={() => handleBlur("name")}
              />,
            )}
            {renderField(
              "email",
              "E-Mail",
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                placeholder="name@beispiel.de"
                value={values.email}
                disabled={submitting}
                onChange={(e) => handleChange("email", e.target.value)}
                onBlur={() => handleBlur("email")}
              />,
            )}
            {renderField(
              "phone",
              "Telefon",
              <input
                id="phone"
                name="phone"
                type="tel"
                autoComplete="tel"
                placeholder="+49 30 12345678"
                value={values.phone}
                disabled={submitting}
                onChange={(e) => handleChange("phone", e.target.value)}
                onBlur={() => handleBlur("phone")}
              />,
            )}
          </div>

          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: "var(--space-3)",
              marginBottom: "var(--space-4)",
            }}
          >
            <h3>Fahrzeugdaten</h3>
            {renderField(
              "plate",
              "Kennzeichen",
              <input
                id="plate"
                name="plate"
                type="text"
                className="mono"
                placeholder="B-AB 1234"
                value={values.plate}
                disabled={submitting}
                onChange={(e) => handleChange("plate", e.target.value.toUpperCase())}
                onBlur={() => handleBlur("plate")}
              />,
            )}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
                gap: "var(--space-2)",
              }}
            >
              {renderField(
                "brand",
                "Marke",
                <input
                  id="brand"
                  name="brand"
                  type="text"
                  placeholder="VW"
                  value={values.brand}
                  disabled={submitting}
                  onChange={(e) => handleChange("brand", e.target.value)}
                  onBlur={() => handleBlur("brand")}
                />,
              )}
              {renderField(
                "model",
                "Modell",
                <input
                  id="model"
                  name="model"
                  type="text"
                  placeholder="Golf 7"
                  value={values.model}
                  disabled={submitting}
                  onChange={(e) => handleChange("model", e.target.value)}
                  onBlur={() => handleBlur("model")}
                />,
              )}
            </div>
            {renderField(
              "mileageKm",
              "Kilometerstand",
              <div
                className="field__control"
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "var(--space-1)",
                }}
              >
                <input
                  id="mileageKm"
                  name="mileageKm"
                  type="number"
                  min={0}
                  step={1}
                  placeholder="128400"
                  value={values.mileageKm}
                  disabled={submitting}
                  onChange={(e) => handleChange("mileageKm", e.target.value)}
                  onBlur={() => handleBlur("mileageKm")}
                  style={{
                    flex: "1 1 auto",
                    minWidth: 0,
                    minHeight: "auto",
                    padding: 0,
                    border: "none",
                    background: "transparent",
                  }}
                />
                <span className="muted" aria-hidden="true">
                  km
                </span>
              </div>,
            )}
          </div>

          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: "var(--space-3)",
              marginBottom: "var(--space-4)",
            }}
          >
            <h3>Termin &amp; Anliegen</h3>
            {renderField(
              "requestedDate",
              "Wunschtermin",
              <input
                id="requestedDate"
                name="requestedDate"
                type="date"
                value={values.requestedDate}
                disabled={submitting}
                onChange={(e) => handleChange("requestedDate", e.target.value)}
                onBlur={() => handleBlur("requestedDate")}
              />,
            )}
            {renderField(
              "problemDescription",
              "Problembeschreibung",
              <textarea
                id="problemDescription"
                name="problemDescription"
                placeholder="Beschreiben Sie das Problem möglichst genau."
                value={values.problemDescription}
                disabled={submitting}
                onChange={(e) => handleChange("problemDescription", e.target.value)}
                onBlur={() => handleBlur("problemDescription")}
              />,
            )}
          </div>

          <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-1)" }}>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitting}
              aria-busy={submitting}
            >
              {submitting ? "Wird gesendet …" : "Anfrage senden"}
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={handleReset}
              disabled={submitting}
            >
              Abbrechen
            </button>
          </div>
        </form>
      </section>
    </section>
  );
}
