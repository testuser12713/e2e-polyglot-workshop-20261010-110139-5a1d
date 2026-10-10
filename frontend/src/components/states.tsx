import type { ReactNode } from "react";

// Shared non-content states, one implementation used on every page (AC-21).
// Loading renders skeletons that mirror the layout, empty explains the next
// step, error shows the message plus an optional retry.

interface LoadingProps {
  title?: string;
  description?: string;
}

export function Loading({ title = "Wird geladen …", description }: LoadingProps) {
  return (
    <div className="state-panel state-panel--loading" role="status" aria-live="polite">
      <div className="state-panel__skeletons" aria-hidden="true">
        <span className="skeleton skeleton--100" />
        <span className="skeleton skeleton--80" />
        <span className="skeleton skeleton--60" />
      </div>
      <h2 className="state-panel__title">{title}</h2>
      {description ? <p className="state-panel__description">{description}</p> : null}
    </div>
  );
}

interface EmptyProps {
  title: string;
  description?: string;
  action?: ReactNode;
}

export function Empty({ title, description, action }: EmptyProps) {
  return (
    <div className="state-panel state-panel--empty">
      <span className="state-panel__icon" aria-hidden="true">
        ∅
      </span>
      <h2 className="state-panel__title">{title}</h2>
      {description ? <p className="state-panel__description">{description}</p> : null}
      {action}
    </div>
  );
}

interface ErrorStateProps {
  title?: string;
  message: string;
  code?: string;
  onRetry?: () => void;
  retryLabel?: string;
}

export function ErrorState({
  title = "Es ist ein Fehler aufgetreten",
  message,
  code,
  onRetry,
  retryLabel = "Erneut versuchen",
}: ErrorStateProps) {
  return (
    <div className="state-panel state-panel--error" role="alert">
      <span className="state-panel__icon" aria-hidden="true">
        !
      </span>
      <h2 className="state-panel__title">{title}</h2>
      <p className="state-panel__description">{message}</p>
      {code ? <p className="state-panel__code">Code: {code}</p> : null}
      {onRetry ? (
        <button type="button" className="btn btn-secondary" onClick={onRetry}>
          {retryLabel}
        </button>
      ) : null}
    </div>
  );
}
