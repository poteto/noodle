import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReviewList } from "./ReviewList";
import { buildReview, buildSnapshot } from "~/test-utils";

let mockSnapshot = buildSnapshot();
const mockMutate = vi.fn();

vi.mock("~/client", async () => {
  const actual = await vi.importActual("~/client");
  return {
    ...(actual as Record<string, unknown>),
    useSuspenseSnapshot: () => ({ data: mockSnapshot }),
    useSendControl: () => ({ mutate: mockMutate, isPending: false }),
    useReviewDiff: () => ({ data: { diff: "", stat: "" }, isLoading: false, error: undefined }),
  };
});

describe("ReviewList", () => {
  beforeEach(() => {
    mockMutate.mockReset();
    mockSnapshot = buildSnapshot();
  });

  it("keeps a valid selection when reviews shrink", () => {
    const first = buildReview({ order_id: "o-1", title: "First" });
    const second = buildReview({ order_id: "o-2", title: "Second" });
    mockSnapshot = buildSnapshot({
      pending_reviews: [first, second],
      pending_review_count: 2,
    });

    const view = render(<ReviewList />);
    fireEvent.click(screen.getByRole("tab", { name: "Second" }));

    mockSnapshot = buildSnapshot({
      pending_reviews: [first],
      pending_review_count: 1,
    });
    view.rerender(<ReviewList />);

    expect(screen.getByText("Review Actions")).toBeInTheDocument();
    expect(screen.getByText(/Quality Review:/)).toBeInTheDocument();
    expect(screen.getByText("o-1")).toBeInTheDocument();
  });
});
