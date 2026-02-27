import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AgentCard } from "./AgentCard";
import { ControlContext } from "./ControlContext";
import { buildSession } from "~/test-utils";

function renderWithControl(ui: React.ReactElement, send = vi.fn()) {
  return { send, ...render(<ControlContext.Provider value={send}>{ui}</ControlContext.Provider>) };
}

describe("AgentCard", () => {
  it("renders session display name and model", () => {
    const session = buildSession({
      display_name: "agent-alpha",
      model: "opus-4",
    });
    renderWithControl(<AgentCard session={session} />);
    expect(screen.getByText("agent-alpha")).toBeInTheDocument();
    expect(screen.getByText("opus-4")).toBeInTheDocument();
  });

  it("renders title instead of display_name when present", () => {
    const session = buildSession({
      display_name: "agent-1",
      title: "Fix authentication",
    });
    renderWithControl(<AgentCard session={session} />);
    expect(screen.getByText("Fix authentication")).toBeInTheDocument();
    // display_name shown as subtitle
    expect(screen.getByText("agent-1")).toBeInTheDocument();
  });

  it("displays cost and duration", () => {
    const session = buildSession({
      total_cost_usd: 1.5,
      duration_seconds: 90,
    });
    renderWithControl(<AgentCard session={session} />);
    expect(screen.getByText("$1.50")).toBeInTheDocument();
    expect(screen.getByText("1m 30s")).toBeInTheDocument();
  });

  it("renders context window usage percentage", () => {
    const session = buildSession({ context_window_usage_pct: 75 });
    renderWithControl(<AgentCard session={session} />);
    expect(screen.getByText("ctx 75%")).toBeInTheDocument();
  });

  it("renders current action tool type as badge", () => {
    const session = buildSession({ current_action: "Edit src/main.ts" });
    renderWithControl(<AgentCard session={session} />);
    // "Edit" appears as a tool badge and also within the action text split
    const badges = screen.getAllByText("Edit");
    expect(badges.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("src/main.ts")).toBeInTheDocument();
  });

  it("shows retry count when > 0", () => {
    const session = buildSession({ retry_count: 2 });
    renderWithControl(<AgentCard session={session} />);
    expect(screen.getByText("retry 2")).toBeInTheDocument();
  });

  it("sends stop command on stop button click", async () => {
    const user = userEvent.setup();
    const session = buildSession({ id: "s-42" });
    const { send, container } = renderWithControl(<AgentCard session={session} />);

    // The stop button is the small inner button (not the card button).
    // It contains the Square SVG icon. Find it by querying the inner button.
    const buttons = container.querySelectorAll("button");
    // The outer button is the card, the inner one is the stop button
    const stopBtn = Array.from(buttons).find((b) =>
      b.classList.contains("flex") && b.querySelector("svg"),
    );
    expect(stopBtn).toBeTruthy();
    await user.click(stopBtn!);
    expect(send).toHaveBeenCalledWith({ action: "stop", name: "s-42" });
  });

  it("does not propagate stop click to card onClick", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    const session = buildSession({ id: "s-42" });
    const { container } = renderWithControl(<AgentCard session={session} onClick={onClick} />);

    const buttons = container.querySelectorAll("button");
    const stopBtn = Array.from(buttons).find((b) =>
      b.classList.contains("flex") && b.querySelector("svg"),
    );
    expect(stopBtn).toBeTruthy();
    await user.click(stopBtn!);
    expect(onClick).not.toHaveBeenCalled();
  });
});
