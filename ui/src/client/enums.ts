// Hand-written narrow union types for Go string constants.
// Tygo generates `string` for these — these unions provide type safety.

export type LoopState = "running" | "paused" | "draining" | "idle";
export type Health = "green" | "yellow" | "red";
export type TraceFilter = "all" | "tools" | "think" | "ticket";

// UI-only enums (no Go constants exist).
export type StageStatus = "pending" | "active" | "merging" | "completed" | "failed" | "cancelled";
export type OrderStatus = "active" | "completed" | "failed" | "failing";
