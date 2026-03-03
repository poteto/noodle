Back to [[plans/90-interactive-sessions/overview]]

# Phase 3 — Multi-turn Message Delivery

## Goal

Enable multi-turn conversation in interactive sessions. The user sends messages via `steer`, the agent processes them and responds, and the event stream records each user turn.

## Changes

**`loop/cook_steer.go`** — Extend `controlSteer()` to check `l.chats` map when the target is an interactive session ID. Route to `steerChat()` which:
1. Check `session.Status()` — if not `"running"`, return error immediately (prevents sending to a session that's in the completion buffer but not yet cleaned up)
2. Gets the session's `AgentController`
3. If steerable (Claude): check single-inflight guard, then interrupt current turn, send message via `SendMessage()`
4. If not steerable: return error (interactive sessions require steerability — enforced at spawn time too, but defense in depth)
5. If inflight guard rejects (a message is already pending delivery): return error so the UI can show "message pending, try again"

**`loop/chat.go`** — Add `steerChat()` method. Record the user message as an `EventAction` with `tool: "user"` — the same event type already used by order session steering. No new event type needed (`EventChatMessage` was considered and rejected — reusing `EventAction` avoids event type proliferation and dual-path drift). The UI differentiates rendering based on session kind, not event type.

**No changes to `event/types.go`** — reuse existing `EventAction`.

**`internal/snapshot/session_events.go`** — No changes needed. `EventAction` with `tool: "user"` already formats as `Label: "User"`.

## Data Structures

- No new types. Reuse `EventAction` with `tool: "user"` payload.

## Single-Inflight Guard

The Claude controller has a single-slot `pendingPrompt`. Rather than allowing silent message drops, `steerChat()` adds a guard: if a message is already pending delivery (interrupt sent, waiting for turn completion), reject the new send with an explicit error. The UI disables the send button or shows "message pending" until delivery completes. This is simpler than a full message queue and makes the constraint visible rather than silent. A proper multi-message queue in the controller is follow-up work.

## Routing

- **Provider:** `claude`
- **Model:** `claude-opus-4-6`

## Verification

### Static
- `go build ./...`
- `go vet ./...`
- Existing steer tests pass (order session steering unchanged)

### Runtime
- Test: steer interactive session, verify `EventAction tool=user` event emitted (same as order steer)
- Test: steer interactive session, verify agent receives message and responds (check canonical event log for subsequent agent actions)
- Test: multiple steer messages in sequence (with pauses between turns), verify all delivered
- Test: steer non-steerable interactive session, verify error returned
- Test: steer a completed interactive session (still in `l.chats` but done), verify error returned
- Test: send second message while first is pending delivery, verify rejection (not silent drop)
- Test: after pending message is delivered, next send succeeds
- Test with `-race`: concurrent steer to same interactive session
