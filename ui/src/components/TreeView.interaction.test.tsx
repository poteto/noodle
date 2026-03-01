import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { TreeView } from "./TreeView";
import { buildSnapshot, buildOrder, buildStage } from "~/test-utils";
import type { Snapshot } from "~/client";

const mockNavigate = vi.fn();
let mockSnapshot: Snapshot = buildSnapshot();

vi.mock("~/client", async () => {
  const actual = await vi.importActual<typeof import("~/client")>("~/client");
  return {
    ...actual,
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
  };
});

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-router")>(
    "@tanstack/react-router",
  );
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

beforeEach(() => {
  mockNavigate.mockReset();
  mockSnapshot = buildSnapshot();
  Object.defineProperty(SVGSVGElement.prototype, "width", {
    configurable: true,
    value: { baseVal: { value: 1200 } },
  });
  Object.defineProperty(SVGSVGElement.prototype, "height", {
    configurable: true,
    value: { baseVal: { value: 800 } },
  });
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

    fireEvent.click(clickableNodes[0]!);

    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/actor/$id",
      params: { id: "actor-1" },
    });
  });
});
