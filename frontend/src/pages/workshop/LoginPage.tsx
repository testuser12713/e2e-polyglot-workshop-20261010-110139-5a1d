import { useId, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { apiFetch, ApiError } from "../../api/client";
import type { LoginResponse } from "../../api/types";
import { setSession } from "../../auth/session";

// The login screen of the workshop. It talks to the shared api client only, so
// the bearer token, JSON handling and the uniform API error body live in one
// place. On success the session is persisted (surviving a reload) and the user
// lands in the workshop area; on failure the API's message is shown — generic
// for 401 so it never reveals which field was wrong, explicit for 429.

interface Notice {
  variant: "error" | "warning";
  message: string;
  code?: string;
}

const CREDENTIALS_ERROR = "E-Mail oder Passwort ist falsch.";
const RATE_LIMIT_ERROR = "Zu viele Versuche. Bitte später erneut versuchen.";
const GENERIC_ERROR = "Die Anmeldung ist fehlgeschlagen. Bitte später erneut versuchen.";

function targetFrom(state: unknown): string {
  const from = (state as { from?: unknown } | null)?.from;
  if (
    typeof from === "string" &&
    from.startsWith("/werkstatt") &&
    from !== "/werkstatt/anmeldung"
  ) {
    return from;
  }
  return "/werkstatt/dashboard";
}

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const emailId = useId();
  const passwordId = useId();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [emailTouched, setEmailTouched] = useState(false);
  const [passwordTouched, setPasswordTouched] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);

  const emailError = email.trim() ? "" : "Bitte E-Mail angeben.";
  const passwordError = password ? "" : "Bitte Passwort angeben.";
  const showEmailError = (emailTouched || submitted) && emailError !== "";
  const showPasswordError = (passwordTouched || submitted) && passwordError !== "";

  async function onSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setSubmitted(true);
    if (emailError || passwordError) return;

    setSubmitting(true);
    setNotice(null);
    try {
      const data = await apiFetch<LoginResponse>("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email: email.trim(), password }),
      });
      setSession(data.token, data.expiresAt);
      navigate(targetFrom(location.state), { replace: true });
    } catch (error) {
      setNotice(toNotice(error));
      setSubmitting(false);
    }
  }

  return (
    <section className="page" aria-labelledby={`${emailId}-title`}>
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          paddingTop: "var(--space-4)",
          paddingBottom: "var(--space-4)",
        }}
      >
        <div className="card" style={{ width: "100%", maxWidth: "400px", gap: "var(--space-3)" }}>
          <div>
            <h1 id={`${emailId}-title`} style={{ fontSize: "20px", lineHeight: "28px" }}>
              Anmeldung Werkstatt
            </h1>
            <p className="muted" style={{ marginTop: "var(--space-1)" }}>
              Bitte melden Sie sich mit Ihrem Mitarbeiterkonto an.
            </p>
          </div>

          {notice ? (
            <div className={`notice notice--${notice.variant}`} role="alert">
              <div>
                <div>{notice.message}</div>
                {notice.code ? <div className="notice__code">Code: {notice.code}</div> : null}
              </div>
            </div>
          ) : null}

          <form onSubmit={onSubmit} noValidate>
            <div className={`field${showEmailError ? " field--invalid" : ""}`}>
              <label className="field__label" htmlFor={emailId}>
                E-Mail
              </label>
              <input
                id={emailId}
                type="email"
                autoComplete="username"
                autoFocus
                value={email}
                disabled={submitting}
                aria-invalid={showEmailError || undefined}
                aria-describedby={`${emailId}-error`}
                onChange={(event: ChangeEvent<HTMLInputElement>) => setEmail(event.target.value)}
                onBlur={() => setEmailTouched(true)}
              />
              <div className="field__error" id={`${emailId}-error`}>
                {showEmailError ? emailError : ""}
              </div>
            </div>

            <div className={`field${showPasswordError ? " field--invalid" : ""}`} style={{ marginBottom: 0 }}>
              <label className="field__label" htmlFor={passwordId}>
                Passwort
              </label>
              <div style={{ position: "relative" }}>
                <input
                  id={passwordId}
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  value={password}
                  disabled={submitting}
                  style={{ paddingRight: "52px" }}
                  aria-invalid={showPasswordError || undefined}
                  aria-describedby={`${passwordId}-error`}
                  onChange={(event: ChangeEvent<HTMLInputElement>) => setPassword(event.target.value)}
                  onBlur={() => setPasswordTouched(true)}
                />
                <button
                  type="button"
                  className="btn btn-ghost"
                  aria-label={showPassword ? "Passwort verbergen" : "Passwort anzeigen"}
                  aria-pressed={showPassword}
                  onClick={() => setShowPassword((value) => !value)}
                  style={{
                    position: "absolute",
                    right: "4px",
                    top: "50%",
                    transform: "translateY(-50%)",
                    minHeight: "36px",
                    width: "44px",
                    padding: 0,
                  }}
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="18"
                    height="18"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                </button>
              </div>
              <div className="field__error" id={`${passwordId}-error`}>
                {showPasswordError ? passwordError : ""}
              </div>
            </div>

            <div style={{ marginTop: "var(--space-4)" }}>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={submitting}
                style={{ width: "100%" }}
              >
                {submitting ? "Anmeldung läuft …" : "Anmelden"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </section>
  );
}

function toNotice(error: unknown): Notice {
  if (error instanceof ApiError) {
    if (error.status === 429) {
      return { variant: "warning", message: RATE_LIMIT_ERROR, code: error.code };
    }
    if (error.status === 401) {
      return { variant: "error", message: CREDENTIALS_ERROR, code: error.code };
    }
    if (error.status === 422) {
      return { variant: "error", message: error.message || GENERIC_ERROR, code: error.code };
    }
    return { variant: "error", message: error.message || GENERIC_ERROR, code: error.code };
  }
  return { variant: "error", message: GENERIC_ERROR };
}
