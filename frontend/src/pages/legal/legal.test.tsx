import "@testing-library/jest-dom/vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { App } from "../../App";

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <App />
    </MemoryRouter>,
  );
}

const IMPRESSUM_SECTIONS = [
  "Anbieter",
  "Verantwortlich für den Inhalt",
  "Registereintrag und Kammer",
  "Streitschlichtung",
  "Haftung für Inhalte und Links",
  "Urheberrecht",
];

const DATENSCHUTZ_SECTIONS = [
  "Verantwortlicher",
  "Zwecke der Verarbeitung",
  "Rechtsgrundlagen",
  "Arten der verarbeiteten Daten",
  "Empfänger der Daten",
  "Empfänger von Benachrichtigungen",
  "Speicherdauer",
  "Keine Ressourcen Dritter",
  "Ihre Rechte",
];

describe("Impressum page", () => {
  it("renders every section of its approved screen in order", () => {
    renderAt("/impressum");
    expect(screen.getByRole("heading", { level: 1, name: "Impressum" })).toBeInTheDocument();
    const headings = screen
      .getAllByRole("heading", { level: 2 })
      .map((heading) => heading.textContent);
    expect(headings).toEqual(IMPRESSUM_SECTIONS);
  });

  it("names the provider and the responsible person", () => {
    renderAt("/impressum");
    expect(screen.getByText("Kfz-Meisterwerkstatt Vogel")).toBeInTheDocument();
    expect(screen.getByText("Thomas Vogel")).toBeInTheDocument();
    expect(screen.getByText("kontakt@werkstatt-vogel.de")).toBeInTheDocument();
  });
});

describe("Datenschutz page", () => {
  it("renders every section of its approved screen in order", () => {
    renderAt("/datenschutz");
    expect(
      screen.getByRole("heading", { level: 1, name: "Datenschutzerklärung" }),
    ).toBeInTheDocument();
    const headings = screen
      .getAllByRole("heading", { level: 2 })
      .map((heading) => heading.textContent);
    expect(headings).toEqual(DATENSCHUTZ_SECTIONS);
  });

  it("names the responsible party", () => {
    renderAt("/datenschutz");
    const list = screen.getByRole("list");
    expect(within(list).getByText(/Kundendaten: Name, E-Mail-Adresse, Telefonnummer/)).toBeInTheDocument();
    expect(
      within(list).getByText(/Fahrzeugdaten: Kennzeichen, Marke, Modell, Kilometerstand/),
    ).toBeInTheDocument();
    expect(within(list).getByText(/Auftragsdaten:/)).toBeInTheDocument();
    expect(within(list).getByText(/Rechnungsdaten:/)).toBeInTheDocument();
  });
});

describe("footer legal links", () => {
  it("opens the Impressum from another page", async () => {
    const user = userEvent.setup();
    renderAt("/termin");
    const footer = screen.getByRole("navigation", { name: "Rechtliches" });
    await user.click(within(footer).getByRole("link", { name: "Impressum" }));
    expect(screen.getByRole("heading", { level: 1, name: "Impressum" })).toBeInTheDocument();
  });

  it("opens the Datenschutzerklärung from another page", async () => {
    const user = userEvent.setup();
    renderAt("/termin");
    const footer = screen.getByRole("navigation", { name: "Rechtliches" });
    await user.click(within(footer).getByRole("link", { name: "Datenschutzerklärung" }));
    expect(
      screen.getByRole("heading", { level: 1, name: "Datenschutzerklärung" }),
    ).toBeInTheDocument();
  });
});
