import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BacklogView } from "./BacklogView";
import { buildBacklogItem, buildSnapshot } from "~/test-utils";

let mockSnapshot = buildSnapshot();
const mockMutate = vi.fn();

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as Record<string, unknown>),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useSendControl: () => ({ mutate: mockMutate, isPending: false }),
  };
});

describe("BacklogView", () => {
  beforeEach(() => {
    mockMutate.mockReset();
    mockSnapshot = buildSnapshot();
  });

  it("lists items with their tags, estimate, and progress", () => {
    mockSnapshot = buildSnapshot({
      backlog: [
        buildBacklogItem({ id: "1", title: "Add search", tags: ["ui"], estimate: "small" }),
        buildBacklogItem({ id: "2", title: "Being built", status: "in_progress" }),
      ],
    });
    render(<BacklogView />);

    const list = screen.getByRole("list", { name: "Open items" });
    expect(within(list).getByText("Add search")).toBeInTheDocument();
    expect(within(list).getByText("#ui")).toBeInTheDocument();
    expect(within(list).getByText("~small")).toBeInTheDocument();
    expect(within(list).getByText("in progress")).toBeInTheDocument();
    expect(screen.getByText("2 open")).toBeInTheDocument();
  });

  it("adds an item with the trimmed title and clears the field", () => {
    render(<BacklogView />);
    const input = screen.getByLabelText("New backlog item");

    fireEvent.change(input, { target: { value: "  Write the docs  " } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(mockMutate).toHaveBeenCalledWith({ action: "backlog-add", prompt: "Write the docs" });
    expect(input).toHaveValue("");
  });

  it("does not submit on the Enter that confirms an IME composition", () => {
    render(<BacklogView />);
    const input = screen.getByLabelText("New backlog item");

    fireEvent.change(input, { target: { value: "検索" } });
    fireEvent.keyDown(input, { key: "Enter", isComposing: true });

    expect(mockMutate).not.toHaveBeenCalled();
  });

  it("edits a title inline and sends only a real change", () => {
    mockSnapshot = buildSnapshot({ backlog: [buildBacklogItem({ id: "4", title: "Old title" })] });
    render(<BacklogView />);

    fireEvent.click(screen.getByRole("button", { name: "Edit #4" }));
    const field = screen.getByLabelText("Title for #4");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(mockMutate).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Edit #4" }));
    fireEvent.change(screen.getByLabelText("Title for #4"), { target: { value: "New title" } });
    fireEvent.keyDown(screen.getByLabelText("Title for #4"), { key: "Enter" });
    expect(mockMutate).toHaveBeenCalledWith({
      action: "backlog-edit",
      target: "4",
      prompt: "New title",
    });
  });

  it("saves an edit once when the field also loses focus", () => {
    mockSnapshot = buildSnapshot({ backlog: [buildBacklogItem({ id: "4", title: "Old title" })] });
    render(<BacklogView />);

    fireEvent.click(screen.getByRole("button", { name: "Edit #4" }));
    const field = screen.getByLabelText("Title for #4");
    fireEvent.change(field, { target: { value: "New title" } });
    fireEvent.keyDown(field, { key: "Enter" });
    fireEvent.blur(field);

    expect(mockMutate).toHaveBeenCalledTimes(1);
  });

  it("saves an edit when the field loses focus", () => {
    mockSnapshot = buildSnapshot({ backlog: [buildBacklogItem({ id: "4", title: "Old title" })] });
    render(<BacklogView />);

    fireEvent.click(screen.getByRole("button", { name: "Edit #4" }));
    fireEvent.change(screen.getByLabelText("Title for #4"), { target: { value: "Clicked away" } });
    fireEvent.blur(screen.getByLabelText("Title for #4"));

    expect(mockMutate).toHaveBeenCalledWith({
      action: "backlog-edit",
      target: "4",
      prompt: "Clicked away",
    });
  });

  it("cancels an edit on Escape", () => {
    mockSnapshot = buildSnapshot({ backlog: [buildBacklogItem({ id: "4", title: "Keep me" })] });
    render(<BacklogView />);

    fireEvent.click(screen.getByRole("button", { name: "Edit #4" }));
    fireEvent.change(screen.getByLabelText("Title for #4"), { target: { value: "Discard me" } });
    fireEvent.keyDown(screen.getByLabelText("Title for #4"), { key: "Escape" });

    expect(mockMutate).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Edit #4" })).toHaveTextContent("Keep me");
  });

  it("marks an item done", () => {
    mockSnapshot = buildSnapshot({ backlog: [buildBacklogItem({ id: "7" })] });
    render(<BacklogView />);

    fireEvent.click(screen.getByRole("button", { name: "Mark #7 done" }));

    expect(mockMutate).toHaveBeenCalledWith({ action: "backlog-done", target: "7" });
  });

  it("shows a failed backlog change", () => {
    mockSnapshot = buildSnapshot({ backlog_error: 'adapter "backlog" is not configured' });
    render(<BacklogView />);

    expect(screen.getByRole("alert")).toHaveTextContent('adapter "backlog" is not configured');
  });

  it("labels sections only when the backlog has more than one", () => {
    mockSnapshot = buildSnapshot({
      backlog: [
        buildBacklogItem({ id: "1", section: "Inbox" }),
        buildBacklogItem({ id: "2", section: "Inbox" }),
      ],
    });
    const view = render(<BacklogView />);
    expect(screen.queryByText("Inbox")).not.toBeInTheDocument();

    mockSnapshot = buildSnapshot({
      backlog: [
        buildBacklogItem({ id: "1", section: "Inbox" }),
        buildBacklogItem({ id: "2", section: "Later" }),
      ],
    });
    view.rerender(<BacklogView />);
    expect(screen.getByText("Inbox")).toBeInTheDocument();
    expect(screen.getByText("Later")).toBeInTheDocument();
  });

  it("explains the empty backlog", () => {
    render(<BacklogView />);

    expect(screen.getByText(/The backlog is empty/)).toBeInTheDocument();
  });
});
