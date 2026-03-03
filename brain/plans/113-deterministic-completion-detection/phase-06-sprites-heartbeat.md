Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 6: Sprites Heartbeat Writing

## Goal

Fix the monitor gap where sprites sessions don't write heartbeats, making `HeartbeatObserver` unable to check their liveness.

## Changes

**`dispatcher/sprites_session.go`** (or `session_base.go`):
- Add heartbeat writing to the canonical event processing path in `sessionBase` — since both session types share `sessionBase`, heartbeat writing can be unified there
- Currently `processSession` writes heartbeats in its own `writeHeartbeat()` method called from session-specific code. Move this to `sessionBase.consumeCanonicalLine()` so it fires for every session type on every canonical event.

**`dispatcher/process_session.go`**:
- Remove the session-specific `writeHeartbeat()` calls (now handled by sessionBase)

## Data Structures

- No new types — reuses existing `heartbeat.json` format

## Design Notes

The heartbeat mechanism is simple: write `{timestamp, ttl_seconds}` to `heartbeat.json` on every canonical event. By moving this to sessionBase, sprites sessions automatically get heartbeat support without any sprites-specific code.

This also simplifies processSession — one less responsibility in the session-type layer.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Unit test: verify sprites session writes `heartbeat.json` after processing events
- Unit test: verify `HeartbeatObserver` correctly reports sprites session as alive
- Integration test: `CompositeObserver` routes to heartbeat observer for sprites runtime and gets correct liveness
- `go test ./dispatcher/... ./monitor/... -race`
