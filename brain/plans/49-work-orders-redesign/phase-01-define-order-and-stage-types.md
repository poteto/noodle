Back to [[plans/49-work-orders-redesign/overview]]

# Phase 1: Define Order and Stage types

## Goal

Define the new `Order`, `Stage`, and `OrdersFile` types that replace `QueueItem` and `Queue`. These are the foundational data structures — get them right and downstream code follows naturally.

## Changes

**`loop/types.go`** — Add new types alongside existing ones (old types deleted in phase 9):
- `Stage` — unit of work within an order. Carries the fields from QueueItem that are stage-specific: `TaskKey`, `Prompt`, `Skill`, `Provider`, `Model`, `Runtime`, `Status`.
- `Order` — a pipeline of stages. Carries grouping fields: `ID`, `Title`, `Plan`, `Rationale`, `Stages`, `Status`.
- `OrdersFile` — top-level file structure: `GeneratedAt`, `Orders`, `ActionNeeded`. Replaces `Queue`.

**`internal/queuex/queue.go`** — Add parallel types (`orderx.Stage`, `orderx.Order`, `orderx.OrdersFile`) mirroring the loop types. These are the serialization types the I/O layer uses. Keep existing `queuex` types for now — they'll be deleted in phase 9.

## Data structures

- `Stage.Status`: one of `"pending"`, `"active"`, `"completed"`, `"failed"`, `"cancelled"`
- `Order.Status`: one of `"active"`, `"completed"`, `"failed"`
- Stage fields match current QueueItem fields (same JSON keys) so the scheduler's mental model stays consistent
- Order-level fields (`Plan`, `Rationale`) move off the stage — they describe the work unit, not individual steps

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Architectural decision — types shape everything downstream.

## Verification

### Static
- `go build ./...` passes
- New types are used only in tests at this point — no production callers yet
- JSON tags match expected schema

### Runtime
- Unit tests: JSON marshal/unmarshal round-trip for Order, Stage, OrdersFile
- Test that status constants are correct strings
- Test that an OrdersFile with mixed stage statuses serializes correctly
