// Re-exports generated Go types with narrowed enum fields.
// Generated interfaces use `string` where Go has untyped constants;
// we override those fields with the unions from enums.ts.

import type {
  Snapshot as RawSnapshot,
  Session as RawSession,
  Order as RawOrder,
  Stage as RawStage,
  EventLine as RawEventLine,
} from "./generated-types";
import type { LoopState, Health, TraceFilter, StageStatus, OrderStatus } from "./enums";
export type { FeedEvent } from "./generated-types";
export type { PendingReviewItem } from "./generated-loop-types";

export type { LoopState, Health, TraceFilter, StageStatus, OrderStatus } from "./enums";

export interface EventLine extends Omit<RawEventLine, "category"> {
  category: TraceFilter;
}

// Narrow the `string` fields that correspond to enum-like Go constants.
export interface Snapshot extends Omit<
  RawSnapshot,
  "loop_state" | "sessions" | "active" | "recent" | "orders" | "events_by_session"
> {
  loop_state: LoopState;
  sessions: Session[];
  active: Session[];
  recent: Session[];
  orders: Order[];
  events_by_session: Record<string, EventLine[]>;
}

export interface Session extends Omit<RawSession, "health" | "loop_state"> {
  health: Health;
  loop_state: LoopState;
}

export interface Stage extends Omit<RawStage, "status"> {
  status: StageStatus;
}

export interface Order extends Omit<RawOrder, "status" | "stages"> {
  stages: Stage[];
  status: OrderStatus;
}

export interface DiffResponse {
  diff: string;
  stat: string;
}

// Stage-level fields shared by enqueue, edit-item, and add-stage.
interface StageFields {
  prompt?: string;
  task_key?: string;
  skill?: string;
  provider?: string;
  model?: string;
}

// Discriminated union — each action carries only its valid fields.
// Adding the wrong field (e.g. `name` on a steer) is a compile error.
export type ControlCommand = { id?: string } & (
  | { action: "pause" }
  | { action: "resume" }
  | { action: "drain" }
  | { action: "stop-all" }
  | { action: "skip"; order_id: string }
  | { action: "merge"; order_id: string }
  | { action: "reject"; order_id: string }
  | { action: "requeue"; order_id: string }
  | { action: "advance"; order_id: string }
  | { action: "kill"; name: string }
  | { action: "stop"; name: string }
  | { action: "steer"; target: string; prompt: string }
  | { action: "request-changes"; order_id: string; prompt?: string }
  | { action: "park-review"; order_id: string; prompt?: string }
  | { action: "mode"; value: string }
  | { action: "set-max-cooks"; value: string }
  | { action: "reorder"; order_id: string; value: string }
  | ({ action: "enqueue"; order_id: string } & StageFields)
  | ({ action: "edit-item"; order_id: string } & StageFields)
  | ({ action: "add-stage"; order_id: string; task_key: string } & Omit<StageFields, "task_key">)
);

export type ControlAction = ControlCommand["action"];

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
  mode: string;
  task_types: string[];
}

export type ChannelId = { type: "scheduler" } | { type: "agent"; sessionId: string };
