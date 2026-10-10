import "@testing-library/jest-dom/vitest";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { App } from "../../App";
import { PortalPage } from "./PortalPage";

function renderPortal() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <PortalPage />
    </MemoryRouter>,
  );
}

describe("customer portal landing", () => {
  it("renders its heading and the one-line description", () => {
    renderPortal();
    expect(screen.getByRole("heading", { level: 1, name: "Kundenportal" })).toBeInTheDocument();
    expect(screen.getByText(/Stand Ihrer Reparatur/)).toBeInTheDocument();
  });

  it("links the main actions to the appointment and status pages", () => {
    renderPortal();
    const actions = screen.getByRole("region", { name: "Hauptaktionen" });
    expect(within(actions).getByRole("link", { name: "Termin anfragen" })).toHaveAttribute(
      "href",
      "/termin",
    );
    expect(within(actions).getByRole("link", { name: "Status abrufen" })).toHaveAttribute(
      "href",
      "/auftragsstatus",
    );
  });

  it("renders the three steps of the flow", () => {
    renderPortal();
    const ablauf = screen.getByRole("region", { name: "So funktioniert es" });
    expect(within(ablauf).getByRole("heading", { name: "Anfrage senden" })).toBeInTheDocument();
    expect(
      within(ablauf).getByRole("heading", { name: "Auftragsnummer erhalten" }),
    ).toBeInTheDocument();
    expect(within(ablauf).getByRole("heading", { name: "Status verfolgen" })).toBeInTheDocument();
  });

  it("links every overview entry to its page", () => {
    renderPortal();
    const uebersicht = screen.getByRole("region", { name: "Seitenübersicht" });
    const expected: { name: string; href: string }[] = [
      { name: "Start", href: "/" },
      { name: "Termin anfragen", href: "/termin" },
      { name: "Status abrufen", href: "/auftragsstatus" },
      { name: "Anmeldung", href: "/werkstatt/anmeldung" },
      { name: "Dashboard", href: "/werkstatt/dashboard" },
      { name: "Aufträge", href: "/werkstatt/auftraege" },
      { name: "Impressum", href: "/impressum" },
      { name: "Datenschutzerklärung", href: "/datenschutz" },
    ];
    for (const entry of expected) {
      expect(within(uebersicht).getByRole("link", { name: entry.name })).toHaveAttribute(
        "href",
        entry.href,
      );
    }
  });

  it("renders the portal landing as the app's start screen", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "Kundenportal" })).toBeInTheDocument();
  });
});
