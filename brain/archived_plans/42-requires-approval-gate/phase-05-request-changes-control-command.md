Back to [[archived_plans/42-requires-approval-gate/overview]]

# Phase 5: Request Changes Control Command

## Goal

Add a `request-changes` control command that takes feedback from the user and spawns a new agent in the same worktree to address the feedback. This enables the approve -> request changes -> fix -> re-approve cycle.

## Changes

### `loop/control.go` — `applyControlCommand`

Add `case "request-changes"` to the switch. Delegates to new `controlRequestChanges(itemID, prompt)`.

### `loop/control.go` — `controlRequestChanges`

New function:

1. Look up the item in `pendingReview`
2. Spawn a new cook in the **same worktree** using the worktree name/path from `pendingReviewCook` (not derived from queue item — `cookBaseName` may differ from the original worktree name due to retry suffixes)
3. Only remove from `pendingReview` AFTER successful spawn — if dispatch fails, the item stays parked so the user can retry
4. Pass the feedback as resume prompt: "Previous work needs changes. Feedback: {prompt}"
5. The new cook re-enters `activeByTarget` / `activeByID` as normal

Note: `spawnCook` derives worktree name from `cookBaseName(item)`, which won't match the `pendingReviewCook.worktreeName` if the original was a retry. Either pass the worktree name explicitly or add a `requestChanges`-specific spawn path that takes the name from `pendingReviewCook`.

### Fix delete-before-side-effects in merge/reject

The existing `controlMerge` and `controlReject` commands delete the pending item from `pendingReview` before running side effects (worktree merge, adapter call, cleanup). If the side effect fails, the item is lost and cannot be retried. Now that parking is the normal approval path, this must be fixed: remove from `pendingReview` (and its persistence file) only AFTER the side effects succeed. Apply the same "remove after success" pattern used for `request-changes` to both `controlMerge` and `controlReject`.

### `loop/types.go` — `ControlCommand`

Already has `Prompt string` field — use it for the feedback text. No struct change needed.

### Tests

- `loop/loop_test.go` — add `request-changes` control command tests (existing control-path tests are in this file, around line 483):
  - Item in `pendingReview` -> spawns new cook with feedback prompt, removes from `pendingReview`
  - Item not in `pendingReview` -> error
  - Empty prompt -> still works (edge case, user might just want a redo)
  - Dispatch failure -> item stays in `pendingReview` (not lost)

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — new behavior with edge cases.

## Verification

```sh
go test ./loop/... -run TestControl
# Manual: park a cook, send request-changes via control.ndjson, verify new session spawns
```
