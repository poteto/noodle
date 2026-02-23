Back to [[plans/27-remote-dispatchers/overview]]

# Phase 7: Monitor observer abstraction

**Routing:** `codex` / `gpt-5.3-codex` — mechanical refactor with clear target

## Goal

The monitor uses `TmuxObserver` to check if sessions are alive (`tmux has-session`). Extract an `Observer` interface so remote sessions can report liveness through their own mechanism. For streaming sessions, liveness comes from process state. For polling sessions, liveness comes from the last polled status.

## Data structures

- `Observer` interface — `Observe(sessionID) → (Observation, error)` (already exists as a concrete type)
- `SessionObserver` — runtime-aware observer for remote sessions. Checks session liveness via the session's own state (done channel / status) rather than external heuristics. Falls back to heartbeat file for monitor restarts.

## Changes

**`monitor/observer.go`**
Extract `Observer` interface from `TmuxObserver`. The `Observe` method signature stays the same. `TmuxObserver` implements it (already does, just make it explicit).

Add `SessionObserver` that checks liveness via the session's own state:
- Primary: query the session's `Status()` and `Done()` channel directly (streaming sessions track backend liveness, polling sessions track poll loop state)
- Fallback (monitor restart, session from previous process): check a heartbeat file (`heartbeat.json`) written periodically by the dispatcher session goroutines. Contains a timestamp — alive if within 2× the write interval. This avoids the false-failure problem of a static mtime threshold on `canonical.ndjson`, since remote sessions may go long periods between events.

**`monitor/monitor.go`**
Change `observer` field from `*TmuxObserver` to `Observer` interface. Constructor accepts `Observer`. No other changes — `DeriveSessionMeta` and `ClaimsReader` are already generic.

**`dispatcher/streaming_session.go` (from Phase 3)**
Write `heartbeat.json` periodically (every 10s) from the monitor goroutine. Also write on each event received. This gives `SessionObserver` a reliable liveness signal even during long gaps between events.

**`dispatcher/polling_session.go` (from Phase 4)**
Write `heartbeat.json` on each poll cycle. The poll interval (default 5s) naturally keeps the heartbeat fresh.

## Verification

### Static
- Compiles, passes vet
- Existing monitor tests pass unchanged (TmuxObserver still default)

### Runtime
- Unit test: SessionObserver returns alive when session status is "running"
- Unit test: SessionObserver returns not-alive when done channel is closed
- Unit test: heartbeat fallback — alive when heartbeat is fresh, not-alive when stale (>2× interval)
- Unit test: Monitor works with SessionObserver
