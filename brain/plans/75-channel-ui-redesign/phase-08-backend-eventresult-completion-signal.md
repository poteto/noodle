Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 8: Backend — Early Completion Signal

## Goal

Order stage completion currently depends on the agent process exiting. For providers that emit a terminal canonical event (`EventComplete`), the loop should advance the stage without waiting for process exit. Additionally, `handleCompletion` should use `StageResult.Status` instead of re-deriving status from the session — fixing a latent bug where the session reports `"running"` if queried before exit.

## Scope limitation

**This phase only benefits Codex sessions.** Claude emits `EventResult` (a turn boundary that may repeat across multi-turn sessions), never `EventComplete`. Only Codex emits `EventComplete` (via `parse/codex.go:94,189`), which IS terminal. Treating `EventResult` as a completion signal would cause premature advancement mid-conversation — that's wrong. Claude sessions continue to use process exit as their completion signal, which is correct behavior.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle logic change with concurrency implications, needs careful reasoning

## What already exists (do NOT reimplement)

- `monitor/claims.go` sets `Completed = true` for both `EventResult` and `EventComplete` — this is claim-level tracking, separate from stage advancement
- `dispatcher/process_session.go` already intercepts canonical lines live via `io.MultiWriter` + interceptor
- `event/types.go` and `parse/canonical.go` define the event types
- `parse/codex.go:94,189` emits `EventComplete` for Codex `turn.completed` and `task_complete`
- `parse/claude.go:103` emits `EventResult` for Claude `type: "result"` — this is NOT terminal

## Changes

### Modify
- `dispatcher/process_session.go` — when the live interceptor sees `EventComplete`, close a `CompletedEarly` channel. Do NOT close it on `EventResult`. This is a small addition to the existing interceptor logic.
- `loop/cook_watcher.go` — watch `CompletedEarly` alongside `session.Done()` via select. Whichever fires first creates the `StageResult`. Deduplicate: ignore the second signal. **The stage stays `active` until the loop processes the result** — do NOT mark it `completed` in the watcher.
- `loop/cook_completion.go` — `handleCompletion` uses `StageResult.Status` instead of `cook.session.Status()`. Thread `StageResult` through the adopted-session path (`cook_completion.go:310`) as well.

### Concurrency constraints
1. **`Done()` stays tied to process exit.** Concurrency accounting (`runtime/dispatcher_runtime.go:67`) decrements on `Done()`. The cook stays in the active map until `Done()`. `CompletedEarly` is a separate signal — it does not affect active count, busy targets, or dispatch capacity.
2. **For mergeable stages: defer merge until `Done()`.** Even if `CompletedEarly` fires, the process may still be writing to the worktree. The loop can mark the stage result as ready but must not merge until process exit.
3. **For non-mergeable stages: advance immediately on `CompletedEarly`.** No worktree to protect. The old process can keep running — its output goes to stdout which doesn't affect state.
4. **Cook stays in active map.** `applyStageResult` removes the cook (`cook_completion.go:78`). When early completion fires, the cook must remain visible for steer/stop/shutdown until `Done()`. Split result processing: advance the stage immediately, defer cook cleanup to `Done()`.
5. **Pre-existing race: `terminalStatus()` returns `"completed"` on first `EventResult`/`EventComplete`, masking later errors.** Document this but don't fix here — it's not introduced by this change.

### What NOT to change
- Don't close `Done()` early — breaks concurrency accounting
- Don't trigger on `EventResult` — it's a turn boundary, not terminal
- Don't change claims.go — it correctly tracks both event types for claim-level completion

## Data Structures

- `StageResult` — ensure `Status` is a first-class field, not derived from session at consumption time
- `SessionHandle` — add `CompletedEarly() <-chan struct{}` alongside existing `Done() <-chan struct{}`

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: `CompletedEarly` fires on `EventComplete`, does NOT fire on `EventResult` alone
- Unit test: `StageResult.Status` is used by `handleCompletion`, not `session.Status()`
- Unit test: `Done()` still fires only on process exit (concurrency accounting preserved)
- Unit test: when both `CompletedEarly` and `Done()` fire, only one `StageResult` is processed
- Unit test: cook remains in active map after `CompletedEarly` until `Done()`

### Runtime
- Start a Codex cook session — stage advances before process exits
- Start a Claude cook session — stage waits for process exit as before (no behavior change)
- Verify concurrency: early completion doesn't allow double-dispatching to the same worktree
