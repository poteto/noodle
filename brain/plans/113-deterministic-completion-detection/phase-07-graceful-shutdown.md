Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 7: Graceful Shutdown

## Goal

Replace the hard-coded 2-second shutdown deadline with a configurable, stage_yield-aware graceful shutdown that gives agents meaningful time to finish work.

## Changes

**`loop/loop.go`**:
- Change `shutdownDeadline` constant from `2 * time.Second` to `10 * time.Second` (fixed, not configurable — no concrete second use case for tunability)
- Update `Shutdown()` to be stage_yield-aware: after SIGTERM, check each active session for an existing `stage_yield` event. Sessions that already yielded are safe — SIGKILL immediately. Remaining sessions wait for the deadline then get SIGKILL.
- Update `shutdownAndDrain()` accordingly

**`loop/loop.go`** — update shutdown logic:
- After SIGTERM, do a single `readStageYield()` check per session (not polling — one read, not a loop). Sessions with yield get immediate SIGKILL. Remaining sessions get the full 10-second deadline.
- No file polling during shutdown — the one-shot check catches yields emitted before shutdown. Agents that emit yield after SIGTERM are still safe because the 10-second window is generous.

## Data Structures

- No new types or config fields — just a constant change

## Design Notes

The stage_yield-aware shutdown leverages an existing mechanism: agents already emit `stage_yield` to declare their deliverable is complete. During shutdown, this signal means "my work is committed, you can kill me safely." Agents that emit `stage_yield` early (before shutdown) are already safe — the loop can kill them immediately after SIGTERM.

10 seconds is generous enough for an agent to write final files and commit, while still being bounded. No config knob — per subtract-before-you-add, add tunability only when a concrete second use case emerges.

The `stop-all` and `kill` commands remain immediate SIGKILL — they're explicit user requests for immediate termination.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (shutdown timing is a design decision)

## Verification

- Unit test: shutdown with yielded session → immediate kill after SIGTERM (no waiting)
- Unit test: shutdown without yield → waits full 10s deadline, then SIGKILL
- Integration test: spawn session, emit stage_yield via mock, trigger shutdown, verify fast kill
- `go test ./loop/... -race`
