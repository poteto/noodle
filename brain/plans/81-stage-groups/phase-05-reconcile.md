Back to [[plans/81-stage-groups/overview]]

# Phase 5 — Crash recovery with parallel groups

**Routing:** codex / gpt-5.3-codex

## Goal

Ensure the reconcile logic handles crash recovery when multiple stages in a group were active at crash time.

## Changes

### `loop/reconcile.go` — `adoptedTargets` map

`adoptedTargets` is currently `map[string]string` keyed by bare `orderID`. With parallel groups, a crashed loop may have had multiple sessions for the same order. Last-write-wins means only one session gets tracked; the others are orphaned.

Change `adoptedTargets` to use composite keys: `map[string]string` keyed by `cookKey(orderID, stageIndex)`. This requires the runtime's `RecoveredSession` to carry `StageIndex` alongside `OrderID` — update the recovery path to extract it from the session metadata.

### `loop/reconcile.go` — `reconcileMergingStages`

The adopted check at line 118 uses `adoptedTargets[ms.order.ID]` (bare orderID). With composite keys, update to `adoptedTargets[cookKey(ms.order.ID, ms.stageIdx)]`. Without this fix, if only one of two crashed sessions is recovered, the other merging stage gets incorrectly reset to `active` with no corresponding session.

### `loop/adopted_helpers.go` — `buildAdoptedCook`

`buildAdoptedCook` calls `activeStageForOrder(order)` which returns the first active/pending stage. With parallel groups, this returns the wrong stage. Update to accept and use a specific `stageIndex` parameter, derived from the composite key in `adoptedTargets`.

### `loop/reconcile.go` — `advanceAndPersist` in reconcile path

`reconcileMergingStages` at line 144 builds a synthetic `cookHandle` and calls `advanceAndPersist`. After Phase 4 changes `advanceOrder` to accept `stageIndex`, this path already passes `ms.stageIdx` via `cook.stageIndex` — verify it flows through correctly.

## Verification

- Write a test that simulates crash recovery with 2 active stages in the same group:
  - Both stages should be reconciled (adopted or reset)
  - Neither should block the other
- `go test ./loop/...`
- Manual test: start noodle with a multi-group order, kill the process mid-execution, restart and verify recovery
