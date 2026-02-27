import { describe, expect, it } from "vitest";
import { deriveKanbanColumns } from "./types";
import type { Snapshot, Order, Session, PendingReviewItem } from "./types";

function emptySnapshot(overrides?: Partial<Snapshot>): Snapshot {
  return {
    updated_at: new Date().toISOString(),
    loop_state: "running",
    sessions: [],
    active: [],
    recent: [],
    orders: [],
    active_order_ids: [],
    action_needed: [],
    events_by_session: {},
    feed_events: [],
    total_cost_usd: 0,
    pending_reviews: [],
    pending_review_count: 0,
    autonomy: "supervised",
    max_cooks: 4,
    warnings: [],
    ...overrides,
  };
}

function makeOrder(id: string, overrides?: Partial<Order>): Order {
  return {
    id,
    stages: [{ status: "pending", provider: "claude", model: "opus" }],
    status: "active",
    ...overrides,
  };
}

function makeSession(id: string, overrides?: Partial<Session>): Session {
  return {
    id,
    display_name: id,
    status: "running",
    runtime: "local",
    provider: "claude",
    model: "opus",
    total_cost_usd: 0,
    duration_seconds: 0,
    last_activity: new Date().toISOString(),
    current_action: "",
    health: "green",
    context_window_usage_pct: 0,
    retry_count: 0,
    idle_seconds: 0,
    stuck_threshold_seconds: 300,
    loop_state: "running",
    ...overrides,
  };
}

function makeReview(orderId: string): PendingReviewItem {
  return {
    order_id: orderId,
    stage_index: 0,
    worktree_name: "wt-" + orderId,
    worktree_path: "/tmp/wt-" + orderId,
  };
}

describe("deriveKanbanColumns", () => {
  it("returns empty columns for empty snapshot", () => {
    const cols = deriveKanbanColumns(emptySnapshot());
    expect(cols.queued).toEqual([]);
    expect(cols.cooking).toEqual([]);
    expect(cols.review).toEqual([]);
    expect(cols.done).toEqual([]);
  });

  it("splits orders between queued and active", () => {
    const o1 = makeOrder("o1");
    const o2 = makeOrder("o2");
    const o3 = makeOrder("o3");
    const s1 = makeSession("s1");

    const snap = emptySnapshot({
      orders: [o1, o2, o3],
      active_order_ids: ["o2"],
      active: [s1],
    });

    const cols = deriveKanbanColumns(snap);
    expect(cols.queued).toEqual([o1, o3]);
    expect(cols.cooking).toEqual([s1]);
  });

  it("ignores active_order_ids not present in orders", () => {
    const o1 = makeOrder("o1");
    const snap = emptySnapshot({
      orders: [o1],
      active_order_ids: ["o1", "ghost"],
    });

    const cols = deriveKanbanColumns(snap);
    // o1 is active, ghost is silently ignored
    expect(cols.queued).toEqual([]);
  });

  it("routes pending_reviews to review column", () => {
    const review = makeReview("r1");
    const snap = emptySnapshot({
      pending_reviews: [review],
      pending_review_count: 1,
    });

    const cols = deriveKanbanColumns(snap);
    expect(cols.review).toEqual([review]);
  });

  it("routes recent sessions to done column", () => {
    const s1 = makeSession("done1");
    const snap = emptySnapshot({ recent: [s1] });
    const cols = deriveKanbanColumns(snap);
    expect(cols.done).toEqual([s1]);
  });
});
