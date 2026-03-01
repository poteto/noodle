import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { AppLayout } from "./AppLayout";
import { buildSnapshot } from "../test-utils";

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as Record<string, unknown>),
    useSuspenseSnapshot: () => ({ data: buildSnapshot() }),
    useWSStatus: () => "connected" as const,
    useSendControl: () => ({ mutate: vi.fn(), isPending: false }),
    useActiveChannel: () => ({
      activeChannel: { type: "scheduler" },
      setActiveChannel: vi.fn(),
    }),
    useSessionEvents: () => ({ data: [] }),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useLocation: () => ({ pathname: "/" }),
  useNavigate: () => vi.fn(),
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

describe("AppLayout", () => {
  it("renders feed and context panel", () => {
    const { container } = render(<AppLayout />);
    const asides = container.querySelectorAll("aside");
    expect(asides.length).toBe(1);
    const mains = container.querySelectorAll("main");
    expect(mains.length).toBe(1);
  });

  it("renders context panel with context-panel class", () => {
    const { container } = render(<AppLayout />);
    const panel = container.querySelector(".context-panel");
    expect(panel).toBeTruthy();
  });
});
