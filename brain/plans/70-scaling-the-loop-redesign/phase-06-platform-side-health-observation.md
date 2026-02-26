Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 6: Platform-side health observation

## Goal

Move session health monitoring into the Runtime. Each runtime observes its own sessions internally and pushes health events to the loop via a channel. The loop no longer runs `Monitor.RunOnce()` to enumerate and check all sessions.

## Changes

**`runtime/runtime.go`** — Add `HealthEvent` type with a monotonic sequence number per session to prevent stale events from regressing state. Add `Health() <-chan HealthEvent` method to the Runtime interface (already has `Start`/`Close` from phase 3 — observation goroutines start in `Start()` and stop in `Close()`).

**`runtime/tmux.go`** — `TmuxRuntime.Start()` launches a background goroutine that runs the existing tmux observation logic (batch `tmux list-sessions`, meta file reads) on a configurable interval. Pushes `HealthEvent` entries for stuck, idle, or dead sessions. **Terminal events** (`dead`) use context-aware blocking send (`select` on `ctx.Done()` and channel send) — they must not be dropped, but must also not block shutdown. If the context is cancelled (runtime closing), the event is lost but that's safe because the loop is shutting down anyway. **Informational events** (`stuck`, `idle`, `healthy`) use non-blocking send with last-writer-wins per session — dropping a stale "still stuck" event is safe because the observer will re-emit it.

**`runtime/sprites.go`** — `SpritesRuntime` uses the existing heartbeat observer internally. Pushes HealthEvent on status changes. Same context-aware-blocking for terminal, non-blocking for informational.

**`loop/loop.go`** — `runCycleMaintenance()` drains health channels from all registered runtimes (fan-in). For each event, update the session's known health state. If the event's sequence number is lower than the last seen for that session, discard it. Health events feed into stuck detection, failure handling, and status reporting.

**`loop/types.go`** — Remove `Monitor` interface from Dependencies. The loop receives health via runtime channels. Add `sessionHealth map[string]HealthEvent` to track latest health per session.

**`monitor/`** — The monitor package becomes internal to the runtime layer. Its types and logic are preserved but no longer called directly by the loop.

## Data structures

- `HealthEvent` — `SessionID string`, `OrderID string`, `Type HealthEventType` (typed constant: `HealthStuck`, `HealthIdle`, `HealthDead`, `HealthHealthy`), `Detail string`, `At time.Time`, `Seq uint64`
- `sessionHealth map[string]HealthEvent` — latest health per session, updated on drain

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — requires judgment about observation cadence, stale-event handling, and graceful shutdown

## Verification

### Static
- `go test ./...` — all tests pass
- `Monitor` interface removed from loop Dependencies
- Loop does not import `monitor` package directly
- Health events have monotonic sequence numbers per session

### Runtime
- Integration test: start a session, kill it externally, verify HealthEvent(dead) arrives and loop handles failure
- Test: stuck session (no canonical.ndjson updates) produces HealthEvent(stuck)
- Test: stale health event (lower seq) is discarded
- Test: terminal event (dead) blocks until loop drains, but unblocks when context is cancelled (no deadlock on Close())
- Test: informational events (stuck, idle) use non-blocking send — observer doesn't block when channel is full
- Race detector: `go test -race ./...`
- Verify runtime observation goroutines stop cleanly when `Close()` is called
