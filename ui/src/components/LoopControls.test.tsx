import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LoopControls } from "./LoopControls";
import { ControlContext } from "./ControlContext";

function renderWithControl(ui: React.ReactElement, send = vi.fn()) {
  return { send, ...render(<ControlContext.Provider value={send}>{ui}</ControlContext.Provider>) };
}

describe("LoopControls", () => {
  it("shows pause when running and sends pause on click", async () => {
    const user = userEvent.setup();
    const { send } = renderWithControl(<LoopControls loopState="running" />);
    const btn = screen.getByRole("button", { name: "pause" });
    expect(btn).toBeInTheDocument();

    await user.click(btn);
    expect(send).toHaveBeenCalledWith({ action: "pause" });
  });

  it("shows resume when paused and sends resume on click", async () => {
    const user = userEvent.setup();
    const { send } = renderWithControl(<LoopControls loopState="paused" />);
    const btn = screen.getByRole("button", { name: "resume" });
    expect(btn).toBeInTheDocument();

    await user.click(btn);
    expect(send).toHaveBeenCalledWith({ action: "resume" });
  });
});
