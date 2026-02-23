Back to [[plans/27-remote-dispatchers/overview]]

# Phase 10: Minimal sync-back for remote runtimes

**Routing:** `claude` / `claude-opus-4-6` — judgment needed for cook loop integration and error handling

## Goal

Remote sessions produce changes remotely, not in a local worktree. Without sync-back, `handleCompletion` can only skip the merge step — no code lands locally. This phase adds a minimal contract: streaming backends (Sprites) push to a branch, the cook loop fetches and merges it. Polling backends (Cursor) create PRs natively, so sync-back just records the PR URL.

## Data structures

- `SyncResult` struct — holds branch name or PR URL, result type (branch/pr/none)
- `StreamingSyncBacker` interface — `SyncBack(ctx, sessionID string) (SyncResult, error)` for streaming backends that push branches
- `PollingSyncBacker` interface — `SyncBack(ctx, remoteID string) (SyncResult, error)` for polling backends that create PRs

Two separate interfaces because the inputs differ: streaming backends identify by session ID (used to name the remote branch), polling backends identify by remote agent ID (used to query the PR URL from the API).

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
Not all backends need these — `TmuxBackend` doesn't (local diffs merge via worktree). Check with type assertions.

**`dispatcher/sprites_backend.go`**
Implement `StreamingSyncBacker`. After the agent session completes, the Sprite VM has local git changes. The agent should commit and push to a branch named `noodle/<session-id>`. `SyncBack` returns the branch name. The agent prompt (composed by StreamingDispatcher) should include instructions to commit and push before exiting.

**`dispatcher/cursor_backend.go`**
Implement `PollingSyncBacker`. Cursor creates PRs natively — `SyncBack` calls `GET /v0/agents/{id}` to read the PR URL from the response and returns it.

**`dispatcher/streaming_session.go`**
After session completes (EOF on stream, backend not alive), check if backend implements `StreamingSyncBacker`. If so, call `SyncBack(ctx, sessionID)` and store the `SyncResult` in session metadata (`spawn.json`). This happens before the session signals done.

**`dispatcher/polling_session.go`**
After terminal status reached, check if backend implements `PollingSyncBacker`. If so, call `SyncBack(ctx, remoteID)` and store the `SyncResult` in session metadata.

**`loop/cook.go`**
Thread `sessionID` through merge APIs. Currently `mergeCook(ctx, item, worktreeName)` has no way to find session metadata. Change signature to `mergeCook(ctx, item, worktreeName, sessionID)`. Update all call sites: `handleCompletion` (has `cook.session.ID()`), `pendingReviewCook` (already stores sessionID), and `control.go` merge path.

Update `mergeCook` — sync-back artifacts flow through the existing completion pipeline:
1. Quality gate and pending-approval checks run first (unchanged)
2. Read `SyncResult` from `spawn.json` using the threaded `sessionID`
3. If result type is `branch`: `git fetch origin && git merge origin/noodle/<session-id>` into the integration branch, then delete the remote branch. This replaces the worktree merge for remote sessions.
4. If result type is `pr`: mark item as `action_needed` with the PR URL — item stays in queue until the PR is merged. Do not mark done.
5. If no sync result (tmux): existing worktree merge path (unchanged)
6. On merge conflict: return a target-scoped error that the loop handles as a cook failure (same path as quality rejection / retry), not as a runtime-repair trigger. User resolves manually.

**PR completion tracking:** Items with `action_needed` and a PR URL stay in the queue. A future enhancement can poll PR merge status or the user can manually mark done. For now, the prioritize agent sees the action_needed flag and skips the item.

## Verification

### Static
- Compiles, passes vet
- Both sync-back interfaces are optional — existing `TmuxBackend` compiles without implementing either
- `mergeCook` signature change compiles across all call sites

### Runtime
- Unit test: mock streaming backend implementing `StreamingSyncBacker` returns a branch → `mergeCook` calls git fetch + merge
- Unit test: mock polling backend implementing `PollingSyncBacker` returns a PR URL → item marked `action_needed`, not done
- Unit test: backend not implementing sync-back → `mergeCook` uses existing worktree merge path
- Unit test: git merge conflict → cook fails with target-scoped error, does not trigger runtime-repair
- Unit test: `mergeCook` reads spawn.json via threaded sessionID
