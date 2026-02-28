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
