Back to [[plans/49-work-orders-redesign/overview]]

# Phase 5: Control commands and failed targets

## Goal

Migrate all control commands and the failed targets system to operate on orders instead of queue items.

## Changes

**`loop/control.go`** — Migrate control commands:
- `controlEnqueue()` — Creates a new single-stage order (user-added tasks are always single-stage). Reads orders, appends new order, writes atomically.
- `controlEditItem()` — Finds order by ID, edits order-level fields (Title) or stage-level fields (Prompt, TaskKey, Provider, Model, Skill) on the current pending stage. Error if order not found or no editable stage.
- `controlReorder()` — Reorders orders (not stages within an order). Removes order from old position, inserts at new position.
- `controlSkip()` (was skip on queue item) — Calls `cancelOrder()` to cancel remaining stages, removes order from orders file.
- `controlReject()` — Calls `cancelOrder()` (user-initiated rejection skips OnFailure — the human decided to reject), marks order in failed targets.
- `controlRequeue()` — Removes order ID from failed targets. If order still exists in orders file, resets all failed/cancelled stages in both `Stages` and `OnFailure` back to `"pending"`, and sets `Order.Status` back to `"active"`. This is a full reset — the order re-runs from the first incomplete stage.
- `controlMerge()` — Merges a pending-review stage's worktree, then advances the order. If `advanceOrder` returns `removed=true`: for `"active"` orders, fire adapter `done`; for `"failing"` orders, call `markFailed`.
- `controlRequestChanges()` — Rejects a pending-review stage. Calls `failStage()` — if OnFailure stages exist, they run (e.g., debugging). If no OnFailure, marks order terminally failed. Only calls `markFailed` if `failStage` returns `terminal=true`.

**`loop/failures.go`** — Update key semantics:
- `failedTargets` map remains `map[string]string` — keyed by order ID instead of queue item ID
- `markFailed(orderID, reason)` — unchanged signature, different semantic (order, not item)
- Spawn planning checks `failedTargets` against order IDs
- **Intentional stickiness:** If order "29" fails, a new order with the same ID "29" from the scheduler is blocked until `controlRequeue` clears the failure. This prevents infinite retry loops — same behavior as current queue items. The scheduler should not re-create work for failed IDs without human intervention.

**`loop/cycle_spawn_plan.go`** — Already migrated in phase 4 to use `dispatchableStages()`. Verify `failedTargets` filtering works with order IDs.

## Data structures

- No new types — uses Order/Stage from phase 1
- Rename `ControlCommand` payload field from `item_id` to `order_id` (or equivalent — check existing struct and update all references)

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical migration — same patterns, new types.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- No remaining references to `QueueItem` in control.go

### Runtime
- Unit test: controlEnqueue creates single-stage order
- Unit test: controlEditItem modifies stage fields on pending stage
- Unit test: controlReorder changes order position
- Unit test: controlSkip cancels all remaining stages
- Unit test: controlReject removes order from orders file and marks order ID in failed targets
- Unit test: controlRequeue resets failed order stages to pending (both Stages and OnFailure)
- Unit test: controlRequeue on a `"failing"` order resets status to `"active"`
- Unit test: controlReject skips OnFailure (user rejection is terminal)
- Unit test: controlRequestChanges with OnFailure stages — calls failStage, order becomes "failing", markFailed not called
- Unit test: controlRequestChanges without OnFailure — calls failStage, terminal=true, calls markFailed
- Unit test: controlMerge on final stage of active order fires adapter "done"
- Unit test: controlMerge on final OnFailure stage of failing order calls markFailed (not adapter "done")
- Unit test: failed-target stickiness — new order with same ID as failed order is blocked until controlRequeue clears it
