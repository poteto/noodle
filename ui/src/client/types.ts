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
export type { FeedEvent } from "./generated-types";
export type { PendingReviewItem } from "./generated-loop-types";

export type {
  LoopState,
  Health,
  TraceFilter,
  StageStatus,
  OrderStatus,
} from "./enums";

import type { LoopState, Health, TraceFilter, StageStatus, OrderStatus } from "./enums";

export interface EventLine extends Omit<RawEventLine, "category"> {
  category: TraceFilter;
}

// Narrow the `string` fields that correspond to enum-like Go constants.
export interface Snapshot extends Omit<RawSnapshot, "loop_state" | "sessions" | "active" | "recent" | "orders" | "events_by_session"> {
  loop_state: LoopState;
  sessions: Session[];
  active: Session[];
  recent: Session[];
  orders: Order[];
  events_by_session: { [key: string]: EventLine[] };
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

export type ChannelId =
  | { type: "scheduler" }
  | { type: "agent"; sessionId: string };
