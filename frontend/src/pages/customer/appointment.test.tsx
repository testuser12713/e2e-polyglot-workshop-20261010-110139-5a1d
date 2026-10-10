import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppointmentPage } from "./AppointmentPage";

// The shared api client is the only thing that talks to the network; mock it so
// the page is exercised against a controlled contract.
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, apiFetch: vi.fn() };
});

import { apiFetch, ApiError } from "../../api/client";

const mockApiFetch = vi.mocked(apiFetch);

function renderPage() {
  return render(
    <MemoryRouter>
      <AppointmentPage />
    </MemoryRouter>,
  );
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText("Name"), "Anna Muster");
  await user.type(screen.getByLabelText("E-Mail"), "anna@example.com");
  await user.type(screen.getByLabelText("Telefon"), "+49 30 123456");
  await user.type(screen.getByLabelText("Kennzeichen"), "B-AB 1234");
  await user.type(screen.getByLabelText("Marke"), "VW");
  await user.type(screen.getByLabelText("Modell"), "Golf 7");
  await user.type(screen.getByLabelText("Kilometerstand"), "128400");
  fireEvent.change(screen.getByLabelText("Wunschtermin"), {
    target: { value: "2026-11-02" },
  });
  await user.type(screen.getByLabelText("Problembeschreibung"), "Bremsen quietschen.");
}

beforeEach(() => {
  mockApiFetch.mockReset();
});

describe("AppointmentPage", () => {
  it("shows the returned order number after a successful submit", async () => {
    const user = userEvent.setup();
    mockApiFetch.mockResolvedValueOnce({ orderNumber: "A-2026-0046", status: "requested" });

    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Anfrage senden" }));

    expect(await screen.findByText("A-2026-0046")).toBeInTheDocument();
    expect(screen.getByText(/Ihre Anfrage wurde gesendet/)).toBeInTheDocument();

    expect(mockApiFetch).toHaveBeenCalledTimes(1);
    const [path, init] = mockApiFetch.mock.calls[0];
    expect(path).toBe("/api/public/appointments");
    expect(init?.method).toBe("POST");
    const body = JSON.parse(String(init?.body));
    expect(body).toMatchObject({
      name: "Anna Muster",
      email: "anna@example.com",
      plate: "B-AB 1234",
      requestedDate: "2026-11-02",
      mileageKm: 128400,
    });
  });

  it("starts neutral and only shows field errors after a submit attempt", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.queryByText("Dieses Feld ist erforderlich.")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Anfrage senden" }));

    expect(
      (await screen.findAllByText("Dieses Feld ist erforderlich.")).length,
    ).toBeGreaterThan(0);
    expect(mockApiFetch).not.toHaveBeenCalled();
  });

  it("shows an e-mail field error only after the user typed into it", async () => {
    const user = userEvent.setup();
    renderPage();

    const email = screen.getByLabelText("E-Mail");
    expect(
      screen.queryByText("Bitte eine gültige E-Mail-Adresse eingeben."),
    ).not.toBeInTheDocument();

    await user.type(email, "keine-mail");

    expect(
      await screen.findByText("Bitte eine gültige E-Mail-Adresse eingeben."),
    ).toBeInTheDocument();
  });

  it("shows a visible message for a 409 duplicate plate", async () => {
    const user = userEvent.setup();
    mockApiFetch.mockRejectedValueOnce(
      new ApiError(409, "duplicate_plate", "Für dieses Kennzeichen besteht bereits eine Anfrage."),
    );

    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Anfrage senden" }));

    expect(
      await screen.findByText("Für dieses Kennzeichen besteht bereits eine Anfrage."),
    ).toBeInTheDocument();
    expect(screen.getByText(/duplicate_plate/)).toBeInTheDocument();
  });

  it("shows a visible message for a 422 validation error", async () => {
    const user = userEvent.setup();
    mockApiFetch.mockRejectedValueOnce(
      new ApiError(422, "validation_error", "Die Anfrage ist ungültig."),
    );

    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Anfrage senden" }));

    expect(await screen.findByText("Die Anfrage ist ungültig.")).toBeInTheDocument();
    expect(screen.getByText(/validation_error/)).toBeInTheDocument();
  });

  it("resets the form when Abbrechen is clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    const name = screen.getByLabelText("Name");
    await user.type(name, "Anna");
    expect(name).toHaveValue("Anna");

    await user.click(screen.getByRole("button", { name: "Abbrechen" }));
    await waitFor(() => expect(screen.getByLabelText("Name")).toHaveValue(""));
    expect(screen.queryByText("Dieses Feld ist erforderlich.")).not.toBeInTheDocument();
  });
});
