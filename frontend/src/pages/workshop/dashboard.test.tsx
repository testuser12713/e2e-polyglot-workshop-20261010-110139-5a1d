import "@testing-library/jest-dom/vitest";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockApiFetch } = vi.hoisted(() => ({ mockApiFetch: vi.fn() }));

vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, apiFetch: mockApiFetch };
});

import { ApiError } from "../../api/client";
import { DashboardPage } from "./DashboardPage";

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );
}

describe("DashboardPage", () => {
  beforeEach(() => {
    mockApiFetch.mockReset();
  });

  it("reads the dashboard endpoint through the shared api client", async () => {
    mockApiFetch.mockResolvedValue({
      openCounts: { requested: 3, confirmed: 2, in_progress: 4 },
      completedToday: 1,
      monthRevenueGrossCents: 841255,
    });

    renderPage();

    await screen.findByTestId("kpi-requested");
    expect(mockApiFetch).toHaveBeenCalledWith("/api/workshop/dashboard");
  });

  it("renders the open-order counts and the orders completed today", async () => {
    mockApiFetch.mockResolvedValue({
      openCounts: { requested: 3, confirmed: 2, in_progress: 4 },
      completedToday: 7,
      monthRevenueGrossCents: 0,
    });

    renderPage();

    const requested = await screen.findByTestId("kpi-requested");
    expect(requested).toHaveTextContent("angefragt");
    expect(within(requested).getByText("3")).toBeInTheDocument();

    const confirmed = screen.getByTestId("kpi-confirmed");
    expect(confirmed).toHaveTextContent("bestätigt");
    expect(within(confirmed).getByText("2")).toBeInTheDocument();

    const inProgress = screen.getByTestId("kpi-in-progress");
    expect(inProgress).toHaveTextContent("in Arbeit");
    expect(within(inProgress).getByText("4")).toBeInTheDocument();

    expect(within(screen.getByTestId("kpi-done")).getByText("7")).toBeInTheDocument();
  });

  it("formats the month revenue from whole cents as German currency", async () => {
    mockApiFetch.mockResolvedValue({
      openCounts: { requested: 0, confirmed: 0, in_progress: 0 },
      completedToday: 0,
      monthRevenueGrossCents: 123456,
    });

    renderPage();

    const revenue = await screen.findByTestId("kpi-revenue");
    expect(within(revenue).getByText("1.234,56 €")).toBeInTheDocument();
  });

  it("shows a legitimate zero as 0,00 € and not as an error", async () => {
    mockApiFetch.mockResolvedValue({
      openCounts: { requested: 0, confirmed: 0, in_progress: 0 },
      completedToday: 0,
      monthRevenueGrossCents: 0,
    });

    renderPage();

    const revenue = await screen.findByTestId("kpi-revenue");
    expect(within(revenue).getByText("0,00 €")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("renders the two list sections below the KPI cards", async () => {
    mockApiFetch.mockResolvedValue({
      openCounts: { requested: 0, confirmed: 0, in_progress: 0 },
      completedToday: 0,
      monthRevenueGrossCents: 0,
    });

    renderPage();

    await screen.findByTestId("kpi-requested");
    const lists = screen.getByRole("region", { name: "Listen" });
    expect(
      within(lists).getByRole("heading", { name: "Nächste Wunschtermine" }),
    ).toBeInTheDocument();
    expect(
      within(lists).getByRole("heading", { name: "Heute fertig geworden" }),
    ).toBeInTheDocument();
  });

  it("shows the error state instead of fake zeros when the request fails", async () => {
    mockApiFetch.mockRejectedValue(new ApiError(401, "unauthorized", "Nicht angemeldet."));

    renderPage();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Nicht angemeldet.");
    expect(screen.queryByTestId("kpi-revenue")).not.toBeInTheDocument();
    expect(screen.queryByText("0,00 €")).not.toBeInTheDocument();
  });

  it("shows the error state when the response shape is incomplete", async () => {
    mockApiFetch.mockResolvedValue({ openCounts: { requested: 1 } });

    renderPage();

    await screen.findByRole("alert");
    expect(screen.queryByTestId("kpi-requested")).not.toBeInTheDocument();
  });
});
