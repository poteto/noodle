import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SchedulerFeed } from "./SchedulerFeed";
import { buildSnapshot, buildOrder } from "../test-utils";
import type { Snapshot } from "~/client";

const mockSend = vi.fn();
let mockSnapshot: Snapshot = buildSnapshot();

vi.mock("~/client", async () => {
  const actual = await vi.importActual<typeof import("~/client")>("~/client");
  return {
    ...actual,
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useSendControl: () => ({ mutate: mockSend, isPending: false }),
  };
});

beforeEach(() => {
  mockSnapshot = buildSnapshot();
  mockSend.mockClear();
});

describe("SchedulerFeed", () => {
  it("renders scheduler header", () => {
    render(<SchedulerFeed />);
    expect(screen.getByText("SCHEDULER")).toBeInTheDocument();
  });

  it("renders empty state when no orders", () => {
    render(<SchedulerFeed />);
    expect(screen.getByText("No orders yet. Send a prompt to start.")).toBeInTheDocument();
  });

  it("renders orders as summary items", () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({ id: "order-1", title: "Fix the tests" }),
        buildOrder({ id: "order-2", title: "Add login page" }),
      ],
    });
    render(<SchedulerFeed />);
    expect(screen.getByText("Fix the tests")).toBeInTheDocument();
    expect(screen.getByText("Add login page")).toBeInTheDocument();
  });

  it("renders input area", () => {
    render(<SchedulerFeed />);
    expect(screen.getByPlaceholderText("Send a prompt to the scheduler...")).toBeInTheDocument();
    expect(screen.getByText("SEND")).toBeInTheDocument();
  });

  it("sends steer command on submit", async () => {
    render(<SchedulerFeed />);
    const user = userEvent.setup();
    const textarea = screen.getByPlaceholderText("Send a prompt to the scheduler...");
    await user.type(textarea, "deploy to staging");
    await user.click(screen.getByText("SEND"));
    expect(mockSend).toHaveBeenCalledWith({
      action: "steer",
      name: "schedule",
      prompt: "deploy to staging",
    });
  });

  it("sends steer command on Enter", async () => {
    render(<SchedulerFeed />);
    const user = userEvent.setup();
    const textarea = screen.getByPlaceholderText("Send a prompt to the scheduler...");
    await user.type(textarea, "run tests{Enter}");
    expect(mockSend).toHaveBeenCalledWith({
      action: "steer",
      name: "schedule",
      prompt: "run tests",
    });
  });

  it("clears input after submit", async () => {
    render(<SchedulerFeed />);
    const user = userEvent.setup();
    const textarea = screen.getByPlaceholderText("Send a prompt to the scheduler...") as HTMLTextAreaElement;
    await user.type(textarea, "do something");
    await user.click(screen.getByText("SEND"));
    expect(textarea.value).toBe("");
  });

  it("does not send empty input", async () => {
    render(<SchedulerFeed />);
    const user = userEvent.setup();
    await user.click(screen.getByText("SEND"));
    expect(mockSend).not.toHaveBeenCalled();
  });
});
