import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { OrderCard } from "./OrderCard";
import { buildOrder, buildStage } from "~/test-utils";

describe("OrderCard", () => {
  it("renders order title", () => {
    const order = buildOrder({ title: "Fix login bug" });
    render(<OrderCard order={order} />);
    expect(screen.getByText("Fix login bug")).toBeInTheDocument();
  });

  it("falls back to order id when no title", () => {
    const order = buildOrder({ id: "order-42", title: undefined });
    render(<OrderCard order={order} />);
    expect(screen.getByText("order-42")).toBeInTheDocument();
  });

  it("does not render stage pipeline for single-stage orders", () => {
    const order = buildOrder({
      stages: [buildStage({ task_key: "code", status: "pending" })],
    });
    const { container } = render(<OrderCard order={order} />);
    // Single stage should not show the pipeline section (which has arrows)
    expect(container.innerHTML).not.toContain("\u2192");
  });

  it("renders multi-stage pipeline with task keys", () => {
    const order = buildOrder({
      stages: [
        buildStage({ task_key: "plan", status: "completed" }),
        buildStage({ task_key: "code", status: "active" }),
        buildStage({ task_key: "test", status: "pending" }),
      ],
    });
    render(<OrderCard order={order} />);
    expect(screen.getByText("plan")).toBeInTheDocument();
    // "code" appears as both a Badge and a pipeline label
    expect(screen.getAllByText("code").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("test")).toBeInTheDocument();
  });

  it("renders on_failure stages with recovery label when failing", () => {
    const order = buildOrder({
      status: "failing",
      stages: [
        buildStage({ task_key: "code", status: "failed" }),
        buildStage({ task_key: "test", status: "pending" }),
      ],
      on_failure: [buildStage({ task_key: "rollback", status: "pending" })],
    });
    render(<OrderCard order={order} />);
    expect(screen.getByText("recovery")).toBeInTheDocument();
    expect(screen.getByText("rollback")).toBeInTheDocument();
  });

  it("shows active stage model badge", () => {
    const order = buildOrder({
      stages: [buildStage({ status: "active", model: "gpt-5" })],
    });
    render(<OrderCard order={order} />);
    expect(screen.getByText("gpt-5")).toBeInTheDocument();
  });
});
