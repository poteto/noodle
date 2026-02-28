Back to [[plans/81-stage-groups/overview]]

# Phase 5 — Crash recovery with parallel groups

**Routing:** codex / gpt-5.3-codex

## Goal

Ensure the reconcile logic handles crash recovery when multiple stages in a group were active at crash time.

## Changes

### `loop/reconcile.go`

Current `reconcileMergingStages`: iterates all orders and stages, handles stuck `merging` and `active` stages. With parallel groups, multiple stages per order could be in `active` or `merging` state simultaneously.

Review the reconcile logic:
- If it already iterates all stages (not just the first active), it may work unchanged
- If it breaks after finding one active stage, update to handle all active stages in a group
- Ensure it doesn't try to adopt the same worktree twice or create conflicting session names

### `loop/cook_spawn.go` — `spawnOptions.adopted`

The adopt path (reconciling a session from a previous loop instance) needs to handle multiple adopted sessions for the same order. Verify `adoptedTargets` tracking works with composite keys.

## Verification

- Write a test that simulates crash recovery with 2 active stages in the same group:
  - Both stages should be reconciled (adopted or reset)
  - Neither should block the other
- `go test ./loop/...`
- Manual test: start noodle with a multi-group order, kill the process mid-execution, restart and verify recovery
