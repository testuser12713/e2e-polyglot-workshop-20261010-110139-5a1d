import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { RequireWorkshopAuth } from "./RequireWorkshopAuth";
import { clearSession, getToken, useSession } from "./session";

// The store keeps the session in a module variable seeded at import time. A
// session that is only present in storage (restored document, another tab) must
// still be recognised the moment a guarded route renders — otherwise a
// signed-in employee clicking "Werkstatt" would be bounced to the login screen.
const STORAGE_KEY = "workshop.session.v1";

function futureExpiry(): string {
  return new Date(Date.now() + 3_600_000).toISOString();
}

function writeStored(token: string, expiresAt = futureExpiry()): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ token, expiresAt, email: null }));
}

beforeEach(() => {
  clearSession();
  window.localStorage.clear();
});

afterEach(() => {
  window.localStorage.clear();
});

describe("session store", () => {
  it("recognises a session that only exists in storage", () => {
    clearSession();
    writeStored("stored-token");

    expect(getToken()).toBe("stored-token");
  });

  it("does not resurrect a session after it was cleared", () => {
    writeStored("stored-token");
    expect(getToken()).toBe("stored-token");

    clearSession();

    expect(getToken()).toBeNull();
  });
});

describe("RequireWorkshopAuth", () => {
  it("renders the guarded route when the session is present at navigation", () => {
    writeStored("stored-token");

    render(
      <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
        <Routes>
          <Route path="/werkstatt/anmeldung" element={<div>Login screen</div>} />
          <Route
            path="/werkstatt/dashboard"
            element={
              <RequireWorkshopAuth>
                <div>Werkstattbereich</div>
              </RequireWorkshopAuth>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Werkstattbereich")).toBeInTheDocument();
    expect(screen.queryByText("Login screen")).not.toBeInTheDocument();
  });

  it("redirects to the login screen when there is no session", () => {
    render(
      <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
        <Routes>
          <Route path="/werkstatt/anmeldung" element={<div>Login screen</div>} />
          <Route
            path="/werkstatt/dashboard"
            element={
              <RequireWorkshopAuth>
                <div>Werkstattbereich</div>
              </RequireWorkshopAuth>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Login screen")).toBeInTheDocument();
  });
});

describe("useSession", () => {
  function Probe() {
    const session = useSession();
    return <div>{session ? `signed-in:${session.token}` : "signed-out"}</div>;
  }

  it("exposes a session that only exists in storage", () => {
    writeStored("stored-token");

    render(<Probe />);

    expect(screen.getByText("signed-in:stored-token")).toBeInTheDocument();
  });
});
