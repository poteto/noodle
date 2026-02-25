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
  queue: QueueItem[];
  active_queue_ids: string[];
  action_needed: string[];
  events_by_session: Record<string, EventLine[]>;
  feed_events: FeedEvent[];
  total_cost_usd: number;
  pending_reviews: PendingReviewItem[];
  pending_review_count: number;
  autonomy: string;
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
  loop_state: string;
  remote_host?: string;
}

export interface QueueItem {
  id: string;
  task_key?: string;
  title?: string;
  prompt?: string;
  provider: string;
  model: string;
  skill?: string;
  plan?: string[];
  rationale?: string;
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
  id: string;
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
}

export interface ControlCommand {
  id?: string;
  action: string;
  item?: string;
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
  action: string;
  status: "ok" | "error";
  message?: string;
  at: string;
}

export interface ConfigDefaults {
  provider: string;
  model: string;
  autonomy: string;
}

// Kanban column derivation from flat snapshot.
export interface KanbanColumns {
  queued: QueueItem[];
  cooking: Session[];
  review: PendingReviewItem[];
  done: Session[];
}

export function deriveKanbanColumns(snapshot: Snapshot): KanbanColumns {
  const activeSet = new Set(snapshot.active_queue_ids);
  return {
    queued: snapshot.queue.filter((item) => !activeSet.has(item.id)),
    cooking: snapshot.active,
    review: snapshot.pending_reviews,
    done: snapshot.recent,
  };
}
