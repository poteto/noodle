Back to [[plans/48-live-agent-steering/overview]]

# Phase 1 — Process Manager

## Goal

Replace tmux as the process host with direct child process management. This is the foundation — every subsequent phase depends on Noodle owning the provider process and its stdin/stdout pipes.

## Changes

**New file: `dispatcher/process.go`**

A `ProcessHandle` type that wraps `os/exec.Cmd` with the lifecycle primitives the rest of the codebase needs:

- Spawn a child process with `SysProcAttr{Setpgid: true}` for process group isolation
- Expose `Stdin() io.WriteCloser`, `Stdout() io.ReadCloser`, `Stderr() io.ReadCloser`
- `Done() <-chan struct{}` — closed when process exits (backed by a goroutine calling `Cmd.Wait()`)
- `Kill()` — SIGTERM → 5s timeout → SIGKILL on the process group
- `ExitCode() (int, bool)` — returns exit code after done
- Write PID + metadata to `sessions/<id>/process.json` on spawn for crash recovery

This replaces what `tmux new-session` + `tmux has-session` + `tmux kill-session` provided.

**New file: `dispatcher/process_dispatcher.go`**

A new `Dispatcher` implementation that uses `ProcessHandle` instead of tmux:

- Builds the provider command (reuse logic from `tmux_command.go`)
- Spawns via `ProcessHandle` instead of `tmux new-session`
- Pipes provider stdout through an in-process stamp processor (like sprites already does via `canonicalEventInterceptor`)
- Returns a `processSession` that monitors the child process

**New file: `dispatcher/process_session.go`**

A `processSession` implementing `Session`:

- Wraps `ProcessHandle`
- Runs stamp processor on provider stdout in a goroutine
- Publishes `SessionEvent` stream from canonical events (same as `tmuxSession.monitorCanonicalEvents` but reading from a pipe, not polling a file)
- `Kill()` delegates to `ProcessHandle.Kill()`
- Writes heartbeat, stamped NDJSON, and canonical NDJSON to session directory (same paths as before — existing file consumers don't break)

## Data Structures

- `ProcessHandle` — wraps `*exec.Cmd`, owns pipes and lifecycle
- `processSession` — implements `Session`, wraps `ProcessHandle` + stamp processor
- `ProcessDispatcher` — implements `Dispatcher`, replaces `TmuxDispatcher`

## Routing

Provider: `claude`, Model: `claude-opus-4-6` — architectural foundation, concurrency design.

## Verification

### Static
- `go build ./dispatcher/...`
- `go vet ./dispatcher/...`

### Runtime
- Unit test: spawn a simple command (`echo hello`), read stdout, verify `Done()` fires
- Unit test: `Kill()` sends SIGTERM, process exits, verify exit code
- Unit test: PID file written and contains correct PID
- Integration test: spawn Claude with current flags (file-redirection still, pipe transport comes in Phase 3), verify events stream through in-process stamp
