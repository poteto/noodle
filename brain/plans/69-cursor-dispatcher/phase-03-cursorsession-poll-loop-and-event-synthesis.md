Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 3: cursorSession poll loop and event synthesis

**Routing:** `claude` / `claude-opus-4-6` — state machine design, concurrency, event synthesis judgment

## Goal

Implement `cursorSession` — the Session interface implementation for Cursor Cloud Agents. It runs a polling loop that checks status and conversation, synthesizes SessionEvents, and signals completion. This is the Cursor equivalent of `spritesSession`.

## Data structures

- `cursorSession` struct — holds cursor agent ID, backend reference, events channel, done channel, poll interval, last-seen conversation length (for incremental fetch)
- `cursorSessionConfig` struct — construction params (agent ID, backend, poll interval, runtime dir, session ID, event writer)

## Changes

**`dispatcher/cursor_session.go` (new)**

Session implementation:

Constructor: `newCursorSession(cfg cursorSessionConfig) *cursorSession`. Initializes channels, sets initial status to "running".

`start(ctx)` launches one goroutine: `pollLoop`.

`pollLoop`:
1. Tick every N seconds (configurable, default 5s)
2. Call `backend.PollStatus(agentID)`. On status change from previous, emit a SessionEvent:
   - CREATING → running: `{Type: "action", Message: "Cursor agent initializing..."}`
   - Running: no event (already emitted)
   - FINISHED → completed: handled below
   - ERROR → failed: `{Type: "error", Message: "Cursor agent failed"}`
3. Call `backend.GetConversation(agentID)`. Compare message count to `lastSeenCount`. For each new message, emit:
   - assistant_message → `{Type: "action", Message: <text>}`
   - user_message → `{Type: "action", Message: "Follow-up: " + <text>}`
4. On terminal status (completed/failed), do a final conversation fetch, emit remaining messages, then `markDone(status)`.
5. On context cancellation, clean up and `markDone("cancelled")`.

The polling loop also checks for a webhook notification channel — if a webhook arrives (phase 5), it short-circuits the next poll and processes immediately.

`webhookNotify(status string)` — method that sends a signal to the polling loop to wake up and process a status change immediately. Used by the webhook receiver in phase 5. For now, this is a no-op channel that the poll loop selects on.

Session interface methods — same pattern as spritesSession:
- `ID()`, `Status()`, `Events()`, `Done()`, `TotalCost()` — standard accessors
- `Kill()` — calls `backend.Stop(agentID)`, marks done as "killed"
- `publish()` — non-blocking event send with drop-on-full (reuse pattern from spritesSession)
- `markDone()`, `closeEventsWhenDone()` — same sync.Once pattern

Event log: write synthesized events to `events.ndjson` via the event writer, and write a simplified `canonical.ndjson` for monitor compatibility. Since Cursor doesn't provide cost/token data, TotalCost returns 0.

**`dispatcher/cursor_session_test.go` (new)**

- Mock backend returns CREATING → RUNNING → FINISHED with conversation. Verify: status events appear, conversation messages emit as actions, done channel closes, final status is "completed".
- Mock backend returns RUNNING → ERROR. Verify: session status is "failed", done closes.
- Kill() calls backend.Stop, session status is "killed".
- Context cancellation mid-poll → poll goroutine exits, no leaked goroutines (use goleak or manual check).
- Conversation incremental fetch: only new messages since last poll are emitted.
- Backend.PollStatus returns transient error → session retries on next tick, doesn't crash.

## Verification

### Static
- Compiles, passes vet
- `cursorSession` satisfies `Session` interface (`var _ Session = (*cursorSession)(nil)`)
- All tests pass: `go test ./dispatcher/... -race -count=1`

### Runtime
- Unit tests with mock backend cover happy path, failure, kill, cancellation, incremental conversation, transient errors
- Race detector finds no issues
