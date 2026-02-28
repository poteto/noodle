import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { AppLayout } from "./AppLayout";
import { buildSnapshot } from "../test-utils";

vi.mock("~/client", async () => {
  const actual = await vi.importActual<typeof import("~/client")>("~/client");
  return {
    ...actual,
    useSuspenseSnapshot: () => ({ data: buildSnapshot() }),
    useSSEStatus: () => "connected" as const,
    useSendControl: () => ({ mutate: vi.fn(), isPending: false }),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useRouter: () => ({ state: { location: { pathname: "/" } } }),
}));

describe("AppLayout", () => {
  it("renders three-column layout", () => {
    const { container } = render(<AppLayout />);
    const grid = container.querySelector(".grid") as HTMLElement;
    expect(grid).toBeTruthy();
    expect(grid.className).toContain("grid-cols-[260px_1fr_300px]");
  });

  it("renders sidebar, feed, and context panel", () => {
    render(<AppLayout />);
    expect(screen.getByText("NOODLE")).toBeDefined();
    const asides = document.querySelectorAll("aside");
    expect(asides.length).toBe(2);
    const mains = document.querySelectorAll("main");
    expect(mains.length).toBe(1);
  });
});
