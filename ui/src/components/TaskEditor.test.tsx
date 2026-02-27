import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TaskEditor } from "./TaskEditor";

// Mock the hooks that TaskEditor uses.
const mockMutate = vi.fn();
vi.mock("~/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/client")>();
  return {
    ...actual,
    useSendControl: () => ({
      mutate: mockMutate,
      isPending: false,
    }),
    useConfig: () => ({
      data: {
        provider: "claude",
        model: "opus",
        autonomy: "supervised",
        task_types: ["code", "test", "review"],
      },
    }),
  };
});

beforeEach(() => {
  mockMutate.mockReset();
});

function renderTaskEditor(props: { onClose?: () => void; editItemId?: string } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onClose = props.onClose ?? vi.fn();
  return {
    onClose,
    ...render(
      <QueryClientProvider client={queryClient}>
        <TaskEditor onClose={onClose} editItemId={props.editItemId} />
      </QueryClientProvider>,
    ),
  };
}

describe("TaskEditor", () => {
  it("renders new task form", () => {
    renderTaskEditor();
    expect(screen.getByText("New Task")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("What should the agent do?")).toBeInTheDocument();
    expect(screen.getByText("enqueue")).toBeInTheDocument();
  });

  it("renders edit task form when editItemId is set", () => {
    renderTaskEditor({ editItemId: "order-1" });
    expect(screen.getByText("Edit Task")).toBeInTheDocument();
    expect(screen.getByText("save")).toBeInTheDocument();
  });

  it("sends enqueue command on submit", async () => {
    const user = userEvent.setup();
    renderTaskEditor();

    const textarea = screen.getByPlaceholderText("What should the agent do?");
    await user.type(textarea, "add error handling");

    await user.click(screen.getByText("enqueue"));

    expect(mockMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "enqueue",
        prompt: "add error handling",
        task_key: "code",
        provider: "claude",
        model: "opus",
      }),
      expect.any(Object),
    );
  });

  it("disables submit with empty prompt", () => {
    renderTaskEditor();
    const submitBtn = screen.getByText("enqueue");
    expect(submitBtn).toBeDisabled();
  });

  it("calls onClose when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderTaskEditor({ onClose });
    await user.click(screen.getByText("cancel"));
    expect(onClose).toHaveBeenCalled();
  });
});
