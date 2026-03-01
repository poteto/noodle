Back to [[plans/89-simplify-task-type-frontmatter/overview]]

# Phase 2 — Replace `canMergeStage` with runtime detection

## Goal

Swap the static `canMergeStage()` registry lookup for a runtime `worktreeHasChanges()` check. After this phase, merge decisions are based on actual worktree state, not frontmatter declarations.

## Changes

**`loop/cook_merge.go`**:
- Delete `canMergeStage()` method (lines 101-110)
- Add `worktreeHasChanges(cook *cookHandle) (bool, error)` with three-path check:
  1. Check persisted merge metadata first — if the stage's status is `"merging"` AND its `Extra` map has `merge_branch`, use that. The status check is critical: only `"merging"` indicates a crash mid-merge. A freshly completed or requeued stage may have stale `Extra` from a previous failed attempt — `failStage`/`resetStages` don't clear `Extra`. This handles `controlMerge` after restart when `spawn.json` may be gone.
  2. Check sync result — if `readSessionSyncResult` returns a `SyncResultTypeBranch` with a non-empty branch, return `true` (sprites pushed to a remote branch)
  3. Check local worktree — if `cook.worktreeName` is empty, return `false`; otherwise call `l.deps.Worktree.HasUnmergedCommits(cook.worktreeName)` and propagate errors

This mirrors the existing two-path merge in `mergeCookWorktree` (local vs `MergeRemoteBranch`) but moves the detection upstream of the merge pipeline. The persisted-metadata fallback ensures crash recovery works when `spawn.json` is cleaned up.

**`loop/cook_completion.go`**:
- Replace `canMerge := l.canMergeStage(cook.stage)` with `canMerge, err := l.worktreeHasChanges(cook)`. On error, fail the stage (don't silently advance).

**`loop/control_review.go`**:
- Same replacement in `controlMerge()` (line 41). On error, return the error.

**`loop/loop_merge_reconcile_test.go`**:
- Rewrite `TestApprovalAutoCanMergeTrueAutoMerges` → `TestAutoMergeWithLocalChanges` — set `fakeWorktree.hasUnmergedCommits["..."] = true`
- Rewrite `TestApprovalAutoCanMergeFalseAdvances` → `TestAutoAdvanceWithoutChanges` — set `fakeWorktree.hasUnmergedCommits["..."] = false`
- Add `TestAutoMergeWithRemoteSyncResult` — local worktree has no commits, but spawn.json contains a sync result with `type: "branch"`. Verify the stage enters the merge pipeline and calls `MergeRemoteBranch`.

**`loop/sous_chef_test.go`**:
- Remove `CanMerge: false` from test task type fixtures (field will be deleted in phase 4, but stop using it here)

## Data structures

- `worktreeHasChanges` method on `*Loop` — returns `(bool, error)`, no new types

## Principles

- **prove-it-works** — test both local and remote-sync paths; test error propagation
- **fix-root-causes** — errors fail the stage instead of silently advancing
- **redesign-from-first-principles** — if we had runtime detection from day one, we'd never have added a static declaration

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./loop/...
```

- Verify "no commits" case: `hasUnmergedCommits[name] = false` → stage advances without entering merge pipeline
- Verify "has commits" case: default `true` → merge pipeline runs
- Verify "sprites remote-sync" case: local worktree has no commits, spawn.json has sync result with branch → stage enters merge pipeline and calls `MergeRemoteBranch`
- Verify error propagation: `HasUnmergedCommits` returns error → stage fails, does not advance
- Verify crash recovery: persisted `merge_branch` in orders.json Extra → `worktreeHasChanges` returns true even without spawn.json
