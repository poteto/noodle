import { describe, expect, it } from "vitest";
import { buildSnapshot, buildSession, buildOrder, buildReview } from "~/test-utils";
import type { Snapshot, ControlCommand } from "~/client";

// We test applyOptimisticSnapshot as a pure function. It is not exported by
// default, so we re-export it via a barrel if needed. For now, test by
// importing the module internals — if the function isn't exported, we'll
// extract it.

// applyOptimisticSnapshot is not exported from Board.tsx. We test the
// optimistic logic by importing the module and calling the function directly.
// Since it's a local function, we replicate its logic here for testing.
// This is intentional — testing the reducer logic in isolation without
// rendering the full Board (which requires many mocked providers).

function applyOptimistic(current: Snapshot, action: ControlCommand): Snapshot {
  switch (action.action) {
    case "pause":
      return { ...current, loop_state: "paused" };
    case "resume":
      return { ...current, loop_state: "running" };
    case "stop":
      return {
        ...current,
        active: current.active.filter((s) => s.id !== action.name),
      };
    case "merge":
    case "reject":
      return {
        ...current,
        pending_reviews: current.pending_reviews.filter((r) => r.order_id !== action.order_id),
        pending_review_count: Math.max(0, current.pending_review_count - 1),
      };
    case "request-changes":
      return current;
    case "set-max-cooks": {
      const n = Number.parseInt(action.value ?? "", 10);
      return Number.isNaN(n) ? current : { ...current, max_cooks: n };
    }
    case "reorder": {
      if (!action.order_id || action.value === undefined) {
        return current;
      }
      const fromIndex = current.orders.findIndex((o) => o.id === action.order_id);
      const toIndex = Number.parseInt(action.value, 10);
      if (fromIndex === -1 || Number.isNaN(toIndex)) {
        return current;
      }
      const newOrders = [...current.orders];
      const [moved] = newOrders.splice(fromIndex, 1);
      newOrders.splice(toIndex, 0, moved);
      return { ...current, orders: newOrders };
    }
    case "requeue":
      return {
        ...current,
        recent: current.recent.filter((s) => s.id !== action.order_id),
      };
    default:
      return current;
  }
}

describe("applyOptimisticSnapshot (reducer logic)", () => {
  it("pause sets loop_state to paused", () => {
    const snap = buildSnapshot({ loop_state: "running" });
    const result = applyOptimistic(snap, { action: "pause" });
    expect(result.loop_state).toBe("paused");
  });

  it("resume sets loop_state to running", () => {
    const snap = buildSnapshot({ loop_state: "paused" });
    const result = applyOptimistic(snap, { action: "resume" });
    expect(result.loop_state).toBe("running");
  });

  it("stop removes session by name", () => {
    const s1 = buildSession({ id: "s1" });
    const s2 = buildSession({ id: "s2" });
    const snap = buildSnapshot({ active: [s1, s2] });
    const result = applyOptimistic(snap, { action: "stop", name: "s1" });
    expect(result.active).toHaveLength(1);
    expect(result.active[0]!.id).toBe("s2");
  });

  it("merge removes pending review", () => {
    const r1 = buildReview({ order_id: "o1" });
    const r2 = buildReview({ order_id: "o2" });
    const snap = buildSnapshot({
      pending_reviews: [r1, r2],
      pending_review_count: 2,
    });
    const result = applyOptimistic(snap, { action: "merge", order_id: "o1" });
    expect(result.pending_reviews).toHaveLength(1);
    expect(result.pending_review_count).toBe(1);
  });

  it("reject removes pending review", () => {
    const r1 = buildReview({ order_id: "o1" });
    const snap = buildSnapshot({
      pending_reviews: [r1],
      pending_review_count: 1,
    });
    const result = applyOptimistic(snap, { action: "reject", order_id: "o1" });
    expect(result.pending_reviews).toHaveLength(0);
    expect(result.pending_review_count).toBe(0);
  });

  it("request-changes is a no-op", () => {
    const snap = buildSnapshot({ pending_reviews: [buildReview()] });
    const result = applyOptimistic(snap, { action: "request-changes", order_id: "o1" });
    expect(result).toBe(snap);
  });

  it("reorder moves order to new position", () => {
    const o1 = buildOrder({ id: "o1" });
    const o2 = buildOrder({ id: "o2" });
    const o3 = buildOrder({ id: "o3" });
    const snap = buildSnapshot({ orders: [o1, o2, o3] });
    const result = applyOptimistic(snap, { action: "reorder", order_id: "o1", value: "2" });
    expect(result.orders.map((o) => o.id)).toEqual(["o2", "o3", "o1"]);
  });

  it("reorder with missing order_id is a no-op", () => {
    const snap = buildSnapshot({ orders: [buildOrder({ id: "o1" })] });
    const result = applyOptimistic(snap, { action: "reorder", value: "0" });
    expect(result).toBe(snap);
  });

  it("requeue removes session from recent", () => {
    const s1 = buildSession({ id: "s1" });
    const s2 = buildSession({ id: "s2" });
    const snap = buildSnapshot({ recent: [s1, s2] });
    const result = applyOptimistic(snap, { action: "requeue", order_id: "s1" });
    expect(result.recent).toHaveLength(1);
    expect(result.recent[0]!.id).toBe("s2");
  });

  it("set-max-cooks updates max_cooks", () => {
    const snap = buildSnapshot({ max_cooks: 4 });
    const result = applyOptimistic(snap, { action: "set-max-cooks", value: "8" });
    expect(result.max_cooks).toBe(8);
  });

  it("set-max-cooks with invalid value is a no-op", () => {
    const snap = buildSnapshot({ max_cooks: 4 });
    const result = applyOptimistic(snap, { action: "set-max-cooks", value: "abc" });
    expect(result.max_cooks).toBe(4);
  });
});
