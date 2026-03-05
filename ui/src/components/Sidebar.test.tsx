import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Sidebar } from "./Sidebar";
import { buildSnapshot, buildOrder, buildStage, buildSession } from "../test-utils";
import type { Snapshot, ChannelId } from "~/client";

const mockSetActiveChannel = vi.fn();
const mockNavigate = vi.fn();
let mockActiveChannel: ChannelId = { type: "scheduler" };
let mockSnapshot: Snapshot = buildSnapshot();
let mockPathname = "/";

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as object),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useWSStatus: () => "connected" as const,
    useActiveChannel: () => ({
      activeChannel: mockActiveChannel,
      setActiveChannel: mockSetActiveChannel,
    }),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useLocation: () => ({ pathname: mockPathname }),
  useNavigate: () => mockNavigate,
  Link: ({
    to,
    children,
    className,
  }: {
    to: string;
    children: React.ReactNode;
    className?: string;
  }) => (
    <a href={to} className={className}>
      {children}
    </a>
  ),
}));

beforeEach(() => {
  mockActiveChannel = { type: "scheduler" };
  mockSnapshot = buildSnapshot();
  mockPathname = "/";
  mockSetActiveChannel.mockClear();
  mockNavigate.mockClear();
});

describe("Sidebar", () => {
  it("renders nav links", () => {
    render(<Sidebar />);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Live Feed")).toBeInTheDocument();
    expect(screen.getByText("Topology")).toBeInTheDocument();
  });

  it("renders scheduler channel item", () => {
    render(<Sidebar />);
    expect(screen.getByText("Scheduler")).toBeInTheDocument();
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

  it("renders repeated stage signatures as distinct indexed labels", async () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "order-dup",
          title: "Duplicate signature order",
          status: "active",
          stages: [
            buildStage({ status: "completed", task_key: "execute" }),
            buildStage({ status: "completed", task_key: "execute" }),
            buildStage({ status: "pending", task_key: "execute" }),
            buildStage({ status: "pending", task_key: "execute" }),
          ],
        }),
      ],
    });
    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("Duplicate signature order"));
    expect(screen.getByText("execute 1")).toBeInTheDocument();
    expect(screen.getByText("execute 2")).toBeInTheDocument();
    expect(screen.getByText("execute 3")).toBeInTheDocument();
    expect(screen.getByText("execute 4")).toBeInTheDocument();
  });

  it("shows schedule bootstrap order in sidebar during bootstrap session", () => {
    mockSnapshot = buildSnapshot({
      sessions: [
        buildSession({
          id: "bootstrap-schedule-123",
          task_key: "schedule",
          status: "running",
        }),
      ],
      orders: [
        buildOrder({
          id: "schedule",
          title: "scheduling tasks based on your backlog",
          status: "active",
          stages: [buildStage({ status: "pending", task_key: "schedule" })],
        }),
      ],
    });

    render(<Sidebar />);
    expect(screen.getByText("Bootstrapping schedule skill")).toBeInTheDocument();
  });

  it("shows schedule bootstrap order in sidebar while bootstrap is pending", () => {
    mockSnapshot = buildSnapshot({
      loop_state: "running",
      sessions: [],
      orders: [
        buildOrder({
          id: "schedule",
          title: "scheduling tasks based on your backlog",
          status: "active",
          stages: [buildStage({ status: "pending", task_key: "schedule" })],
        }),
      ],
    });

    render(<Sidebar />);
    expect(screen.getByText("Bootstrapping schedule skill")).toBeInTheDocument();
  });

  it("navigates to / when Scheduler clicked", async () => {
    mockActiveChannel = { type: "agent", sessionId: "s1" };
    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("Scheduler"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/" });
  });

  it("navigates to actor route when stage with session_id clicked", async () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "order-1",
          title: "Test order",
          status: "active",
          stages: [buildStage({ status: "active", task_key: "execute", session_id: "s1" })],
        }),
      ],
    });
    render(<Sidebar />);
    const user = userEvent.setup();
    // Click to expand the order first
    await user.click(screen.getByText("Test order"));
    await user.click(screen.getByText("execute 1"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/actor/$id", params: { id: "s1" } });
  });

  it("navigates to bootstrap actor when clicking schedule stage during bootstrap", async () => {
    mockSnapshot = buildSnapshot({
      sessions: [
        buildSession({
          id: "bootstrap-schedule-123",
          task_key: "schedule",
          status: "running",
        }),
      ],
      orders: [
        buildOrder({
          id: "schedule",
          title: "scheduling tasks based on your backlog",
          status: "active",
          stages: [buildStage({ status: "pending", task_key: "schedule" })],
        }),
      ],
    });

    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("Bootstrapping schedule skill"));
    await user.click(screen.getByText("schedule 1"));
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/actor/$id",
      params: { id: "bootstrap-schedule-123" },
    });
  });

  it("displays total cost in footer", () => {
    mockSnapshot = buildSnapshot({ total_cost_usd: 3.45 });
    render(<Sidebar />);
    expect(screen.getByText("$3.45")).toBeInTheDocument();
  });

  it("highlights active nav link", () => {
    render(<Sidebar />);
    const liveFeed = screen.getByText("Live Feed");
    const link = liveFeed.closest("a");
    expect(link?.className).toContain("active");
  });

  it("navigates to actor route from non-feed page", async () => {
    mockPathname = "/dashboard";
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "order-1",
          title: "Test order",
          status: "active",
          stages: [buildStage({ status: "active", task_key: "execute", session_id: "s1" })],
        }),
      ],
    });
    render(<Sidebar />);
    const user = userEvent.setup();
    // Click to expand the order first
    await user.click(screen.getByText("Test order"));
    await user.click(screen.getByText("execute 1"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/actor/$id", params: { id: "s1" } });
  });

  it("navigates to / when Scheduler clicked from feed page", async () => {
    mockPathname = "/";
    render(<Sidebar />);
    const user = userEvent.setup();
    await user.click(screen.getByText("Scheduler"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/" });
  });

  it("renders logo mark", () => {
    const { container } = render(<Sidebar />);
    expect(container.querySelector(".logo-mark")).toBeTruthy();
  });

  it("renders agent avatar for scheduler", () => {
    const { container } = render(<Sidebar />);
    const avatar = container.querySelector(".agent-avatar");
    expect(avatar).toBeTruthy();
    expect(avatar?.textContent).toBe("S");
  });
});
