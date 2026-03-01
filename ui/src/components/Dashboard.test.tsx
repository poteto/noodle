import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { buildSnapshot, buildSession } from "~/test-utils";
import { Dashboard } from "./Dashboard";
import type { Snapshot } from "~/client";

let mockSnapshot: Snapshot = buildSnapshot();
const mockNavigate = vi.fn();

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as object),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
  createFileRoute: () => () => ({}),
}));

function setSnapshot(overrides: Partial<Snapshot>) {
  mockSnapshot = buildSnapshot(overrides);
}

describe("Dashboard", () => {
  it("clicking an active session row navigates to actor route", async () => {
    const user = userEvent.setup();
    setSnapshot({
      active: [buildSession({ id: "s1", status: "running", title: "Active Session" })],
      recent: [],
    });
    mockNavigate.mockReset();

    render(<Dashboard />);
    await user.click(screen.getByText("s1"));

    expect(mockNavigate).toHaveBeenCalledWith({ to: "/actor/$id", params: { id: "s1" } });
  });

  it("clicking a completed session row navigates to review route", async () => {
    const user = userEvent.setup();
    setSnapshot({
      active: [],
      recent: [buildSession({ id: "s2", status: "completed", title: "Completed Session" })],
    });
    mockNavigate.mockReset();

    render(<Dashboard />);
    await user.click(screen.getByText("s2"));

    expect(mockNavigate).toHaveBeenCalledWith({ to: "/review" });
  });

  it("clicking a pending-review session row navigates to review route", async () => {
    const user = userEvent.setup();
    setSnapshot({
      active: [buildSession({ id: "s3", status: "running", title: "Needs Review Session" })],
      recent: [],
      pending_reviews: [
        {
          order_id: "order-1",
          stage_index: 0,
          worktree_name: "wt-order-1",
          worktree_path: "/tmp/wt-order-1",
          session_id: "s3",
        },
      ],
      pending_review_count: 1,
    });
    mockNavigate.mockReset();

    render(<Dashboard />);
    await user.click(screen.getByText("s3"));

    expect(mockNavigate).toHaveBeenCalledWith({ to: "/review" });
  });

  it("renders stats bar with correct totals", () => {
    const active = [
      buildSession({ id: "s1", status: "running" }),
      buildSession({ id: "s2", status: "running" }),
    ];
    const recent = [buildSession({ id: "s3", status: "merged" })];
    setSnapshot({ active, recent, total_cost_usd: 5.67 });

    render(<Dashboard />);

    const statsBar = screen.getByTestId("stats-bar");
    expect(within(statsBar).getByText("3")).toBeInTheDocument(); // total
    expect(within(statsBar).getByText("2")).toBeInTheDocument(); // active
    expect(within(statsBar).getByText("1")).toBeInTheDocument(); // completed
    expect(within(statsBar).getByText("$5.67")).toBeInTheDocument(); // cost
  });

  it("renders session grid rows from snapshot data", () => {
    const active = [
      buildSession({
        id: "sess-1",
        title: "Fix auth",
        status: "running",
        model: "opus-4",
        remote_host: "remote-1",
        duration_seconds: 90,
        total_cost_usd: 1.5,
      }),
    ];
    const recent = [
      buildSession({
        id: "sess-2",
        display_name: "agent-bravo",
        status: "merged",
        model: "sonnet",
        duration_seconds: 300,
        total_cost_usd: 3,
      }),
    ];
    setSnapshot({ active, recent });

    render(<Dashboard />);

    expect(screen.getByText("sess-1")).toBeInTheDocument();
    expect(screen.getByText("Fix auth")).toBeInTheDocument();
    expect(screen.getByText("opus-4")).toBeInTheDocument();
    expect(screen.getByText("remote-1")).toBeInTheDocument();
    expect(screen.getByText("1m 30s")).toBeInTheDocument();
    expect(screen.getByText("$1.50")).toBeInTheDocument();

    expect(screen.getByText("sess-2")).toBeInTheDocument();
    expect(screen.getByText("agent-bravo")).toBeInTheDocument();
    expect(screen.getByText("sonnet")).toBeInTheDocument();
    expect(screen.getByText("5m")).toBeInTheDocument();
    expect(screen.getByText("$3.00")).toBeInTheDocument();
  });

  it("status badges show correct variant colors", () => {
    const active = [buildSession({ id: "s1", status: "running" })];
    const recent = [
      buildSession({ id: "s2", status: "merged" }),
      buildSession({ id: "s3", status: "failed" }),
    ];
    setSnapshot({ active, recent });

    render(<Dashboard />);

    const badges = screen.getAllByTestId("status-badge");
    const runningBadge = badges.find((b) => b.textContent === "running");
    const mergedBadge = badges.find((b) => b.textContent === "merged");
    const failedBadge = badges.find((b) => b.textContent === "failed");

    expect(runningBadge?.className).toContain("bg-accent");
    expect(mergedBadge?.className).toContain("bg-green");
    expect(failedBadge?.className).toContain("bg-red");
  });

  it("sort by column works", async () => {
    const user = userEvent.setup();
    const active = [
      buildSession({ id: "a-first", title: "Alpha", status: "running" }),
      buildSession({ id: "b-second", title: "Bravo", status: "running" }),
    ];
    setSnapshot({ active, recent: [] });

    render(<Dashboard />);

    // Default sort by ID ascending — a-first should be before b-second
    const rows = screen.getAllByRole("row");
    // rows[0] is header, rows[1] and rows[2] are data
    expect(rows[1]).toHaveTextContent("a-first");
    expect(rows[2]).toHaveTextContent("b-second");

    // Click ID header to flip to descending
    const idHeader = screen.getByText("ID");
    await user.click(idHeader);

    const rowsAfter = screen.getAllByRole("row");
    expect(rowsAfter[1]).toHaveTextContent("b-second");
    expect(rowsAfter[2]).toHaveTextContent("a-first");
  });

  it("shows empty state when no sessions", () => {
    setSnapshot({ active: [], recent: [] });

    render(<Dashboard />);

    expect(screen.getByText("No sessions")).toBeInTheDocument();
  });

  it("uses display_name as title fallback", () => {
    const active = [buildSession({ id: "s1", display_name: "my-agent", title: undefined })];
    setSnapshot({ active, recent: [] });

    render(<Dashboard />);

    expect(screen.getByText("my-agent")).toBeInTheDocument();
  });

  it("shows local as host fallback", () => {
    const active = [buildSession({ id: "s1", remote_host: undefined })];
    setSnapshot({ active, recent: [] });

    render(<Dashboard />);

    expect(screen.getByText("local")).toBeInTheDocument();
  });
});
