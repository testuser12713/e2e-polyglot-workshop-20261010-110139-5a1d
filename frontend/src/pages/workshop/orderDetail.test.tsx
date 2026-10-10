import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { apiFetch, ApiError } from "../../api/client";
import type { Order } from "../../api/types";
import { OrderDetailPage } from "./OrderDetailPage";

// The shared api client is mocked: the page under test is the screen, not the
// fetch wrapper. ApiError stays the real class so a rejected step can carry the
// contract's status code.
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, apiFetch: vi.fn() };
});

const apiFetchMock = apiFetch as unknown as Mock;

function makeOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: 1,
    orderNumber: "AU-2026-0001",
    vehicleId: 1,
    customerId: 1,
    plate: "B-AB 1234",
    brand: "VW",
    model: "Golf",
    status: "requested",
    requestedDate: "2026-10-16",
    problemDescription: "Bremsen quietschen beim Anhalten.",
    createdAt: "2026-10-10T09:00:00Z",
    items: [],
    history: [{ status: "requested", changedAt: "2026-10-10T09:00:00Z", employeeName: "Meister" }],
    ...overrides,
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/auftraege/1"]}>
      <Routes>
        <Route path="/werkstatt/auftraege/:id" element={<OrderDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

function headerBadge(container: HTMLElement): HTMLElement {
  const badge = container.querySelector(".od-detail__head .status-badge");
  if (!badge) throw new Error("header status badge not found");
  return badge as HTMLElement;
}

beforeEach(() => {
  apiFetchMock.mockReset();
});

describe("OrderDetailPage", () => {
  it("confirms an order and shows the new status", async () => {
    const user = userEvent.setup();
    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === "/api/workshop/orders/1" && (!init || !init.method)) {
        return makeOrder({ status: "requested" });
      }
      if (path === "/api/workshop/orders/1/status" && init?.method === "POST") {
        return makeOrder({
          status: "confirmed",
          history: [
            { status: "requested", changedAt: "2026-10-10T09:00:00Z", employeeName: "Meister" },
            { status: "confirmed", changedAt: "2026-10-11T10:00:00Z", employeeName: "Meister" },
          ],
        });
      }
      throw new Error(`unexpected request ${path}`);
    });

    const { container } = renderPage();

    await waitFor(() => {
      expect(headerBadge(container)).toHaveTextContent("angefragt");
    });
    const confirmButton = await screen.findByRole("button", { name: "Auftrag bestätigen" });
    await user.click(confirmButton);

    await waitFor(() => {
      expect(headerBadge(container)).toHaveTextContent("bestätigt");
    });
    expect(headerBadge(container)).toHaveAttribute("data-status", "confirmed");
  });

  it("adds a labor position and shows it in the list", async () => {
    const user = userEvent.setup();
    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === "/api/workshop/orders/1" && (!init || !init.method)) {
        return makeOrder({ status: "confirmed" });
      }
      if (path === "/api/workshop/orders/1/items" && init?.method === "POST") {
        return makeOrder({
          status: "confirmed",
          items: [
            {
              id: 7,
              kind: "labor",
              description: "Arbeitszeit: Werkstattleistung",
              hours: 1.5,
              quantity: null,
              unitPriceCents: null,
              amountCents: 0,
            },
          ],
        });
      }
      throw new Error(`unexpected request ${path}`);
    });

    renderPage();

    expect(await screen.findByText("Noch keine Positionen erfasst.")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Arbeitszeit"), "1.5");
    await user.click(screen.getByRole("button", { name: "Arbeitszeit hinzufügen" }));

    expect(await screen.findByText("Arbeitszeit: Werkstattleistung")).toBeInTheDocument();
    expect(screen.queryByText("Noch keine Positionen erfasst.")).not.toBeInTheDocument();
  });

  it("offers only the allowed transition from the current status", async () => {
    apiFetchMock.mockResolvedValue(makeOrder({ status: "requested" }));
    const { unmount } = renderPage();

    expect(await screen.findByRole("button", { name: "Auftrag bestätigen" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "In Arbeit setzen" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Fertig melden" })).not.toBeInTheDocument();

    unmount();
    apiFetchMock.mockReset();
    apiFetchMock.mockResolvedValue(makeOrder({ status: "picked_up" }));
    renderPage();

    await screen.findByText("Auftrag abgeschlossen");
    expect(screen.queryByRole("button", { name: "Auftrag bestätigen" })).not.toBeInTheDocument();
  });

  it("shows a 409 as an error and keeps the unchanged status", async () => {
    const user = userEvent.setup();
    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === "/api/workshop/orders/1" && (!init || !init.method)) {
        return makeOrder({ status: "requested" });
      }
      if (path === "/api/workshop/orders/1/status" && init?.method === "POST") {
        throw new ApiError(409, "invalid_transition", "order status transition is not allowed");
      }
      throw new Error(`unexpected request ${path}`);
    });

    const { container } = renderPage();

    const confirmButton = await screen.findByRole("button", { name: "Auftrag bestätigen" });
    await user.click(confirmButton);

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("order status transition is not allowed");
    expect(alert).toHaveTextContent("invalid_transition");
    await waitFor(() => {
      expect(headerBadge(container)).toHaveTextContent("angefragt");
    });
  });
});
