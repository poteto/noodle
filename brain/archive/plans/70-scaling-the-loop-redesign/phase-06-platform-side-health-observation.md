Back to [[archive/plans/70-scaling-the-loop-redesign/overview]]

# Phase 6: Platform-side health observation

## Goal

Move session health monitoring into the Runtime. Each runtime observes its own sessions internally and pushes health events to the loop via a channel. The loop no longer runs `Monitor.RunOnce()` to enumerate and check all sessions.

## Changes

**`runtime/runtime.go`** — Add `HealthEvent` type with a monotonic sequence number per session to prevent stale events from regressing state. Add `Health() <-chan HealthEvent` method to the Runtime interface (already has `Start`/`Close` from phase 3 — observation goroutines start in `Start()` and stop in `Close()`). **Prerequisite**: this phase depends on phase 3 delivering the `runtime/` package. All file paths below reference `runtime/` which is created in phase 3.

**`runtime/tmux.go`** — `TmuxRuntime.Start()` launches a background goroutine that runs the existing tmux observation logic (batch `tmux list-sessions`, meta file reads) on a configurable interval. Pushes `HealthEvent` entries for stuck, idle, or dead sessions. To avoid head-of-line blocking, emit via two paths: `terminalHealth` (`dead`, reliable and drained first) and `infoHealth` (`stuck`/`idle`/`healthy`, non-blocking last-writer-wins). If runtime context is cancelled, pending terminal events can be dropped because shutdown is already in progress.

**`runtime/sprites.go`** — `SpritesRuntime` uses the existing heartbeat observer internally. Pushes HealthEvent on status changes. Same split-path semantics as tmux (`terminalHealth` reliable-first drain, `infoHealth` coalesced non-blocking).

**`loop/loop.go`** — `runCycleMaintenance()` drains health channels from all registered runtimes (fan-in). For each event, update the session's known health state. If the event's sequence number is lower than the last seen for that session, discard it. Health events feed into stuck detection, failure handling, and status reporting.

**`loop/types.go`** — Remove `Monitor` interface from Dependencies. The loop receives health via runtime channels. Add `sessionHealth map[string]HealthEvent` to track latest health per session.

**`monitor/`** — The monitor package becomes internal to the runtime layer. Its types and logic are preserved but no longer called directly by the loop. **Critical**: `Monitor.RunOnce()` currently produces two outputs consumed outside the loop: (a) session meta files written per-session (`monitor.go:74`) consumed by `sessionmeta.ReadAll()` in `mise/builder.go:160`, `cmd_status.go:83`, and `internal/snapshot/snapshot.go:128`; (b) materialized tickets (`monitor.go:80`) consumed by the snapshot and brief. Migration ownership is explicit: session meta production stays runtime-owned; ticket materialization becomes **loop-owned** via a single ticket materializer that consumes runtime health events and writes canonical ticket artifacts.

**Health event taxonomy mapping**: Current monitor uses `running`/`stuck`/`exited`/`failed` plus health colors green/yellow/red (`monitor/types.go:15-30`, `monitor/derive.go:32-67`). The proposed `HealthStuck`/`HealthIdle`/`HealthDead`/`HealthHealthy` must map cleanly:
- `running` + green → `HealthHealthy`
- `running` + yellow (approaching stuck) → `HealthIdle`
- `stuck` → `HealthStuck`
- `exited`/`failed` → `HealthDead`

Define this mapping table in the runtime package as a canonical reference.

## Data structures

- `HealthEvent` — `SessionID string`, `OrderID string`, `Type HealthEventType` (typed constant: `HealthStuck`, `HealthIdle`, `HealthDead`, `HealthHealthy`), `Detail string`, `At time.Time`, `Seq uint64`
- `sessionHealth map[string]HealthEvent` — latest health per session, updated on drain
- Health channel: buffered to 256 per runtime. Per-session coalescing via last-writer-wins map inside the observer goroutine — only emit when state changes, not on every observation tick. Drop metric counter for monitoring channel saturation.

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
- Test: terminal events drain before informational events under burst (no starvation, no deadlock on Close())
- Test: informational events (stuck, idle) use non-blocking send — observer doesn't block when channel is full
- Race detector: `go test -race ./...`
- Verify runtime observation goroutines stop cleanly when `Close()` is called
- Test: health-event-driven ticket materializer emits the same ticket artifacts as pre-migration monitor flow
