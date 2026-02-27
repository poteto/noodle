Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 5: PollingDispatcher

**Routing:** `codex` / `gpt-5.3-codex` — wiring against established patterns

## Goal

Implement `PollingDispatcher` — a `Dispatcher` that uses a `PollingBackend` to launch and manage sessions. Follows the same pattern as `SpritesDispatcher`: push worktree branch to GitHub, launch the remote agent, return a session. The session handles polling and completion.

## Data structures

- `PollingDispatcher` struct — holds `PollingBackend`, runtime dir, poll interval, project dir
- `PollingDispatcherConfig` struct — constructor input

## Changes

**`dispatcher/polling_dispatcher.go` (new)**
- `NewPollingDispatcher(cfg PollingDispatcherConfig) *PollingDispatcher`
- `Dispatch(ctx, DispatchRequest) (Session, error)`:
  1. Validate request
  2. Generate session ID, create session directory (`runtimeDir/sessions/{id}/`)
  3. Push worktree branch to GitHub — reuse `pushWorktreeBranch(ctx, worktreePath, branch)` (already package-level in `sprites_dispatcher.go`)
  4. Build `PollLaunchConfig` from request (Prompt, Repository from config, Model, Branch)
  5. Call `backend.Launch(ctx, config)` → remote ID
  6. Write `spawn.json` via `writeDispatchMetadata`
  7. Create `pollingSession` with remote ID, interval, runtime dir
  8. Call `session.start(ctx)` — launches poll goroutine
  9. Return session
- Not steerable — `pollingSession` returns `NoopController()`
- `var _ Dispatcher = (*PollingDispatcher)(nil)` compile check

**`dispatcher/polling_dispatcher_test.go` (new)**
- Test with mock `PollingBackend` (skip real git push — test the dispatch flow):
  - Dispatch creates session directory, writes spawn.json
  - Dispatch calls backend.Launch with correct config
  - Returned session is running and polling
  - Dispatch with invalid request → error
  - Dispatch with backend.Launch error → error, session directory cleaned up

## Verification

### Static
- `go vet ./dispatcher/...`
- `PollingDispatcher` satisfies `Dispatcher` interface

### Runtime
- `go test ./dispatcher/... -run TestPollingDispatcher -race`
