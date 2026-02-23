Back to [[plans/27-remote-dispatchers/overview]]

# Phase 4: PollingDispatcher generic implementation

**Routing:** `claude` / `claude-opus-4-6` — architectural judgment, state machine design

## Goal

Build the generic `PollingDispatcher` that accepts any `PollingBackend`. This dispatcher handles: launch the remote agent, poll for status transitions, synthesize events from status changes and conversation history, detect completion.

Lower event fidelity than streaming — no tool-by-tool granularity during execution. The TUI trace view will show status transitions and the conversation when the agent finishes.

## Data structures

- `PollingDispatcher` struct — holds `PollingBackend`, runtime dir, poll interval
- `pollingSession` struct — implements `Session`. Holds remote ID, events channel, done channel, status, last-seen conversation length (for incremental fetch)

## Changes

**`dispatcher/polling_dispatcher.go` (new)**
`PollingDispatcher.Dispatch`:
1. Validate request, generate session ID, create session dir
2. Call `backend.Launch(ctx, config)` → remote ID
3. Write `spawn.json` with remote ID in metadata
4. Return `pollingSession`, launch poll goroutine

**`dispatcher/polling_session.go` (new)**
`pollingSession` implementation:
- `start(ctx)` launches one goroutine: `pollLoop`
- `pollLoop`: every N seconds (configurable, default 5s), calls `backend.PollStatus(remoteID)`. On status change, publishes a `SessionEvent`. When terminal status reached (Completed/Failed/Expired), fetches conversation via `backend.GetConversation(remoteID)`, emits each new message as a `SessionEvent`, marks done.
- Synthesized events: `SessionEvent{Type: "action", Message: "text:Agent is running..."}` for RUNNING, `SessionEvent{Type: "action", Message: "text:<conversation message>"}` for each conversation turn at completion, `SessionEvent{Type: "complete"}` for terminal.
- Still writes `canonical.ndjson` and `events.ndjson` for compatibility, but with synthesized content.
- `Kill()`: calls `backend.Stop(remoteID)`, marks done
- Cost: polling backends may not report cost. `TotalCost()` returns 0 unless backend provides it in status response.

## Verification

### Static
- Compiles, passes vet
- `pollingSession` satisfies `Session` interface

### Runtime
- Unit test with mock `PollingBackend`: returns RUNNING twice, then COMPLETED with conversation. Verify status transitions appear as events, conversation messages emit, done channel closes.
- Unit test: backend returns FAILED → session status "failed", done closes
- Unit test: Kill() calls backend.Stop
- Unit test: context cancellation mid-poll → poll goroutine exits cleanly, no leaked goroutines
- Unit test: backend.PollStatus returns error → session handles gracefully (retry or fail, not hang)
- Run all tests with `-race` flag
