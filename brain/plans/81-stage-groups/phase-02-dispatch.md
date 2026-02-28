Back to [[plans/81-stage-groups/overview]]

# Phase 2 — Group-aware dispatch

**Routing:** claude / claude-opus-4-6

## Goal

Make `dispatchableStages` return all pending stages in the current group instead of just the first pending stage. This is the core behavioral change — after this phase, multiple stages per order can be dispatched concurrently.

## Changes

### `loop/orders.go` — `dispatchableStages`

Current logic: iterates stages, breaks after finding the first pending stage. New logic:

1. Determine the order's current group via `order.CurrentGroup()`
2. If any stage in the current group is `active` or `merging`, the order has in-flight work — but other pending stages in the same group are still dispatchable
3. Collect all `pending` stages in the current group as candidates
4. Skip orders where the current group has no pending stages

### `loop/orders.go` — `busyTargets`

Current logic: order is busy if any stage is active. New logic: order is busy if all stages in the current group are either active/merging (nothing left to dispatch in this group). An order with 3 group-0 stages where 1 is active and 2 are pending is NOT fully busy — it has dispatchable work.

Actually, `busyTargets` is used to skip orders in `dispatchableStages`. The cleaner approach: remove `busyTargets` from the dispatch path and fold the logic directly into `dispatchableStages`, which already handles the per-stage status check. Keep `busyTargets` only for its other callers (if any).

### `loop/orders.go` — `activeStageForOrder`

This function returns a single stage. With groups, it needs to return all active/pending stages in the current group. Rename to `activeStagesForOrder` returning `[]int` (stage indices). Update all callers.

## Verification

- Write unit tests for `dispatchableStages` with group scenarios:
  - Single group (backward compat): returns one stage at a time
  - Two stages in group 0: returns both
  - Group 0 complete, group 1 pending: returns group 1 stages
  - Group 0 has 1 active + 1 pending: returns the pending one
- `go test ./loop/...`
- `go vet ./...`
