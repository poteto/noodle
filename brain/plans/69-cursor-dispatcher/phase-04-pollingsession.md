Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 4: pollingSession

**Routing:** `claude` / `claude-opus-4-6` — concurrency design, state machine, goroutine lifecycle

## Goal

Implement `pollingSession` — a `Session` implementation that polls a `PollingBackend` in a background goroutine. Synthesizes `SessionEvent`s from status transitions and conversation history. Writes `SyncResult` to `spawn.json` on completion (branch-based merge). This is the deferred phase 4 from plan 27, now with real requirements from the Cursor API.

## Data structures

- `pollingSession` struct — holds `PollingBackend`, remote ID, poll interval, runtime dir, session ID, target branch; channels for events/done; mutex-guarded status and cost
- `pollingSessionConfig` struct — constructor input (backend, remote ID, interval, runtime dir, session ID)

## Changes

**`dispatcher/polling_session.go` (new)**
- `newPollingSession(cfg pollingSessionConfig) *pollingSession`
- `start(ctx context.Context)` — launches poll goroutine
- Poll goroutine (`pollLoop`):
  - Every N seconds (configurable, default 10s), calls `backend.PollStatus(remoteID)`
  - On status change: publishes `SessionEvent{Type: "action", Message: status description}`
  - On FINISHED: stores `PollResult.Branch`, fetches conversation via `backend.GetConversation`, emits each message as a `SessionEvent`, writes `SyncResult` via `writeSyncResult`, marks done
  - On ERROR/EXPIRED: marks done with "failed" status
  - On poll error: publishes warning event, continues polling (transient errors shouldn't kill the session)
  - On context cancellation: marks done with "cancelled"
- `ID() string`, `Status() string`, `Events() <-chan SessionEvent`, `Done() <-chan struct{}`
- `TotalCost() float64` — returns 0 (Cursor API doesn't expose cost)
- `Kill() error` — calls `backend.Stop(remoteID)`, marks done with "killed"
- `Controller() AgentController` — returns `NoopController()`
- `markDone(status)` — same pattern as `spritesSession` (sync.Once + close done channel)
- `publish(ev)` — same pattern as `spritesSession` (non-blocking send with drop-oldest)
- `var _ Session = (*pollingSession)(nil)` compile check

**`dispatcher/polling_session_test.go` (new)**
- Test with mock `PollingBackend`:
  - Returns RUNNING twice, then COMPLETED with branch + summary. Verify: status transitions appear as events, done closes, SyncResult written to spawn.json.
  - Returns FAILED → session status "failed", done closes
  - Returns EXPIRED → session status "failed", done closes
  - Kill() calls backend.Stop, done closes
  - Context cancellation → poll goroutine exits, no leaked goroutines
  - PollStatus returns transient error → session continues polling, publishes warning
  - Backend returns conversation on completion → messages emitted as events
- All tests run with `-race`

## Verification

### Static
- `go vet ./dispatcher/...`
- `pollingSession` satisfies `Session` interface

### Runtime
- `go test ./dispatcher/... -run TestPollingSession -race`
- Verify no goroutine leaks (all tests call `markDone` or cancel context)
