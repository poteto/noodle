import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ReviewActions } from "./ReviewActions";
import { ControlContext } from "./ControlContext";

function renderWithControl(ui: React.ReactElement, send = vi.fn()) {
  return { send, ...render(<ControlContext.Provider value={send}>{ui}</ControlContext.Provider>) };
}

describe("ReviewActions", () => {
  it("sends merge command on merge click", async () => {
    const user = userEvent.setup();
    const { send } = renderWithControl(<ReviewActions itemId="order-1" />);
    await user.click(screen.getByText("merge"));
    expect(send).toHaveBeenCalledWith({ action: "merge", order_id: "order-1" });
  });

  it("sends reject command on reject click", async () => {
    const user = userEvent.setup();
    const { send } = renderWithControl(<ReviewActions itemId="order-1" />);
    await user.click(screen.getByText("reject"));
    expect(send).toHaveBeenCalledWith({ action: "reject", order_id: "order-1" });
  });

  it("sends request-changes with feedback on two clicks", async () => {
    const user = userEvent.setup();
    const { send } = renderWithControl(<ReviewActions itemId="order-1" />);

    // First click shows the input
    await user.click(screen.getByText("changes"));
    const input = screen.getByPlaceholderText("What needs to change?");
    expect(input).toBeInTheDocument();

    // Type feedback and click send
    await user.type(input, "fix the tests");
    await user.click(screen.getByText("send"));

    expect(send).toHaveBeenCalledWith({
      action: "request-changes",
      order_id: "order-1",
      prompt: "fix the tests",
    });
  });

  it("calls onAction callback after sending", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    renderWithControl(<ReviewActions itemId="order-1" onAction={onAction} />);
    await user.click(screen.getByText("merge"));
    expect(onAction).toHaveBeenCalledWith("merge");
  });
});
