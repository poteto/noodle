Back to [[plans/27-remote-dispatchers/overview]]

# Phase 7: Monitor observer abstraction

**Routing:** `codex` / `gpt-5.3-codex` — mechanical refactor with clear target

## Goal

The monitor uses `TmuxObserver` to check if sessions are alive (`tmux has-session`). Extract an `Observer` interface so remote sessions can report liveness through their own mechanism. For streaming sessions, liveness comes from process state. For polling sessions, liveness comes from the last polled status.

## Data structures

- `Observer` interface — `Observe(sessionID) → (Observation, error)` (already exists as a concrete type)
- `HeartbeatObserver` — file-based observer for remote sessions. Checks a heartbeat file written by dispatcher session goroutines. No dependency on in-memory session objects.
- `CompositeObserver` — reads runtime from `spawn.json` per session and dispatches to `TmuxObserver` or `HeartbeatObserver`. Handles mixed local+remote sessions in a single monitor instance.

## Changes

**`monitor/observer.go`**
Extract `Observer` interface from `TmuxObserver`. The `Observe` method signature stays the same. `TmuxObserver` implements it (already does, just make it explicit).

Add `HeartbeatObserver` — purely file-based, no coupling to live session objects:
- Reads `heartbeat.json` from the session directory. Contains a timestamp and a TTL (set by the writer).
- Returns `Alive: true` if the timestamp is within 2× the TTL, `Alive: false` otherwise.
- Works across process restarts — the monitor doesn't need session references, just the filesystem.

Add `CompositeObserver` — routes per session based on runtime metadata:
- On each `Observe(sessionID)` call, reads the `runtime` field from the session's `spawn.json`
- Routes to `TmuxObserver` when runtime is `""` or `"tmux"`, routes to `HeartbeatObserver` otherwise
- Caches the runtime→observer mapping per session (runtime doesn't change mid-session)
- Holds both a `TmuxObserver` and `HeartbeatObserver` internally

**`monitor/monitor.go`**
Change `observer` field from `*TmuxObserver` to `Observer` interface. Constructor creates a `CompositeObserver` wrapping both `TmuxObserver` and `HeartbeatObserver`. No per-session selection logic in monitor — `CompositeObserver` handles it.

**`dispatcher/streaming_session.go` (from Phase 3)**
Write `heartbeat.json` periodically (every 10s) from the monitor goroutine. Also write on each event received. Contains `{"timestamp": "...", "ttl_seconds": 30}`. This gives `HeartbeatObserver` a reliable liveness signal even during long gaps between events.

**`dispatcher/polling_session.go` (from Phase 4)**
Write `heartbeat.json` on each poll cycle with `{"timestamp": "...", "ttl_seconds": 15}`. The poll interval (default 5s) naturally keeps the heartbeat fresh.

## Verification

### Static
- Compiles, passes vet
- Existing monitor tests pass unchanged (TmuxObserver still default)

### Runtime
- Unit test: HeartbeatObserver returns alive when heartbeat timestamp is within TTL
- Unit test: HeartbeatObserver returns not-alive when heartbeat is stale (>2× TTL)
- Unit test: HeartbeatObserver returns not-alive when heartbeat file missing
- Unit test: CompositeObserver routes tmux session to TmuxObserver
- Unit test: CompositeObserver routes remote session (runtime="sprites") to HeartbeatObserver
- Unit test: CompositeObserver caches routing decision per session
