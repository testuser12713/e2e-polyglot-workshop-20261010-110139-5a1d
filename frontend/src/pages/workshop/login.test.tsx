import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../../api/client", async () => {
  const actual = await vi.importActual<typeof import("../../api/client")>("../../api/client");
  return { ...actual, apiFetch: vi.fn() };
});

import { apiFetch, ApiError } from "../../api/client";
import type { LoginResponse } from "../../api/types";
import { RequireWorkshopAuth } from "../../auth/RequireWorkshopAuth";
import { clearSession, getToken } from "../../auth/session";
import { LoginPage } from "./LoginPage";

const mockedApiFetch = vi.mocked(apiFetch);

function WorkshopProbe() {
  return <div>Werkstattbereich</div>;
}

beforeEach(() => {
  clearSession();
  mockedApiFetch.mockReset();
});

afterEach(() => {
  clearSession();
});

describe("RequireWorkshopAuth", () => {
  it("redirects to the login screen when there is no session", () => {
    render(
      <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
        <Routes>
          <Route path="/werkstatt/anmeldung" element={<div>Login screen</div>} />
          <Route
            path="/werkstatt/dashboard"
            element={
              <RequireWorkshopAuth>
                <WorkshopProbe />
              </RequireWorkshopAuth>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Login screen")).toBeInTheDocument();
    expect(screen.queryByText("Werkstattbereich")).not.toBeInTheDocument();
  });
});

describe("LoginPage", () => {
  function renderLogin() {
    return render(
      <MemoryRouter initialEntries={["/werkstatt/anmeldung"]}>
        <Routes>
          <Route path="/werkstatt/anmeldung" element={<LoginPage />} />
          <Route path="/werkstatt/dashboard" element={<WorkshopProbe />} />
        </Routes>
      </MemoryRouter>,
    );
  }

  it("stores the session and lands in the workshop area on success", async () => {
    const user = userEvent.setup();
    const expiresAt = new Date(Date.now() + 3_600_000).toISOString();
    mockedApiFetch.mockResolvedValue({ token: "test-token", expiresAt } as LoginResponse);

    renderLogin();
    await user.type(screen.getByLabelText("E-Mail", { selector: "input" }), "meister@werkstatt-vogel.de");
    await user.type(screen.getByLabelText("Passwort", { selector: "input" }), "werkstatt2026");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    await waitFor(() => expect(screen.getByText("Werkstattbereich")).toBeInTheDocument());
    expect(mockedApiFetch).toHaveBeenCalledWith(
      "/api/auth/login",
      expect.objectContaining({ method: "POST" }),
    );
    expect(getToken()).toBe("test-token");
  });

  it("shows a generic error message on wrong credentials", async () => {
    const user = userEvent.setup();
    mockedApiFetch.mockRejectedValue(new ApiError(401, "unauthorized", "unauthorized"));

    renderLogin();
    await user.type(screen.getByLabelText("E-Mail", { selector: "input" }), "wrong@example.com");
    await user.type(screen.getByLabelText("Passwort", { selector: "input" }), "wrong");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("E-Mail oder Passwort ist falsch.");
    expect(getToken()).toBeNull();
  });

  it("shows the rate-limit message on 429", async () => {
    const user = userEvent.setup();
    mockedApiFetch.mockRejectedValue(new ApiError(429, "rate_limited", "too many attempts"));

    renderLogin();
    await user.type(screen.getByLabelText("E-Mail", { selector: "input" }), "meister@werkstatt-vogel.de");
    await user.type(screen.getByLabelText("Passwort", { selector: "input" }), "werkstatt2026");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Zu viele Versuche. Bitte später erneut versuchen.");
    expect(alert).toHaveTextContent("rate_limited");
    expect(getToken()).toBeNull();
  });

  it("does not show field errors on an untouched form", () => {
    renderLogin();
    expect(screen.queryByText("Bitte E-Mail angeben.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte Passwort angeben.")).not.toBeInTheDocument();
  });

  it("shows a field error only after a submit attempt", async () => {
    const user = userEvent.setup();
    renderLogin();
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(screen.getByText("Bitte E-Mail angeben.")).toBeInTheDocument();
    expect(screen.getByText("Bitte Passwort angeben.")).toBeInTheDocument();
    expect(mockedApiFetch).not.toHaveBeenCalled();
  });
});
