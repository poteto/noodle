Back to [[plans/48-live-agent-steering/overview]]

# Phase 6 — Steer Rewrite

## Goal

Replace the kill-and-respawn steer mechanism with interrupt + redirect using the `AgentController`. This is the payoff phase — after this, steering preserves full agent context.

## Changes

**Modify: `loop/control.go`**

Rewrite `steer(target, prompt)`:

New flow:
1. Get controller: `cook.session.Controller()`
2. If `controller.Steerable()`:
   a. `controller.Interrupt(ctx)` — stop current turn (with timeout)
   b. If interrupt succeeds: `controller.SendMessage(ctx, prompt)` — send new direction
   c. If interrupt fails (timeout): fall back to kill + respawn with resume context (existing behavior)
   d. Done. No kill, no respawn when interrupt succeeds.
3. If not steerable (noop controller):
   a. Fall back to kill + respawn with resume context (existing behavior unchanged)

**Steering must not block the main loop.** `processControlCommands()` runs inline. Solutions:
- Run steer in a goroutine: `go l.steerAsync(target, prompt)`
- Per-session steering mutex prevents concurrent steers to the same session
- The control ack is sent immediately ("steering initiated"), not after completion
- Steer completion/failure is logged as a session event

**`request-changes` stays as-is.** It acts on `pendingReview` (session already completed) and spawns a new cook. There is no live session to steer. If we want to send feedback to active sessions in the future, that's a separate `feedback` control command — not this plan.

**Keep `buildSteerResumeContext()` as fallback.** It's used when:
- Controller is noop (sprites, adopted sessions)
- Interrupt times out (process may be hung)
- Retry after process death (`retryCook`)

## Data Structures

No new types. `activeCook` already has the session with a controller.

## Routing

Provider: `claude`, Model: `claude-opus-4-6` — judgment call on fallback behavior, timeout values, edge cases.

## Verification

### Static
- `go build ./...`
- `go vet ./...`

### Runtime
- Integration test: steer a live Claude session, verify PID unchanged (same process)
- Integration test: steer a live Codex session, verify `turn/steer` sent (not kill+respawn)
- Integration test: steer with noop controller → falls back to kill+respawn
- Integration test: interrupt timeout → falls back to kill+respawn
- Integration test: two rapid steers to same session → serialized, both succeed
- Integration test: steer doesn't block the main loop (other cooks still monitored during steer)
- Manual: `noodle start`, spawn cook, steer via web UI, verify agent references pre-steer context in its response
