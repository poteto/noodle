Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 7: Graceful Shutdown

## Goal

Raise the current 2-second shutdown deadline to 10 seconds, giving agents meaningful time to finish work.

## Changes

**`loop/loop.go`**:
- Change `shutdownDeadline` constant from `2 * time.Second` to `10 * time.Second`
- Update `shutdownAndDrain()` watcher timeout accordingly (`shutdownDeadline + 1 second` → 11 seconds)
- Keep the existing serialized shutdown sequence (terminate -> wait with deadline -> force kill -> wait with deadline) and only change timing bounds

## Data Structures

- No new types or config fields — just a constant change

## Design Notes

10 seconds is generous enough for an agent to write final files, commit, and emit `stage_yield`, while still being bounded. No yield-based early kill branching — it adds complexity and can race with sprites sync-back. The serialized terminate/wait/escalate flow remains the same; only the deadline changes.

No config knob — per subtract-before-you-add, add tunability only when a concrete second use case emerges.

The `stop-all` and `kill` commands remain immediate SIGKILL — they're explicit user requests for immediate termination.

## Routing

- Provider: `codex`, Model: `gpt-5.4` (simple constant change)

## Verification

- Unit test: shutdown waits full 10s terminate deadline before escalating to force kill
- Verify `shutdownAndDrain` watcher timeout is `shutdownDeadline + 1 second`
- `go test ./loop/... -race`
