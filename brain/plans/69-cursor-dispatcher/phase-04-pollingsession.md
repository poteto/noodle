Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 4: pollingSession

**Routing:** `claude` / `claude-opus-4-6` — concurrency design, state machine, goroutine lifecycle

## Goal

Implement `pollingSession` — a `Session` implementation that polls a `PollingBackend` in a background goroutine. Synthesizes `SessionEvent`s from status transitions and conversation history. Writes `SyncResult` atomically on completion (branch-based merge). This is the deferred phase 4 from plan 27, now with real requirements from the Cursor API.

## Data structures

- `pollingSession` struct — holds `PollingBackend`, remote ID, target branch (from `LaunchResult`), poll interval, runtime dir, session ID; channels for events/done/nudge; mutex-guarded status; event writer for persistence
- `pollingSessionConfig` struct — constructor input (backend, LaunchResult, interval, runtime dir, session ID, event writer)
- `pollingRegistry` struct — mutex-protected `map[string]*pollingSession` for remoteID → session lookup. Supports `Register`, `Unregister`, `Nudge(remoteID)`. Owned by PollingDispatcher, exposed via interface for webhook notifier.

## Changes

**`dispatcher/polling_session.go` (new)**
- `newPollingSession(cfg pollingSessionConfig) *pollingSession`
- `start(ctx context.Context)` — launches poll goroutine
- Poll goroutine (`pollLoop`) — **single-owner state machine**:
  - Selects on: ticker (poll interval), nudge channel (webhook shortcut), stop channel (Kill request), ctx.Done()
  - Every N seconds (configurable, default 10s), calls `backend.PollStatus(remoteID)`
  - On status change: publishes `SessionEvent{Type: "action", Message: status description}`, writes heartbeat
  - On FINISHED: fetches conversation via `backend.GetConversation`, emits each message as a `SessionEvent`, writes `SyncResult` atomically (use `filex.WriteFileAtomic` or separate `sync.json`), calls `backend.Delete` best-effort, marks done with "completed"
  - On ERROR/EXPIRED: calls `backend.Delete` best-effort, marks done with "failed" status
  - On terminal `APIError` (401/403/404): marks done with "failed" immediately — no retry
  - On retryable error (429/5xx/transient): exponential backoff with jitter, publishes warning event, continues polling
  - On stop request (from Kill): calls `backend.Stop(remoteID)`, then `backend.Delete` best-effort, marks done with "killed"
  - On context cancellation: marks done with "cancelled"
  - **Kill() does NOT call markDone directly** — it sends on the stop channel and the poll loop performs the final transition. This prevents the terminal-state race.
- Heartbeat: writes a heartbeat file on each successful poll for monitor liveness detection
- Event persistence: uses `event.EventWriter` to write event log records (like spritesSession) for UI/snapshot consumption
- `ID() string`, `Status() string`, `Events() <-chan SessionEvent`, `Done() <-chan struct{}`
- `TotalCost() float64` — returns 0 (Cursor API doesn't expose cost; documented as known limitation)
- `Kill() error` — signals stop channel (poll loop handles the rest)
- `Nudge()` — non-blocking send on nudge channel to trigger immediate re-poll (for webhook shortcut)
- `Controller() AgentController` — returns `NoopController()`
- `markDone(status)` — same pattern as `spritesSession` (sync.Once + close done channel)
- `publish(ev)` — same pattern as `spritesSession` (non-blocking send with drop-oldest)
- `var _ Session = (*pollingSession)(nil)` compile check

**`dispatcher/polling_registry.go` (new)**
- `pollingRegistry` struct with `sync.RWMutex` and `map[string]*pollingSession`
- `Register(remoteID, session)` — called at dispatch time
- `Unregister(remoteID)` — called when session reaches terminal state
- `Nudge(remoteID)` — looks up session, calls `session.Nudge()` if found, no-op if not
- `SessionNotifier` interface — `Nudge(remoteID string)` — exposed to server for webhook notifier injection

**`dispatcher/polling_session_test.go` (new)**
- Test with mock `PollingBackend`:
  - Returns RUNNING twice, then COMPLETED with branch + summary. Verify: status transitions appear as events, done closes, SyncResult written atomically.
  - Returns FAILED → session status "failed", done closes, Delete called
  - Returns EXPIRED → session status "failed", done closes, Delete called
  - Kill() → sends stop, backend.Stop called, done closes with "killed"
  - Context cancellation → poll goroutine exits, no leaked goroutines
  - Terminal APIError (401) → session fails immediately, no retry
  - Retryable error (429) → backoff, continues polling, publishes warning
  - Backend returns conversation on completion → messages emitted as events
  - Nudge() triggers immediate re-poll (verify with mock that checks poll timing)
  - Kill() during FINISHED transition → single terminal state, no race
  - Heartbeat file written on each successful poll
  - Event writer receives event records
- All tests run with `-race`

## Verification

### Static
- `go vet ./dispatcher/...`
- `pollingSession` satisfies `Session` interface

### Runtime
- `go test ./dispatcher/... -run TestPollingSession -race`
- `go test ./dispatcher/... -run TestPollingRegistry -race`
- Verify no goroutine leaks (all tests reach terminal state or cancel context)
