Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 2 — Replace `canMergeStage` with runtime detection

## Goal

Swap the static `canMergeStage()` registry lookup for a runtime `worktreeHasChanges()` check. After this phase, merge decisions are based on actual worktree state, not frontmatter declarations.

## Changes

**`loop/cook_merge.go`**:
- Delete `canMergeStage()` method (lines 101-110)
- Add `worktreeHasChanges(cook *cookHandle) (bool, error)` with two-path check:
  1. Check sync result first — if `readSessionSyncResult` returns a `SyncResultTypeBranch` with a non-empty branch, return `true` (sprites pushed to a remote branch)
  2. If no sync result, check local worktree — if `cook.worktreeName` is empty, return `false`; otherwise call `l.deps.Worktree.HasUnmergedCommits(cook.worktreeName)` and propagate errors

This mirrors the existing two-path merge in `mergeCookWorktree` (local vs `MergeRemoteBranch`) but moves the detection upstream of the merge pipeline.

**`loop/cook_completion.go`**:
- Replace `canMerge := l.canMergeStage(cook.stage)` with `canMerge, err := l.worktreeHasChanges(cook)`. On error, fail the stage (don't silently advance).

**`loop/control_review.go`**:
- Same replacement in `controlMerge()` (line 41). On error, return the error.

**`loop/loop_merge_reconcile_test.go`**:
- Rewrite `TestApprovalAutoCanMergeTrueAutoMerges` → `TestAutoMergeWithLocalChanges` — set `fakeWorktree.hasUnmergedCommits["..."] = true`
- Rewrite `TestApprovalAutoCanMergeFalseAdvances` → `TestAutoAdvanceWithoutChanges` — set `fakeWorktree.hasUnmergedCommits["..."] = false`
- Add `TestAutoMergeWithRemoteSyncResult` — local worktree has no commits, but spawn.json contains a sync result with `type: "branch"`. Verify the stage enters the merge pipeline and calls `MergeRemoteBranch`.

**`loop/sous_chef_test.go`**:
- Remove `CanMerge: false` from test task type fixtures (field will be deleted in phase 3, but stop using it here)

## Data structures

- `worktreeHasChanges` method on `*Loop` — returns `(bool, error)`, no new types

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./loop/...
```

- Tests that previously relied on `CanMerge: false` now use `hasUnmergedCommits` map on `fakeWorktree`
- Verify the "execute stage with no commits" case: set `hasUnmergedCommits[name] = false`, confirm stage advances without entering merge pipeline
- Verify the "execute stage with commits" case: default `true`, confirm merge pipeline runs
- Verify the "sprites remote-sync" case: local worktree has no commits, spawn.json has sync result with branch — confirm stage enters merge pipeline and calls `MergeRemoteBranch`
- Verify error propagation: `HasUnmergedCommits` returns error → stage fails, does not advance
