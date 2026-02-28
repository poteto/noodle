Back to [[plans/76-websocket-streaming/overview]]

# Phase 4 — Session Event Broker

## Goal

Build the per-session pub/sub broker that implements `SessionEventSink`. WS clients subscribe to session IDs; the broker fans out events to subscribers.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| claude | claude-opus-4-6 | Concurrent data structure, needs judgment on thread safety |

## Changes

**`server/session_broker.go`** (new):
- `Subscriber` interface: `Send(msg []byte) bool` + `Close()` — `Send` returns false if client is slow/dead; `Close()` triggers client disconnect. Both methods required so the broker can remove and disconnect slow clients. This decouples the broker from `wsClient` (defined in Phase 5), allowing independent development and testing.
- `SessionEventBroker` struct with `sync.RWMutex`, `subscribers map[string]map[Subscriber]struct{}`
- `Subscribe(sessionID, sub)` — add subscriber to session's set
- `Unsubscribe(sessionID, sub)` — remove subscriber from session's set
- `UnsubscribeAll(sub)` — remove subscriber from all sessions (called on disconnect)
- `PublishSessionEvent(sessionID, EventLine)` — implements `dispatcher.SessionEventSink`
  - Marshal event to `{"type":"session_event","session_id":"...","data":<EventLine>}`
  - Call `sub.Send(msg)` for each subscriber
  - If `Send` returns false (slow client), remove subscriber and close it — do not silently drop incremental events, since silent drops create permanent client divergence

**`server/session_broker_test.go`** (new):
- Test subscribe/publish fan-out to multiple clients
- Test unsubscribe stops delivery
- Test UnsubscribeAll on disconnect
- Test slow client gets disconnected (not silently dropped)

## Data Structures

- `Subscriber` — interface for message delivery, decoupled from WS specifics
- `SessionEventBroker` — thread-safe pub/sub keyed by session ID

## Verification

### Static
- `go test ./server/...` — broker tests pass
- `go vet ./server/...`

### Runtime
- Broker is standalone, no integration yet
