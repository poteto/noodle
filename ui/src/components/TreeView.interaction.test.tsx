import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { TreeView } from "./TreeView";
import { buildSnapshot, buildOrder, buildStage } from "~/test-utils";
import type { Snapshot } from "~/client";

const mockNavigate = vi.fn();
let mockSnapshot: Snapshot = buildSnapshot();

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as object),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
  };
});

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual("@tanstack/react-router");
  return {
    ...(actual as object),
    useNavigate: () => mockNavigate,
  };
});

beforeEach(() => {
  mockNavigate.mockReset();
  mockSnapshot = buildSnapshot();
});

describe("TreeView actor navigation", () => {
  it("makes actor nodes clickable and navigates to actor route", async () => {
    mockSnapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "o1",
          title: "Fix auth",
          status: "active",
          stages: [
            buildStage({ task_key: "execute", status: "active", session_id: "actor-1" }),
            buildStage({ task_key: "quality", status: "pending" }),
          ],
        }),
      ],
    });

    const { container } = render(<TreeView />);

    await waitFor(() => {
      expect(container.querySelectorAll(".node").length).toBeGreaterThan(0);
    });

    const clickableNodes = container.querySelectorAll(".node.node-clickable");
    expect(clickableNodes).toHaveLength(1);

    const firstClickable = clickableNodes.item(0);
    expect(firstClickable).not.toBeNull();
    if (!firstClickable) {
      throw new Error("expected clickable actor node");
    }
    fireEvent.click(firstClickable);

    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/actor/$id",
      params: { id: "actor-1" },
    });
  });
});
