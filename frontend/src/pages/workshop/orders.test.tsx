import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, apiFetch } from "../../api/client";
import type { OrderSummary } from "../../api/types";
import { OrdersPage } from "./OrdersPage";

// The shared api client is mocked: the page must reach the network only through
// apiFetch, so mocking it exercises the screen without a running backend.
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, apiFetch: vi.fn() };
});

const apiFetchMock = vi.mocked(apiFetch);

const ORDERS: OrderSummary[] = [
  {
    id: 45,
    orderNumber: "A-2026-0045",
    plate: "B-GH 3456",
    brand: "Opel",
    model: "Corsa",
    status: "requested",
    requestedDate: "2026-10-16",
    createdAt: "2026-10-10T09:20:00Z",
  },
  {
    id: 44,
    orderNumber: "A-2026-0044",
    plate: "B-EF 9012",
    brand: "Ford",
    model: "Focus",
    status: "confirmed",
    requestedDate: "2026-10-17",
    createdAt: "2026-10-10T08:00:00Z",
  },
  {
    id: 43,
    orderNumber: "A-2026-0043",
    plate: "B-AB 1234",
    brand: "Audi",
    model: "A4 Avant",
    status: "in_progress",
    requestedDate: "2026-10-15",
    createdAt: "2026-10-09T12:05:00Z",
  },
];

const normalize = (value: string): string =>
  value.toUpperCase().replace(/[^A-Z0-9]/g, "");

// Mirrors the backend: status is an exact match, plate a partial, case- and
// separator-insensitive match.
function installFilteringMock(): void {
  apiFetchMock.mockImplementation((path: string) => {
    const url = new URL(path, "http://localhost");
    const status = url.searchParams.get("status") ?? "";
    const plate = url.searchParams.get("plate") ?? "";
    const needle = normalize(plate);
    const orders = ORDERS.filter(
      (order) =>
        (!status || order.status === status) &&
        (!needle || normalize(order.plate).includes(needle)),
    );
    return Promise.resolve({ orders });
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
      <Routes>
        <Route path="/werkstatt/auftraege" element={<OrdersPage />} />
        <Route path="/werkstatt/auftraege/:id" element={<div>Auftragsdetail</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  apiFetchMock.mockReset();
});

describe("workshop order list", () => {
  it("renders the page heading and the filter controls", async () => {
    installFilteringMock();
    renderPage();

    expect(
      screen.getByRole("heading", { level: 1, name: "Auftragsliste" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Status")).toBeInTheDocument();
    expect(screen.getByLabelText("Kennzeichen")).toBeInTheDocument();
  });

  it("renders every order with plate, vehicle, status and dates", async () => {
    installFilteringMock();
    renderPage();

    const table = await screen.findByRole("table");
    expect(within(table).getByText("B-GH 3456")).toBeInTheDocument();
    expect(within(table).getByText("Opel Corsa")).toBeInTheDocument();
    expect(within(table).getByText("B-EF 9012")).toBeInTheDocument();
    expect(within(table).getByText("Ford Focus")).toBeInTheDocument();
    expect(within(table).getByText("angefragt")).toBeInTheDocument();
    expect(within(table).getByText("16.10.2026")).toBeInTheDocument();
    expect(within(table).getByText("10.10.2026, 11:20")).toBeInTheDocument();
  });

  it("links each row to the order detail route", async () => {
    installFilteringMock();
    renderPage();

    const link = await screen.findByRole("link", { name: "A-2026-0045" });
    expect(link).toHaveAttribute("href", "/werkstatt/auftraege/45");
    expect(screen.getByRole("link", { name: "A-2026-0043" })).toHaveAttribute(
      "href",
      "/werkstatt/auftraege/43",
    );
  });

  it("filters the list when a status is selected", async () => {
    installFilteringMock();
    const user = userEvent.setup();
    renderPage();

    await screen.findByText("A-2026-0045");
    await user.selectOptions(screen.getByLabelText("Status"), "confirmed");

    await waitFor(() => {
      expect(apiFetchMock).toHaveBeenLastCalledWith(
        expect.stringContaining("status=confirmed"),
      );
    });
    expect(await screen.findByText("A-2026-0044")).toBeInTheDocument();
    expect(screen.queryByText("A-2026-0045")).not.toBeInTheDocument();
  });

  it("narrows the list when a plate fragment is typed", async () => {
    installFilteringMock();
    const user = userEvent.setup();
    renderPage();

    await screen.findByText("A-2026-0045");
    await user.type(screen.getByLabelText("Kennzeichen"), "EF");

    await waitFor(() => {
      expect(apiFetchMock).toHaveBeenLastCalledWith(
        expect.stringContaining("plate=EF"),
      );
    });
    expect(await screen.findByText("A-2026-0044")).toBeInTheDocument();
    expect(screen.queryByText("A-2026-0045")).not.toBeInTheDocument();
  });

  it("shows the empty state when no orders match", async () => {
    apiFetchMock.mockResolvedValue({ orders: [] });
    renderPage();

    expect(await screen.findByText("Noch keine Aufträge")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows the empty state with a reset action when filters exclude every order", async () => {
    installFilteringMock();
    const user = userEvent.setup();
    renderPage();

    await screen.findByText("A-2026-0045");
    await user.selectOptions(screen.getByLabelText("Status"), "completed");

    const emptyTitle = await screen.findByText("Keine Aufträge gefunden");
    const emptyState = emptyTitle.closest(".state-panel") as HTMLElement;
    const reset = within(emptyState).getByRole("button", {
      name: "Filter zurücksetzen",
    });
    await user.click(reset);
    expect(await screen.findByText("A-2026-0045")).toBeInTheDocument();
  });

  it("shows the error state and retries on demand", async () => {
    apiFetchMock.mockRejectedValueOnce(
      new ApiError(500, "internal_error", "Aufträge konnten nicht geladen werden."),
    );
    installFilteringMock();
    const user = userEvent.setup();
    renderPage();

    expect(
      await screen.findByText("Aufträge konnten nicht geladen werden."),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Erneut versuchen" }));
    expect(await screen.findByText("A-2026-0045")).toBeInTheDocument();
  });
});
