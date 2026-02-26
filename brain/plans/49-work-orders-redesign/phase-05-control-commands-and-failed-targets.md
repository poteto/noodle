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
- `controlReject()` — Calls `failOrder()`, marks order in failed targets.
- `controlRequeue()` — Removes order ID from failed targets. If order still exists in orders file, resets its failed/cancelled stages back to pending.
- `controlMerge()` — Merges a pending-review stage's worktree, then advances the order. If this was the final stage, fires adapter `done`.
- `controlRequestChanges()` — Rejects a pending-review stage. Marks stage failed, cancels remaining stages, marks order failed.

**`loop/failures.go`** — Update key semantics:
- `failedTargets` map remains `map[string]string` — keyed by order ID instead of queue item ID
- `markFailed(orderID, reason)` — unchanged signature, different semantic (order, not item)
- Spawn planning checks `failedTargets` against order IDs
- **Intentional stickiness:** If order "29" fails, a new order with the same ID "29" from the scheduler is blocked until `controlRequeue` clears the failure. This prevents infinite retry loops — same behavior as current queue items. The scheduler should not re-create work for failed IDs without human intervention.

**`loop/cycle_spawn_plan.go`** — Already migrated in phase 4 to use `dispatchableStages()`. Verify `failedTargets` filtering works with order IDs.

## Data structures

- No new types — uses Order/Stage from phase 1
- Control command payloads may need `order_id` field instead of `item_id` (check existing ControlCommand struct)

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
- Unit test: controlReject marks order failed in both orders file and failed targets
- Unit test: controlRequeue resets failed order stages to pending
