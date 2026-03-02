import type { Snapshot, Session, Order, Stage, PendingReviewItem } from "~/client";

export function buildStage(overrides?: Partial<Stage>): Stage {
  return {
    provider: "claude",
    model: "opus",
    status: "pending",
    ...overrides,
  };
}

export function buildOrder(overrides?: Partial<Order>): Order {
  return {
    id: `order-${Math.random().toString(36).slice(2, 8)}`,
    stages: [buildStage()],
    status: "active",
    ...overrides,
  };
}

export function buildSession(overrides?: Partial<Session>): Session {
  const id = overrides?.id ?? `session-${Math.random().toString(36).slice(2, 8)}`;
  return {
    id,
    display_name: id,
    status: "running",
    runtime: "local",
    provider: "claude",
    model: "opus",
    total_cost_usd: 0.42,
    duration_seconds: 120,
    last_activity: new Date().toISOString(),
    current_action: "Edit src/main.ts",
    health: "green",
    context_window_usage_pct: 35,
    retry_count: 0,
    idle_seconds: 0,
    loop_state: "running",
    ...overrides,
  };
}

export function buildReview(overrides?: Partial<PendingReviewItem>): PendingReviewItem {
  const orderId = overrides?.order_id ?? `order-${Math.random().toString(36).slice(2, 8)}`;
  return {
    order_id: orderId,
    stage_index: 0,
    worktree_name: `wt-${orderId}`,
    worktree_path: `/tmp/wt-${orderId}`,
    ...overrides,
  };
}

export function buildSnapshot(overrides?: Partial<Snapshot>): Snapshot {
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
    mode: "supervised",
    max_concurrency: 4,
    warnings: [],
    ...overrides,
  };
}
