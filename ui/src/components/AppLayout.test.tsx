import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { AppLayout } from "./AppLayout";

describe("AppLayout", () => {
  it("renders three-column layout", () => {
    const { container } = render(<AppLayout />);
    const grid = container.firstElementChild as HTMLElement;
    expect(grid.className).toContain("grid");
    expect(grid.className).toContain("grid-cols-[260px_1fr_300px]");
  });

  it("renders sidebar, feed, and context panel", () => {
    render(<AppLayout />);
    // Sidebar has the NOODLE heading
    expect(screen.getByText("NOODLE")).toBeDefined();
    // All three panels render as semantic elements
    const asides = document.querySelectorAll("aside");
    expect(asides.length).toBe(2);
    const mains = document.querySelectorAll("main");
    expect(mains.length).toBe(1);
  });
});
