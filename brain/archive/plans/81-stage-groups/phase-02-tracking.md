Back to [[plans/81-stage-groups/overview]]

# Phase 2 — Multi-cook tracking per order

**Routing:** claude / claude-opus-4-6

## Goal

Change cook tracking from one-cook-per-order to multiple-cooks-per-order using a composite key. This unblocks spawning multiple stages from the same order.

## Changes

### `loop/cook_tracker.go` — cook tracking map

Change `activeCooksByOrder map[string]*cookHandle` to use a composite key `"orderID:stageIndex"`. Add helper functions:

- `cookKey(orderID string, stageIndex int) string` — builds the composite key
- `cooksByOrder(orderID string) []*cookHandle` — returns all cooks for an order
- `removeCook(key string)` — removes a cook from the map

These helpers keep the key format encapsulated.

### `loop/cook_spawn.go` — `spawnCook`

Update the tracking write: `l.cooks.activeCooksByOrder[cookKey(cand.OrderID, cand.StageIndex)] = cook`

Update `atMaxConcurrency()` — currently counts `len(activeCooksByOrder)`. This still works since each cook has its own key.

### `loop/cook_completion.go`

Update cook lookup on completion to use the composite key. The `cookHandle` already stores `orderID` and `stageIndex`, so the key can be reconstructed.

### All callers of `activeCooksByOrder` — exhaustive list

Every site that reads or writes the map must be updated. Key-based lookups and deletes break silently with composite keys if missed.

**Write by orderID → use `cookKey(orderID, stageIndex)`:**
- `cook_spawn.go:139` — `spawnCook` tracking write
- `schedule.go:131` — `spawnSchedule` tracking write

**Lookup by orderID → use `cooksByOrder(orderID)`:**
- `control_orders.go:59` — `controlEditItem` busy check
- `control_scheduler.go:134` — `controlParkReview` session/worktree lookup

**Delete by `cook.orderID` → delete by `cookKey(cook.orderID, cook.stageIndex)`:**
- `cook_completion.go:78` — `applyStageResult` delete after processing
- `control.go:323` — `controlStopKill` delete after kill
- `cook_steer.go:122` — `steerRespawn` delete before re-spawn

**Lookup by `result.OrderID` → use `cookKey(result.OrderID, result.StageIndex)`:**
- `cook_completion.go:70` — `applyStageResult` lookup

**Key iteration → extract orderID from composite key or iterate values:**
- `loop_cycle_pipeline.go:161` — `planCycleSpawns` populates `busySet` from map keys. Must extract the orderID portion, not use the raw composite key.
- `stamp_status.go:14-15` — builds `active[]` array for `status.json`. Must deduplicate by orderID, not emit composite keys.

**Key+value iteration → use `cook.orderID` from value, not the map key:**
- `loop.go:354` — shutdown drain logs the map key as `orderID`. Use `cook.orderID` instead.

**Value iteration → unchanged:**
- `loop.go:166` — `Shutdown` kill-all
- `cook_completion.go:254` — `forwardToScheduler` scan
- `cook_spawn.go:39` — `atMaxConcurrency` len check
- `cook_spawn.go:63` — duplicate target check
- `control.go:281,291` — `controlStopAll`/`controlKillAll` iteration
- `state_snapshot.go:60,62` — `CookSummary` building
- `cook_steer.go:20,44` — steer iteration
- `schedule.go:43` — schedule cook scan
- `loop_cycle_pipeline.go:93` — empty-cooks check
- `loop_cycle_pipeline.go:140,175` — capacity sizing

## Verification

- `go test ./loop/...` — all existing tests pass (no behavioral change yet, just tracking refactor)
- `go vet ./...`
- Manual verification: start noodle, dispatch an order, verify cook tracking works as before
