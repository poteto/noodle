Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 8: Backend — EventResult Completion Signal

## Goal

Currently, order stage completion depends on the agent process exiting. This means orders are stuck waiting even when the agent has signaled completion via canonical events. Fix this: when the canonical event stream contains a completion signal, mark the stage as completed without waiting for process exit.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle logic change with concurrency implications, needs careful reasoning

## What already exists (do NOT reimplement)

- `monitor/claims.go` already sets `Completed = true` for both `EventResult` and `EventComplete` canonical events
- `dispatcher/process_session.go` already intercepts canonical lines live via `io.MultiWriter` + interceptor — no need for a separate file-scanning watcher
- `event/types.go` and `parse/canonical.go` already define the event types needed

## Key insight from review

`EventResult` is a **turn boundary** in Claude's control flow, not a guaranteed session completion. A session may emit multiple `EventResult` events across turns. Only the final `EventResult` before process exit (or an `EventComplete`) signals true completion. The plan must handle this — completion gating needs policy, not just "saw an EventResult".

## Changes

### Modify
- `dispatcher/process_session.go` — use the existing live canonical interceptor to detect completion. When the interceptor sees `EventComplete` (which IS terminal), signal early completion. For `EventResult`, do NOT signal early — it may be a mid-session turn boundary. Let the process exit naturally for `EventResult`-only completions.
- `loop/cook_watcher.go` — add a second signal path alongside `session.Done()`: a `CompletedEarly` channel that the interceptor can close when `EventComplete` is seen. Create `StageResult` from either signal, whichever fires first. When both fire, deduplicate (ignore the second).
- `loop/cook_completion.go` — `handleCompletion` currently reads `cook.session.Status()`. If the process is still alive when early completion fires, this returns `"running"`. Fix: `StageResult` must carry its own status (completed/failed), and `handleCompletion` must use `StageResult.Status` instead of re-deriving from session.

### Concurrency concerns
1. **Early `Done()` drops concurrency accounting** — `runtime/dispatcher_runtime.go:67` decrements active count when `Done()` closes. If we close early, the loop may dispatch a new cook while the old process still holds the worktree. Solution: don't close `Done()` early. Use a separate `CompletedEarly` channel that the loop watches, but keep `Done()` tied to actual process exit for concurrency accounting.
2. **Merge races against still-running process** — if we advance/merge while the process is still writing to the worktree, we get data races. Solution: for mergeable stages, defer the merge until `Done()` (process exit). The stage status can advance to "completed" early, but the merge step waits for actual process exit.
3. **Existing race on non-zero exit** — `terminalStatus()` may read canonical before the stream goroutine flushes final events, causing false `failed`. This is a pre-existing issue; document it but don't block on fixing it here.

## Data Structures

- `StageResult` — ensure it carries `Status` as a first-class field, not derived from session at consumption time
- `SessionHandle` — add `CompletedEarly <-chan struct{}` alongside existing `Done() <-chan struct{}`

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: `CompletedEarly` fires on `EventComplete`, does NOT fire on `EventResult` alone
- Unit test: `StageResult.Status` is used by `handleCompletion`, not `session.Status()`
- Unit test: `Done()` still fires only on process exit (concurrency accounting preserved)
- Unit test: when both `CompletedEarly` and `Done()` fire, only one `StageResult` is created

### Runtime
- Start a cook session that emits `EventComplete` — stage advances before process exits
- Start a cook session that only emits `EventResult` — stage waits for process exit as before
- Verify concurrency: early completion doesn't allow double-dispatching to the same worktree
