import "@testing-library/jest-dom/vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { clearSession, setSession } from "./auth/session";

// The workshop screens sit behind RequireWorkshopAuth; rendering them requires
// a valid session, which is seeded here and cleared after every test.
const PROTECTED_PATHS = ["/werkstatt/auftraege", "/werkstatt/auftraege/1", "/werkstatt/dashboard"];

function renderAt(path: string) {
  if (PROTECTED_PATHS.includes(path)) {
    setSession("test-token", new Date(Date.now() + 3_600_000).toISOString());
  } else {
    clearSession();
  }
  return render(
    <MemoryRouter initialEntries={[path]}>
      <App />
    </MemoryRouter>,
  );
}

afterEach(() => {
  clearSession();
});

const ROUTES: { path: string; heading: string }[] = [
  { path: "/termin", heading: "Termin anfragen" },
  { path: "/auftragsstatus", heading: "Auftragsstatus" },
  { path: "/werkstatt/anmeldung", heading: "Anmeldung Werkstatt" },
  { path: "/werkstatt/auftraege", heading: "Auftragsliste" },
  { path: "/werkstatt/auftraege/1", heading: "Auftragsdetail" },
  { path: "/werkstatt/dashboard", heading: "Dashboard" },
  { path: "/impressum", heading: "Impressum" },
  { path: "/datenschutz", heading: "Datenschutzerklärung" },
];

describe("app shell", () => {
  it("renders the primary navigation with the customer and workshop entries", () => {
    renderAt("/termin");
    const nav = screen.getByRole("navigation", { name: "Hauptnavigation" });
    for (const label of [
      "Termin anfragen",
      "Auftragsstatus",
      "Anmeldung",
      "Auftragsliste",
      "Dashboard",
    ]) {
      expect(within(nav).getByRole("link", { name: label })).toBeInTheDocument();
    }
  });

  it.each(ROUTES)("route $path is reachable and renders its heading", ({ path, heading }) => {
    renderAt(path);
    expect(screen.getByRole("heading", { level: 1, name: heading })).toBeInTheDocument();
  });

  it.each(ROUTES)("shows the footer with both legal links on $path", ({ path }) => {
    renderAt(path);
    const footer = screen.getByRole("navigation", { name: "Rechtliches" });
    expect(within(footer).getByRole("link", { name: "Impressum" })).toHaveAttribute(
      "href",
      "/impressum",
    );
    expect(within(footer).getByRole("link", { name: "Datenschutzerklärung" })).toHaveAttribute(
      "href",
      "/datenschutz",
    );
  });

  it("links each primary-navigation entry to its own route", () => {
    renderAt("/termin");
    const nav = screen.getByRole("navigation", { name: "Hauptnavigation" });
    expect(within(nav).getByRole("link", { name: "Termin anfragen" })).toHaveAttribute(
      "href",
      "/termin",
    );
    expect(within(nav).getByRole("link", { name: "Auftragsstatus" })).toHaveAttribute(
      "href",
      "/auftragsstatus",
    );
    expect(within(nav).getByRole("link", { name: "Anmeldung" })).toHaveAttribute(
      "href",
      "/werkstatt/anmeldung",
    );
    expect(within(nav).getByRole("link", { name: "Auftragsliste" })).toHaveAttribute(
      "href",
      "/werkstatt/auftraege",
    );
    expect(within(nav).getByRole("link", { name: "Dashboard" })).toHaveAttribute(
      "href",
      "/werkstatt/dashboard",
    );
  });

  it("navigates to the target page when a navigation entry is clicked", async () => {
    const user = userEvent.setup();
    renderAt("/termin");
    const nav = screen.getByRole("navigation", { name: "Hauptnavigation" });
    await user.click(within(nav).getByRole("link", { name: "Auftragsstatus" }));
    expect(
      screen.getByRole("heading", { level: 1, name: "Auftragsstatus" }),
    ).toBeInTheDocument();
  });

  it("renders the customer portal landing on the root path", () => {
    renderAt("/");
    expect(
      screen.getByRole("heading", { level: 1, name: "Kundenportal" }),
    ).toBeInTheDocument();
  });
});

describe("area switch", () => {
  it("sends a visitor without a session from the customer area to the workshop login when Werkstatt is clicked", async () => {
    const user = userEvent.setup();
    renderAt("/");
    const areaSwitch = screen.getByRole("navigation", { name: "Bereich wählen" });
    const werkstatt = within(areaSwitch).getByRole("link", { name: "Werkstatt" });
    expect(werkstatt).toHaveAttribute("href", "/werkstatt");

    await user.click(werkstatt);

    expect(
      screen.getByRole("heading", { level: 1, name: "Anmeldung Werkstatt" }),
    ).toBeInTheDocument();
  });

  it("opens the workshop area for a signed-in employee when Werkstatt is clicked", async () => {
    const user = userEvent.setup();
    setSession("test-token", new Date(Date.now() + 3_600_000).toISOString());
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>,
    );

    const areaSwitch = screen.getByRole("navigation", { name: "Bereich wählen" });
    await user.click(within(areaSwitch).getByRole("link", { name: "Werkstatt" }));

    expect(screen.getByRole("heading", { level: 1, name: "Dashboard" })).toBeInTheDocument();
  });

  it("switches from the workshop area back to the customer area", async () => {
    const user = userEvent.setup();
    renderAt("/werkstatt/dashboard");
    const areaSwitch = screen.getByRole("navigation", { name: "Bereich wählen" });

    await user.click(within(areaSwitch).getByRole("link", { name: "Kundenbereich" }));

    expect(screen.getByRole("heading", { level: 1, name: "Kundenportal" })).toBeInTheDocument();
  });
});
