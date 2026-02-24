Back to [[plans/27-remote-dispatchers/overview]]

# Phase 10: Minimal sync-back for remote runtimes

**Routing:** `claude` / `claude-opus-4-6` — judgment needed for cook loop integration and error handling

## Goal

Remote sessions produce changes remotely, not in a local worktree. Without sync-back, `handleCompletion` can only skip the merge step — no code lands locally. This phase adds a minimal contract for streaming backends: the agent pushes to a branch, the cook loop fetches and merges it.

For MVP, only `SpritesBackend` implements sync-back. Cursor is a stub (Phase 6) and does not implement sync-back. PR-based sync-back (per-item status tracking, PR URL persistence, completion resolution) is deferred to a future plan when Cursor is fully implemented.

## Data structures

- `SyncResult` struct — holds branch name, result type (branch/none)
- `StreamingSyncBacker` interface — `SyncBack(ctx, sessionID string) (SyncResult, error)` for streaming backends that push branches
- `PollingSyncBacker` interface — `SyncBack(ctx, remoteID string) (SyncResult, error)` for polling backends (defined but not implemented by any backend in this plan)
- `MergeConflictError` — typed error for target-scoped merge failures

## Changes

**`dispatcher/backend.go`**
Add two optional sync-back interfaces:
```go
type StreamingSyncBacker interface {
    SyncBack(ctx context.Context, sessionID string) (SyncResult, error)
}

type PollingSyncBacker interface {
    SyncBack(ctx context.Context, remoteID string) (SyncResult, error)
}
```
Not all backends need these — `TmuxBackend` doesn't (local diffs merge via worktree). Check with type assertions. `PollingSyncBacker` is defined for future use but nothing implements it in this plan.

**`dispatcher/sprites_backend.go`**
Implement `StreamingSyncBacker`. After the agent session completes, the Sprite VM has local git changes. The agent should commit and push to a branch named `noodle/<session-id>`. `SyncBack` returns the branch name. The agent prompt (composed by StreamingDispatcher) should include instructions to commit and push before exiting.

**`dispatcher/streaming_session.go`**
After session completes (EOF on stream, backend not alive), check if backend implements `StreamingSyncBacker`. If so, call `SyncBack(ctx, sessionID)` and store the `SyncResult` in session metadata (`spawn.json`). This happens before the session signals done.

**`loop/cook.go`**
Thread `sessionID` through merge APIs. Currently `mergeCook(ctx, item, worktreeName)` has no way to find session metadata. Change signature to `mergeCook(ctx, item, worktreeName, sessionID)`. Update all call sites: `handleCompletion` (has `cook.session.ID()`), `pendingReviewCook` (already stores sessionID), and `control.go` merge path.

Update `mergeCook` — sync-back artifacts flow through the existing completion pipeline:
1. Quality gate and pending-approval checks run first (unchanged)
2. Read `SyncResult` from `spawn.json` using the threaded `sessionID`
3. If result type is `branch`: delegate to `WorktreeManager.MergeRemoteBranch(branchName)` (see interface changes below).

**`loop/types.go`**
Add `MergeRemoteBranch(branch string) error` to the `WorktreeManager` interface (line 87-91). This extends the existing interface alongside `Create`, `Merge`, and `Cleanup`.

**`loop/defaults.go`**
Add `func (noOpWorktree) MergeRemoteBranch(string) error { return nil }` to the no-op stub (line 16-20).

**`worktree/` (app implementation)**
Implement `MergeRemoteBranch` on the worktree app. Reuses existing merge safeguards (merge lock via `.worktrees/.merge-lock`, integration branch cleanliness check, rebase discipline) but operates on a remote branch: `git fetch origin <branch>`, `git merge origin/<branch>`, delete remote branch on success (`git push origin --delete <branch>`).
4. If no sync result (tmux): existing worktree merge path (unchanged)
5. On merge conflict: return `MergeConflictError` (see below).

**`loop/loop.go`**
Completion errors from `collectCompleted` currently flow into `handleRuntimeIssue` (line 185-186). Add a typed `MergeConflictError` that `mergeCook` returns on git conflicts. In `collectCompleted`, check for this error type with `errors.As` and handle it as a **non-retryable** target-scoped failure: skip the queue item with a failure reason (same as `skipQueueItem` but with an error message), do not requeue for retry (the conflict is deterministic — retrying would loop), and do not pass to `handleRuntimeIssue`. The user resolves the conflict manually and re-queues if needed.

## Verification

### Static
- Compiles, passes vet
- Both sync-back interfaces are optional — existing `TmuxBackend` compiles without implementing either
- `mergeCook` signature change compiles across all call sites

### Runtime
- Unit test: mock streaming backend implementing `StreamingSyncBacker` returns a branch → `mergeCook` calls `WorktreeManager.MergeRemoteBranch`
- Unit test: backend not implementing sync-back → `mergeCook` uses existing worktree merge path
- Unit test: git merge conflict → `MergeConflictError` returned, item skipped (not retried), does not trigger runtime-repair
- Unit test: `mergeCook` reads spawn.json via threaded sessionID
