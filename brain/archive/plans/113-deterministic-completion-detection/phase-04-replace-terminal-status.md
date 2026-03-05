Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 4: Replace terminalStatus in processSession

## Goal

Replace the `terminalStatus()` heuristic in `processSession.waitForExit()` with the CompletionTracker's `Resolve()` output. This eliminates post-hoc file scanning for local sessions.

## Changes

**`dispatcher/process_session.go`**:
- `waitForExit()`: instead of calling `terminalStatus(exitCode)` then `markDone()`, call `s.resolveAndMarkDone(exitCode, ctx.Err() != nil)` (from phase 3) which waits for stream completion, resolves the tracker, and marks done
- Remove `terminalStatus()` method entirely
- Remove the `readCanonicalEvents()` call from the exit path (the tracker already has the state)

**`dispatcher/session_helpers.go`**:
- Keep `readCanonicalEvents()` for now — it's still used by monitor claims and stage_yield reading. Will be evaluated in phase 8.

## Data Structures

- No new types

## Design Notes

The `markDone()` call now receives the status from `SessionOutcome.Status.String()` instead of from the heuristic. The outcome is also stored on the session for callers that use `session.Outcome()`.

The `stageResultStatus()` function in `loop/cook_watcher.go` continues to map string status → `StageResultStatus`. No loop changes needed yet.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Existing `TestProcessSessionTerminalStatus*` tests should be migrated to test `CompletionTracker.Resolve()` instead (they now test the tracker, not the session method)
- New integration test: spawn a real `processSession` with a mock command, verify `Outcome()` after exit
- `go test ./dispatcher/... -race`
- Manual: run a Claude session, verify correct status on clean exit and Ctrl+C
