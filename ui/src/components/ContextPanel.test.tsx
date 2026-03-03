import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ContextPanel } from "./ContextPanel";
import { buildSession, buildSnapshot } from "../test-utils";

let mockSnapshot = buildSnapshot();
let mockActiveChannel: { type: "scheduler" } | { type: "agent"; sessionId: string } = {
  type: "scheduler",
};

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as Record<string, unknown>),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useActiveChannel: () => ({
      activeChannel: mockActiveChannel,
      setActiveChannel: vi.fn(),
    }),
    useReviewDiff: () => ({ data: undefined, isLoading: false, error: undefined }),
  };
});

describe("ContextPanel", () => {
  beforeEach(() => {
    const session = buildSession({ id: "actor-1", worktree_name: "wt-actor-1" });
    mockSnapshot = buildSnapshot({ sessions: [session] });
    mockActiveChannel = { type: "agent", sessionId: session.id };
  });

  it("renders the session worktree in the agent stats area", () => {
    render(<ContextPanel />);

    expect(screen.getByText("Worktree")).toBeInTheDocument();
    expect(screen.getByText("wt-actor-1")).toBeInTheDocument();
  });
});
