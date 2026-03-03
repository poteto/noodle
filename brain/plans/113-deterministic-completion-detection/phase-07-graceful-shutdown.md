Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 7: Graceful Shutdown

## Goal

Replace the hard-coded 2-second shutdown deadline with a configurable, stage_yield-aware graceful shutdown that gives agents meaningful time to finish work.

## Changes

**`loop/loop.go`**:
- Replace `shutdownDeadline = 2 * time.Second` constant with a configurable value from loop config (default: 10 seconds)
- Update `Shutdown()` to be stage_yield-aware: after SIGTERM, poll for stage_yield events on active sessions. If a session emits stage_yield, its work is safe — SIGKILL it immediately (don't wait for the full deadline). Sessions that don't yield within the deadline get SIGKILL as before.
- Update `shutdownAndDrain()` accordingly

**`loop/config.go`** (or wherever loop config lives):
- Add `ShutdownDeadline time.Duration` field with 10-second default

**`loop/loop.go`** — new helper `waitForYieldsOrDeadline()`:
- Takes active sessions and deadline duration
- For each session, checks `readStageYield()` in a polling loop (100ms interval)
- Returns early per-session when yield is detected
- Returns on deadline for remaining sessions

## Data Structures

- `ShutdownDeadline` field on loop config

## Design Notes

The stage_yield-aware shutdown leverages an existing mechanism: agents already emit `stage_yield` to declare their deliverable is complete. During shutdown, this signal means "my work is committed, you can kill me safely." Agents that emit `stage_yield` early (before shutdown) are already safe — the loop can kill them immediately after SIGTERM.

10-second default is generous enough for an agent to write final files and commit, while still being bounded. The configurable value lets operators tune for their environment.

The `stop-all` and `kill` commands remain immediate SIGKILL — they're explicit user requests for immediate termination.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (shutdown timing is a design decision)

## Verification

- Unit test: shutdown with yielded session → immediate kill after SIGTERM (no waiting)
- Unit test: shutdown without yield → waits full deadline, then SIGKILL
- Unit test: configurable deadline is respected
- Integration test: spawn session, emit stage_yield via mock, trigger shutdown, verify fast kill
- `go test ./loop/... -race`
