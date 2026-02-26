Back to [[plans/49-work-orders-redesign/overview]]

# Phase 1: Define Order and Stage types

## Goal

Define the new `Order`, `Stage`, and `OrdersFile` types that replace `QueueItem` and `Queue`. These are the foundational data structures — get them right and downstream code follows naturally.

## Changes

**`loop/types.go`** — Add new types alongside existing ones (old types deleted in phase 9):
- `Stage` — unit of work within an order. Carries the fields from QueueItem that are stage-specific: `TaskKey`, `Prompt`, `Skill`, `Provider`, `Model`, `Runtime`, `Status`, `Extra` (`map[string]json.RawMessage` for arbitrary skill-specific metadata — preserved through serialization round-trips, never interpreted by Go).
- `Order` — a pipeline of stages. Carries grouping fields: `ID`, `Title`, `Plan`, `Rationale`, `Stages`, `Status`, `OnFailure` (`[]Stage` — optional secondary pipeline that runs when a stage fails after retries are exhausted. If empty, failure cancels remaining stages as before).
- `OrdersFile` — top-level file structure: `GeneratedAt`, `Orders`, `ActionNeeded`. Replaces `Queue`.

**`internal/queuex/queue.go`** — Add parallel types (`Stage`, `Order`, `OrdersFile`) in the `queuex` package, mirroring the loop types. These are the serialization types the I/O layer uses. Keep existing `queuex` queue types for now — they'll be deleted in phase 9. (Optional rename of `queuex` → `orderx` happens in phase 9.)

## Data structures

- `Stage.Status`: one of `"pending"`, `"active"`, `"completed"`, `"failed"`, `"cancelled"`
- `Stage.Extra`: `map[string]json.RawMessage` — opaque bag for skill-specific data. Go never reads it; just round-trips it through JSON serialization. Schedulers and skills use it for custom metadata (priority, tags, deadline, etc.)
- `Order.Status`: one of `"active"`, `"completed"`, `"failed"`, `"failing"` (running OnFailure stages). Note: `"completed"` and `"failed"` are transient — the lifecycle functions remove terminal orders from the OrdersFile immediately (same as current `skipQueueItem`). These statuses exist for type completeness and may appear briefly in memory but not in persisted `orders.json`.
- `Order.OnFailure`: `[]Stage` — optional. When present and a stage fails, the loop switches to running these stages instead of cancelling. When all OnFailure stages complete, the order status becomes `"failed"` (the original failure is not erased — OnFailure is remediation, not recovery). When OnFailure is empty or absent, failure cancels remaining stages as before.
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
- Test that `Stage.Extra` round-trips through JSON without data loss (arbitrary JSON preserved)
- Test that `Order.OnFailure` serializes/deserializes correctly, including empty/nil case
- Test that `Order.Status` includes the `"failing"` state
