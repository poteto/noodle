Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 2 — Replace `canMergeStage` with runtime detection

## Goal

Swap the static `canMergeStage()` registry lookup for a runtime `worktreeHasChanges()` check. After this phase, merge decisions are based on actual worktree state, not frontmatter declarations.

## Changes

**`loop/cook_merge.go`**:
- Delete `canMergeStage()` method (lines 101-110)
- Add `worktreeHasChanges(cook *cookHandle) bool` — returns `false` if `cook.worktreeName` is empty, otherwise calls `l.deps.Worktree.HasUnmergedCommits(cook.worktreeName)`

**`loop/cook_completion.go`**:
- Replace `canMerge := l.canMergeStage(cook.stage)` with `canMerge := l.worktreeHasChanges(cook)`

**`loop/control_review.go`**:
- Same replacement in `controlMerge()` (line 41)

**`loop/loop_merge_reconcile_test.go`**:
- Rewrite `TestApprovalAutoCanMergeTrueAutoMerges` — set `fakeWorktree.hasUnmergedCommits["..."] = true`
- Rewrite `TestApprovalAutoCanMergeFalseAdvances` — set `fakeWorktree.hasUnmergedCommits["..."] = false` to simulate no changes
- Update test names to reflect new semantics (e.g., `TestAutoMergeWithChanges`, `TestAutoAdvanceWithoutChanges`)

**`loop/sous_chef_test.go`**:
- Remove `CanMerge: false` from test task type fixtures (field will be deleted in phase 3, but stop using it here)

## Data structures

- `worktreeHasChanges` method on `*Loop` — no new types

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
