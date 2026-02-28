Back to [[plans/81-stage-groups/overview]]

# Phase 4 — Group-aware advancement and failure

**Routing:** claude / claude-opus-4-6

## Goal

When a stage completes, only advance to the next group when all stages in the current group are done. When a stage fails, cancel remaining stages in the group and fail the order.

## Changes

### `loop/orders.go` — `advanceOrder`

Current: finds first active/pending stage, marks it completed, checks if all done. New:

1. Accept `stageIndex int` parameter — mark that specific stage completed
2. Check if all stages in the completed stage's group are now completed
3. If the group is complete and there are more groups, the order stays active (next tick dispatches the next group)
4. If all stages across all groups are completed, remove the order (order complete)

### `loop/orders.go` — `failStage`

Current: fails first active/pending stage, removes order. New:

1. Accept `stageIndex int` parameter — mark that specific stage failed
2. Cancel all other pending/active stages in the same group (set to `cancelled`)
3. Remove the order (order failed) — same as current behavior

### `loop/cook_completion.go` — `handleCompletion` failure path

When a stage fails, kill all sibling cook sessions before removing the order. Without this, watcher goroutines for sibling stages remain alive and complete into a removed order, causing `advanceOrder` to return "order not found" errors.

1. Use `cooksByOrder(cook.orderID)` to find all active cooks for the order
2. Call `Kill()` on each sibling's session (skip the failing cook itself)
3. Delete each sibling from `activeCooksByOrder`
4. Then call `failStage` and remove the order

### `loop/orders.go` — `advanceOrder` return values

Add `groupComplete bool` to the return signature alongside `removed bool`. This lets `advanceAndPersist` distinguish "stage done, group still running" from "group done, order continues" for event emission and logging.

Group is complete when all stages in the group have status `completed` (not `active`, not `merging`, not `pending`).

### `loop/cook_completion.go` — `handleCompletion`

Update to pass `cook.stageIndex` to `advanceOrder` and `failStage`.

### `loop/cook_completion.go` — `advanceAndPersist`

When `advanceOrder` returns `removed=false` (group complete but order has more groups), emit a `stage.completed` event but NOT `order.completed`. Only emit `order.completed` when the order is actually removed.

When a group completes but order continues, cancel the cook's active status — the next tick cycle will dispatch stages from the next group.

## Verification

- Unit tests for `advanceOrder` with groups:
  - Single stage completes → order removed
  - Group of 2: first completes → order stays, second completes → order removed
  - Group 0 complete → group 1 dispatchable
- Unit tests for `failStage` with groups:
  - Stage in group fails → other pending stages in group cancelled, order removed
- `go test ./loop/...`
- Integration: run noodle with a multi-group order, verify stages dispatch in group order
