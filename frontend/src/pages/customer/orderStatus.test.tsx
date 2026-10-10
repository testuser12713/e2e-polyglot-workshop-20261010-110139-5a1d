import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, apiFetch } from "../../api/client";
import type { PublicOrder } from "../../api/types";
import { OrderStatusPage } from "./OrderStatusPage";

// The shared api client is mocked: these tests exercise the page's own state
// handling against the contract's response shape, not the network layer.
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, apiFetch: vi.fn() };
});

const apiFetchMock = vi.mocked(apiFetch);

function makeOrder(overrides: Partial<PublicOrder> = {}): PublicOrder {
  return {
    orderNumber: "A-2026-0042",
    plate: "B-AB 1234",
    status: "completed",
    requestedDate: "2026-10-14",
    history: [
      { status: "requested", changedAt: "2026-10-10T11:20:00Z" },
      { status: "completed", changedAt: "2026-10-15T16:45:00Z" },
    ],
    invoice: null,
    ...overrides,
  };
}

async function submitLookup(orderNumber = "A-2026-0042", plate = "B-AB 1234") {
  const user = userEvent.setup();
  render(<OrderStatusPage />);
  await user.type(screen.getByLabelText("Auftragsnummer"), orderNumber);
  await user.type(screen.getByLabelText("Kennzeichen"), plate);
  await user.click(screen.getByRole("button", { name: "Status abrufen" }));
  return user;
}

beforeEach(() => {
  apiFetchMock.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("OrderStatusPage", () => {
  it("shows the empty state before any lookup", () => {
    render(<OrderStatusPage />);
    expect(screen.getByText("Noch keine Abfrage")).toBeInTheDocument();
  });

  it("renders the status and the history for a matching lookup", async () => {
    apiFetchMock.mockResolvedValueOnce(makeOrder());

    await submitLookup();

    expect(await screen.findByRole("heading", { name: "A-2026-0042" })).toBeInTheDocument();
    expect(screen.getAllByText("fertig").length).toBeGreaterThan(0);
    expect(screen.getByText("Statusverlauf")).toBeInTheDocument();
    // The most recent status change, converted from UTC to local Europe/Berlin time.
    expect(screen.getByText("15.10.2026, 18:45")).toBeInTheDocument();
    // The requested date is shown as TT.MM.JJJJ.
    expect(screen.getByText("14.10.2026")).toBeInTheDocument();

    const calledPath = apiFetchMock.mock.calls[0]?.[0] as string;
    expect(calledPath).toContain("/api/public/orders/A-2026-0042");
    expect(calledPath).toContain("plate=B-AB");
  });

  it("renders the error and no order data when the combination does not match", async () => {
    apiFetchMock.mockRejectedValueOnce(new ApiError(404, "not_found", "order not found"));

    await submitLookup("A-2026-9999", "B-ZZ 0000");

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Auftragsnummer und Kennzeichen passen nicht zusammen.");
    expect(alert).toHaveTextContent("not_found");
    expect(screen.queryByText("Statusverlauf")).not.toBeInTheDocument();
    expect(screen.queryByText("Rechnung")).not.toBeInTheDocument();
  });

  it("renders the invoice amounts when an invoice exists", async () => {
    apiFetchMock.mockResolvedValueOnce(
      makeOrder({
        status: "picked_up",
        invoice: {
          invoiceNumber: "R-2026-0098",
          netCents: 35500,
          taxRatePercent: 19,
          taxCents: 6745,
          grossCents: 42245,
          items: [
            {
              kind: "labor",
              description: "Fehlerdiagnose und Instandsetzung",
              hours: 2.5,
              amountCents: 23750,
              unitPriceCents: 9500,
            },
            {
              kind: "part",
              description: "Zündkerzen",
              quantity: 4,
              amountCents: 5000,
              unitPriceCents: 1250,
            },
          ],
        },
      }),
    );

    await submitLookup();

    expect(await screen.findByText("R-2026-0098")).toBeInTheDocument();
    expect(screen.getByText("Nettosumme")).toBeInTheDocument();
    expect(screen.getByText("355,00 €")).toBeInTheDocument();
    expect(screen.getByText("MwSt. (19 %)")).toBeInTheDocument();
    expect(screen.getByText("67,45 €")).toBeInTheDocument();
    expect(screen.getByText("Bruttosumme")).toBeInTheDocument();
    expect(screen.getByText("422,45 €")).toBeInTheDocument();
  });

  it("renders the pending note when no invoice exists yet", async () => {
    apiFetchMock.mockResolvedValueOnce(makeOrder({ invoice: null }));

    await submitLookup();

    expect(await screen.findByText("Rechnung noch nicht verfügbar.")).toBeInTheDocument();
  });

  it("does not call the api when a field is empty and shows an inline error", async () => {
    const user = userEvent.setup();
    render(<OrderStatusPage />);

    await user.type(screen.getByLabelText("Auftragsnummer"), "A-2026-0042");
    await user.click(screen.getByRole("button", { name: "Status abrufen" }));

    expect(apiFetchMock).not.toHaveBeenCalled();
    expect(screen.getByText("Bitte geben Sie das Kennzeichen ein.")).toBeInTheDocument();
  });
});
