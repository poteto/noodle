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

The heartbeat mechanism writes `{timestamp, ttl_seconds}` to `heartbeat.json`. By moving this to sessionBase, sprites sessions automatically get heartbeat support without any sprites-specific code.

**Throttling:** Write heartbeat at most once per 5 seconds to avoid write amplification during streaming deltas. Track `lastHeartbeatTime` in sessionBase and skip writes within the throttle window.

**No periodic timer** — a single event-driven writer avoids concurrent write corruption (no serialization needed). If no events arrive for 30s, the heartbeat expires and the session appears dead. This is correct: a session producing no canonical events for 30 seconds is genuinely stuck. The monitor/repair layer handles this case.

**Initial heartbeat:** Write one heartbeat immediately on session start (in the constructor or `start()` method) so that sessions are alive from the moment they're created, before any events arrive.

**Fix observer caching bug:** `CompositeObserver.observerForSession()` caches the observer type on first read. If `spawn.json` isn't written yet (sprites setup in progress), it defaults to the local PID observer and caches permanently. Fix: either invalidate the cache when `spawn.json` appears, or don't cache (re-read runtime type each monitor pass — it's a cheap `os.ReadFile`).

This also simplifies processSession — one less responsibility in the session-type layer.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Unit test: verify sprites session writes `heartbeat.json` after processing events
- Unit test: verify `HeartbeatObserver` correctly reports sprites session as alive
- Integration test: `CompositeObserver` routes to heartbeat observer for sprites runtime and gets correct liveness
- `go test ./dispatcher/... ./monitor/... -race`
