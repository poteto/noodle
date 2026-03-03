Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 7: Graceful Shutdown

## Goal

Raise the hard-coded 2-second shutdown deadline to 10 seconds, giving agents meaningful time to finish work.

## Changes

**`loop/loop.go`**:
- Change `shutdownDeadline` constant from `2 * time.Second` to `10 * time.Second`
- Update `shutdownAndDrain()` watcher timeout accordingly (`shutdownDeadline + 1 second` → 11 seconds)

## Data Structures

- No new types or config fields — just a constant change

## Design Notes

10 seconds is generous enough for an agent to write final files, commit, and emit `stage_yield`, while still being bounded. No yield-based early kill branching — it adds complexity and can race with sprites sync-back. The uniform SIGTERM → 10s → SIGKILL flow is simpler and sufficient.

No config knob — per subtract-before-you-add, add tunability only when a concrete second use case emerges.

The `stop-all` and `kill` commands remain immediate SIGKILL — they're explicit user requests for immediate termination.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex` (simple constant change)

## Verification

- Unit test: shutdown waits full 10s deadline, then SIGKILL
- Verify `shutdownAndDrain` watcher timeout is `shutdownDeadline + 1 second`
- `go test ./loop/... -race`
