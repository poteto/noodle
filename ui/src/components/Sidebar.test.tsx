import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Sidebar } from "./Sidebar";
import { buildSnapshot, buildOrder, buildStage } from "../test-utils";
import type { Snapshot, ChannelId } from "~/client";

const mockSetActiveChannel = vi.fn();
let mockActiveChannel: ChannelId = { type: "scheduler" };
let mockSnapshot: Snapshot = buildSnapshot();

vi.mock("~/client", async () => {
  const actual = await vi.importActual<typeof import("~/client")>("~/client");
  return {
    ...actual,
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useSSEStatus: () => "connected" as const,
    useActiveChannel: () => ({
      activeChannel: mockActiveChannel,
      setActiveChannel: mockSetActiveChannel,
    }),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useRouter: () => ({ state: { location: { pathname: "/" } } }),
}));

beforeEach(() => {
  mockActiveChannel = { type: "scheduler" };
  mockSnapshot = buildSnapshot();
  mockSetActiveChannel.mockClear();
});

describe("Sidebar", () => {
  it("renders nav links", () => {
    render(<Sidebar />);
    expect(screen.getByText("DASHBOARD")).toBeInTheDocument();
    expect(screen.getByText("LIVE FEED")).toBeInTheDocument();
    expect(screen.getByText("TREE")).toBeInTheDocument();
  });

  it("renders scheduler channel item", () => {
    render(<Sidebar />);
    expect(screen.getByText("Manager")).toBeInTheDocument();
  });

  it("renders orders from snapshot", () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "order-1",
          title: "Fix auth bug",
          status: "active",
          stages: [
            buildStage({ status: "completed", task_key: "test", session_id: "s1" }),
            buildStage({ status: "active", task_key: "execute", session_id: "s2" }),
          ],
        }),
      ],
    });
    render(<Sidebar />);
    expect(screen.getByText("Fix auth bug")).toBeInTheDocument();
  });

  it("selects scheduler channel when Manager clicked", async () => {
    mockActiveChannel = { type: "agent", sessionId: "s1" };
    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("Manager"));
    expect(mockSetActiveChannel).toHaveBeenCalledWith({ type: "scheduler" });
  });

  it("selects agent channel when stage with session_id clicked", async () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "order-1",
          title: "Test order",
          status: "active",
          stages: [
            buildStage({ status: "active", task_key: "execute", session_id: "s1" }),
          ],
        }),
      ],
    });
    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("execute"));
    expect(mockSetActiveChannel).toHaveBeenCalledWith({ type: "agent", sessionId: "s1" });
  });

  it("displays total cost in footer", () => {
    mockSnapshot = buildSnapshot({ total_cost_usd: 3.45 });
    render(<Sidebar />);
    expect(screen.getByText("$3.45")).toBeInTheDocument();
  });

  it("highlights active nav link", () => {
    render(<Sidebar />);
    const liveFeed = screen.getByText("LIVE FEED");
    expect(liveFeed.className).toContain("border-accent");
    expect(liveFeed.className).toContain("text-accent");
  });
});
