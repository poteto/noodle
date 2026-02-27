Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 5: PollingDispatcher

**Routing:** `codex` / `gpt-5.3-codex` — wiring against established patterns

## Goal

Implement `PollingDispatcher` — a `Dispatcher` that uses a `PollingBackend` to launch and manage sessions. Follows the same pattern as `SpritesDispatcher`: push worktree branch to GitHub, launch the remote agent, return a session. Owns the `pollingRegistry` for session lookup. Handles rollback if local steps fail after remote launch.

## Data structures

- `PollingDispatcher` struct — holds `PollingBackend`, runtime dir, poll interval, project dir, `*pollingRegistry`
- `PollingDispatcherConfig` struct — constructor input

## Changes

**`dispatcher/polling_dispatcher.go` (new)**
- `NewPollingDispatcher(cfg PollingDispatcherConfig) *PollingDispatcher` — creates internal `pollingRegistry`
- `Registry() SessionNotifier` — exposes the registry for webhook wiring
- `Dispatch(ctx, DispatchRequest) (Session, error)`:
  1. Validate request
  2. Generate session ID, create session directory (`runtimeDir/sessions/{id}/`)
  3. Push worktree branch to GitHub — reuse `pushWorktreeBranch(ctx, worktreePath, branch)` (already package-level in `sprites_dispatcher.go`). If push fails, clean up session dir and return error.
  4. Build `PollLaunchConfig` from request (Prompt, Repository from config, Model, Branch)
  5. Call `backend.Launch(ctx, config)` → `LaunchResult{RemoteID, TargetBranch}`
  6. Write `spawn.json` via `writeDispatchMetadata` — **include `remote_id` and `target_branch` in metadata** for crash recovery
  7. If spawn.json write fails after launch: call `backend.Stop` + `backend.Delete` best-effort (rollback), clean session dir, return error
  8. Create `pollingSession` with `LaunchResult`, interval, runtime dir, event writer
  9. Register session in `pollingRegistry`
  10. Call `session.start(ctx)` — launches poll goroutine
  11. Set up goroutine to unregister from registry when session reaches terminal state (`<-session.Done()` → `registry.Unregister(remoteID)`)
  12. Return session
- Not steerable — `pollingSession` returns `NoopController()`
- `var _ Dispatcher = (*PollingDispatcher)(nil)` compile check

**`dispatcher/polling_dispatcher_test.go` (new)**
- Test with mock `PollingBackend` (skip real git push — inject pushWorktreeBranch):
  - Dispatch creates session directory, writes spawn.json with remote_id and target_branch
  - Dispatch calls backend.Launch with correct config
  - Returned session is running and polling
  - Session registered in registry, unregistered on completion
  - Dispatch with invalid request → error
  - Dispatch with backend.Launch error → error, session directory cleaned up
  - Dispatch with post-launch metadata write failure → backend.Stop/Delete called, session dir cleaned
  - Push failure → error returned, no launch attempted

## Verification

### Static
- `go vet ./dispatcher/...`
- `PollingDispatcher` satisfies `Dispatcher` interface

### Runtime
- `go test ./dispatcher/... -run TestPollingDispatcher -race`
