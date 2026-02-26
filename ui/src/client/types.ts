// Mirrors Go snapshot.* and loop.* types.
// All fields match the JSON keys from the Go server.

export type TraceFilter = "all" | "tools" | "think" | "ticket";
export type LoopState = "running" | "paused" | "draining" | "idle";
export type Health = "green" | "yellow" | "red";

export interface Snapshot {
  updated_at: string;
  loop_state: LoopState;
  sessions: Session[];
  active: Session[];
  recent: Session[];
  orders: Order[];
  active_order_ids: string[];
  action_needed: string[];
  events_by_session: Record<string, EventLine[]>;
  feed_events: FeedEvent[];
  total_cost_usd: number;
  pending_reviews: PendingReviewItem[];
  pending_review_count: number;
  autonomy: string;
  max_cooks: number;
}

export interface Session {
  id: string;
  display_name: string;
  status: string;
  runtime: string;
  provider: string;
  model: string;
  total_cost_usd: number;
  duration_seconds: number;
  last_activity: string;
  current_action: string;
  health: Health;
  context_window_usage_pct: number;
  retry_count: number;
  idle_seconds: number;
  stuck_threshold_seconds: number;
  loop_state: LoopState;
  remote_host?: string;
  dispatch_warning?: string;
  worktree_name?: string;
  task_key?: string;
  title?: string;
}

export type StageStatus = "pending" | "active" | "completed" | "failed" | "cancelled";
export type OrderStatus = "active" | "completed" | "failed" | "failing";

export interface Stage {
  task_key?: string;
  prompt?: string;
  skill?: string;
  provider?: string;
  model?: string;
  runtime?: string;
  status: StageStatus;
  extra?: Record<string, unknown>;
}

export interface Order {
  id: string;
  title?: string;
  plan?: string[];
  rationale?: string;
  stages: Stage[];
  on_failure?: Stage[];
  status: OrderStatus;
}

export interface EventLine {
  at: string;
  label: string;
  body: string;
  category: TraceFilter;
}

export interface FeedEvent {
  session_id: string;
  agent_name: string;
  task_type: string;
  at: string;
  label: string;
  body: string;
  category: string;
}

export interface PendingReviewItem {
  order_id: string;
  stage_index: number;
  task_key?: string;
  title?: string;
  prompt?: string;
  provider?: string;
  model?: string;
  runtime?: string;
  skill?: string;
  plan?: string[];
  rationale?: string;
  worktree_name: string;
  worktree_path: string;
  session_id?: string;
  reason?: string;
}

export interface DiffResponse {
  diff: string;
  stat: string;
}

export type ControlAction =
  | "pause"
  | "resume"
  | "drain"
  | "skip"
  | "kill"
  | "steer"
  | "merge"
  | "reject"
  | "request-changes"
  | "autonomy"
  | "enqueue"
  | "stop-all"
  | "requeue"
  | "edit-item"
  | "reorder"
  | "stop"
  | "set-max-cooks";

export interface ControlCommand {
  id?: string;
  action: ControlAction;
  order_id?: string;
  name?: string;
  target?: string;
  prompt?: string;
  value?: string;
  task_key?: string;
  provider?: string;
  model?: string;
  skill?: string;
}

export interface ControlAck {
  id: string;
  action: ControlAction;
  status: "ok" | "error";
  message?: string;
  at: string;
}

export interface ConfigDefaults {
  provider: string;
  model: string;
  autonomy: string;
  task_types: string[];
}

// Kanban column derivation from flat snapshot.
export interface KanbanColumns {
  queued: Order[];
  cooking: Session[];
  review: PendingReviewItem[];
  done: Session[];
}

export function deriveKanbanColumns(snapshot: Snapshot): KanbanColumns {
  const activeSet = new Set(snapshot.active_order_ids);
  return {
    queued: snapshot.orders.filter((order) => !activeSet.has(order.id)),
    cooking: snapshot.active,
    review: snapshot.pending_reviews,
    done: snapshot.recent,
  };
}
