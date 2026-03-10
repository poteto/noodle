Back to [[plans/81-stage-groups/overview]]

# Phase 5 — Crash recovery with parallel groups

**Routing:** codex / gpt-5.4

## Goal

Ensure the reconcile logic handles crash recovery when multiple stages in a group were active at crash time.

## Changes

### `loop/reconcile.go` — `adoptedTargets` map

`adoptedTargets` is currently `map[string]string` keyed by bare `orderID`. With parallel groups, a crashed loop may have had multiple sessions for the same order. Last-write-wins means only one session gets tracked; the others are orphaned.

Change `adoptedTargets` to use composite keys: `map[string]string` keyed by `cookKey(orderID, stageIndex)`. This requires two changes:

1. The runtime's `RecoveredSession` struct (`runtime/runtime.go:40`) must carry `StageIndex` alongside `OrderID`. Currently `process_recover.go:73` only recovers `OrderID`.
2. The dispatch metadata written at session spawn (`dispatcher/dispatch_metadata.go`) must include `stageIndex` so recovery can read it back. Add `StageIndex int` to `dispatchMetadata` and `DispatchRequest`.

### `loop/reconcile.go` — `reconcileMergingStages`

The adopted check at line 118 uses `adoptedTargets[ms.order.ID]` (bare orderID). With composite keys, update to `adoptedTargets[cookKey(ms.order.ID, ms.stageIdx)]`. Without this fix, if only one of two crashed sessions is recovered, the other merging stage gets incorrectly reset to `active` with no corresponding session.

### `loop/adopted_helpers.go` — `buildAdoptedCook`

`buildAdoptedCook` calls `activeStageForOrder(order)` which returns the first active/pending stage. With parallel groups, this returns the wrong stage. Update to accept and use a specific `stageIndex` parameter, derived from the composite key in `adoptedTargets`.

### `loop/reconcile.go` — `reconcileMergingStages` iteration safety

`reconcileMergingStages` snapshots all merging stages, then mutates/removes orders during iteration via `failMergingStage` → `failStage`. With multiple merging stages in the same order, the first failure removes the order, and subsequent entries for that order hit "order not found." Guard the iteration: after processing each entry, check if the order still exists before continuing to the next entry for the same order.

### `loop/reconcile.go` — `advanceAndPersist` in reconcile path

`reconcileMergingStages` at line 144 builds a synthetic `cookHandle` and calls `advanceAndPersist`. After Phase 4 changes `advanceOrder` to accept `stageIndex`, this path already passes `ms.stageIdx` via `cook.stageIndex` — verify it flows through correctly.

## Verification

- Write a test that simulates crash recovery with 2 active stages in the same group:
  - Both stages should be reconciled (adopted or reset)
  - Neither should block the other
- `go test ./loop/...`
- Manual test: start noodle with a multi-group order, kill the process mid-execution, restart and verify recovery
