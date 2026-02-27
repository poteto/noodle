Back to [[plans/48-live-agent-steering/overview]]

# Phase 5 — Session Lifecycle Migration

## Goal

Replace tmux-based session monitoring, reconciliation, and shutdown with PID-based equivalents. After this phase, tmux is no longer required at runtime.

## Changes

**Modify: `loop/reconcile.go`** (~80 lines rewritten)

Replace `tmux list-sessions` enumeration with PID directory scan:

- On startup, scan `sessions/*/process.json` for PID files
- For each PID: check liveness via `os.FindProcess(pid)` + `Signal(0)`
- Live processes with matching session metadata → adopt (create `processSession` wrapper)
- Dead processes → clean up metadata, mark as failed in status
- Unknown processes (no metadata) → ignore (not ours)

Remove `killTmuxSession()` helper — replaced by `ProcessHandle.Kill()`.

**Modify: `monitor/observer.go`** (~30 lines)

Replace `TmuxObserver` with `PidObserver`:

- Read PID from `process.json`
- Check liveness via `Signal(0)` instead of `tmux has-session`
- Same `Observer` interface — `CompositeObserver` unchanged

**Modify: `loop/loop.go`** (~15 lines)

`Shutdown()`:
- Iterate all active sessions, call `session.Kill()` (which now does SIGTERM → SIGKILL via `ProcessHandle`)
- No more `tmux kill-session` calls
- Wait for all `Done()` channels to close (with timeout)

**Modify: `dispatcher/factory.go`**

Register `ProcessDispatcher` as the handler for runtime `"tmux"` (backward compatible config key — rename later in cleanup). Remove `TmuxDispatcher` registration.

## Data Structures

- `PidObserver` — replaces `TmuxObserver`, same interface
- `process.json` — `{pid: int, session_id: string, started_at: string}` written by Phase 1

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` — mechanical migration with clear before/after.

## Verification

### Static
- `go build ./...`
- `go vet ./...`
- All existing tests pass (interface contracts unchanged)

### Runtime
- Integration test: start noodle, spawn cook, kill noodle process, restart → verify cook is detected as dead (not adopted as orphan)
- Integration test: spawn cook, verify `PidObserver` reports liveness correctly
- Integration test: `Shutdown()` terminates all child processes within timeout
- Manual: `noodle start`, spawn multiple cooks, `Ctrl-C`, verify all child processes cleaned up (no orphans in `ps`)
