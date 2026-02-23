Back to [[plans/27-remote-dispatchers/overview]]

# Phase 10: Minimal sync-back for remote runtimes

**Routing:** `claude` / `claude-opus-4-6` — judgment needed for cook loop integration and error handling

## Goal

Remote sessions produce changes remotely, not in a local worktree. Without sync-back, `handleCompletion` can only skip the merge step — no code lands locally. This phase adds a minimal contract: streaming backends (Sprites) push to a branch, the cook loop fetches and merges it. Polling backends (Cursor) create PRs natively, so sync-back just records the PR URL.

## Data structures

- `SyncResult` struct — holds branch name (streaming) or PR URL (polling), merge outcome
- Extend `StreamHandle` or session metadata with `RemoteBranch string`
- Extend polling session metadata with `PullRequestURL string`

## Changes

**`dispatcher/backend.go`**
Add optional `SyncBack` interface that backends can implement:
```go
type SyncBacker interface {
    SyncBack(ctx context.Context, handle StreamHandle) (SyncResult, error)
}
```
Not all backends need this — `TmuxBackend` doesn't (local diffs merge via worktree). Check with a type assertion.

**`dispatcher/sprites_backend.go`**
Implement `SyncBacker`. After the agent session completes, the Sprite VM has local git changes. The agent should commit and push to a branch named `noodle/<session-id>`. `SyncBack` returns the branch name. The agent prompt (composed by StreamingDispatcher) should include instructions to commit and push before exiting.

**`dispatcher/cursor_backend.go`**
Implement `SyncBacker`. Cursor creates PRs natively — `SyncBack` calls `GET /v0/agents/{id}` to read the PR URL from the response and returns it. No local merge needed.

**`loop/cook.go`**
Update `handleCompletion` for remote runtimes:
1. After session completes, check if the dispatcher session carries sync-back metadata
2. For streaming remotes with a branch: `git fetch origin && git merge origin/noodle/<session-id>` into the current integration branch, then delete the remote branch
3. For polling remotes with a PR URL: store the PR URL in session metadata (`spawn.json`), mark item done without local merge
4. On merge conflict: mark session as "conflict", surface to user via queue action_needed — don't auto-resolve

**`dispatcher/streaming_session.go`**
After `backend.IsAlive` returns false and session is complete, call `SyncBacker.SyncBack` if the backend implements it. Store the result in session metadata.

## Verification

### Static
- Compiles, passes vet
- `SyncBacker` interface is optional — existing `TmuxBackend` compiles without implementing it

### Runtime
- Unit test: mock streaming backend implementing `SyncBacker` returns a branch name → cook loop calls git fetch + merge
- Unit test: mock polling backend implementing `SyncBacker` returns a PR URL → cook loop records URL, skips merge
- Unit test: backend not implementing `SyncBacker` → cook loop skips sync-back (existing tmux path)
- Unit test: git merge conflict → session marked "conflict", queue item gets action_needed
