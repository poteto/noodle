Back to [[plans/81-stage-groups/overview]]

# Phase 3 — Group-aware dispatch

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

`busyTargets` has exactly one caller: `planCycleSpawns` at `loop_cycle_pipeline.go:139`. Remove `busyTargets` entirely and fold the group-aware busy logic into `dispatchableStages`. Update `planCycleSpawns` to no longer call `busyTargets` — the busy-set construction from `activeCooksByOrder` keys (line 161) already uses composite keys (Phase 2), so extract orderIDs from values instead.

### `loop/orders.go` — `activeStageForOrder`

This function returns a single stage. With groups, it needs to return all active/pending stages in the current group. Rename to `activeStagesForOrder` returning `[]int` (stage indices). Update all callers:
- `adopted_helpers.go:39` — `buildAdoptedCook`
- `control_orders.go:77` — `controlEditItem`
- `schedule.go:84` — `spawnSchedule`
- `control_scheduler.go:25` — `controlScheduler` order scan
- `control_scheduler.go:121` — `controlParkReview`

## Verification

- Write unit tests for `dispatchableStages` with group scenarios:
  - Two stages in group 0: returns both
  - Group 0 complete, group 1 pending: returns group 1 stages
  - Group 0 has 1 active + 1 pending: returns the pending one
- `go test ./loop/...`
- `go vet ./...`
