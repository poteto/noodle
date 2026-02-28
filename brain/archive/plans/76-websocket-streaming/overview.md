---
id: 76
created: 2026-02-27
status: active
---

# Replace SSE with WebSocket for Real-time Streaming

Back to [[plans/index]]

## Context

The UI uses two separate channels: SSE (`GET /api/events`) for snapshot updates and HTTP polling every 3 seconds for session events. With live chat to agents now possible, we need bidirectional communication and real-time event streaming. WebSocket replaces both SSE and polling with a single connection that pushes events instantly and receives commands.

## Scope

**In scope:**
- Single WebSocket endpoint at `GET /api/ws` replacing SSE and polling
- In-memory event sink from dispatcher → broker → WS clients
- Per-session event subscriptions with backfill
- Control commands over WebSocket
- Delete SSE code (server and client)

**Out of scope:**
- Chat/steer message sending (already works via REST, WS carries control commands)
- Authentication/authorization (localhost-only, same as current SSE)
- Event persistence changes (eventWriter stays as-is)
- streaming-markdown integration (separate concern)

## Constraints

- gorilla/websocket v1.5.3 already in go.mod as indirect dep — promote to direct
- Must maintain snapshot diff-gating (SHA256 dedup) from SSE hub
- Frontend keeps React Query cache pattern — WS pushes data into queryClient
- No backward compatibility needed per project principle — SSE deleted outright

## Alternatives Considered

1. **Keep SSE + add WS for events only** — simpler but maintains two connection types, doesn't solve bidirectional need
2. **Full WebSocket replacement** (chosen) — single connection, bidirectional, cleaner architecture
3. **Server-push via HTTP/2 streams** — overly complex for localhost communication

## Protocol

Single WS connection at `GET /api/ws`. Typed JSON messages both directions.

**Server → Client:**
```
{"type": "snapshot",      "data": <Snapshot>}
{"type": "backfill",      "session_id": "...", "data": [<EventLine>...]}
{"type": "session_event", "session_id": "...", "data": <EventLine>}
{"type": "subscribed",    "session_id": "..."}
{"type": "unsubscribed",  "session_id": "..."}
{"type": "control_ack",   "data": <ControlAck>}
{"type": "error",         "message": "..."}
```

`backfill` replaces the client cache for that session. `session_event` appends a single live event.

**Client → Server:**
```
{"type": "subscribe",   "session_id": "..."}
{"type": "unsubscribe", "session_id": "..."}
{"type": "control",     "data": <ControlCommand>}
```

## Architecture

```
Agent stdout → stamp → canonical → processSession.consumeCanonicalLine()
                                      ├─ eventWriter.Append() (disk)
                                      └─ sink.Publish(sessionID, eventLine) → broker → WS clients

fsnotify(.noodle/) → debounce 300ms → loadSnapshot → hash dedup → wsHub.broadcastSnapshot()
```

The dispatcher's `processSession` already writes events to disk via `eventWriter.Append()`. We add a `SessionEventSink` callback that publishes to a broker for fan-out to subscribed WS clients — no disk round-trip for real-time delivery.

## Applicable Skills

- `go-best-practices` — lifecycle, concurrency, testing patterns
- `ts-best-practices` — type safety for WS message types
- `testing` — test-driven workflow for both backend and frontend
- `react-best-practices` — hooks, effects, data fetching patterns

## Phases

1. [[plans/76-websocket-streaming/phase-01-event-sink-interface]]
2. [[plans/76-websocket-streaming/phase-02-wire-sink-process-dispatcher]]
3. [[plans/76-websocket-streaming/phase-03-wire-sink-sprites-dispatcher]]
4. [[plans/76-websocket-streaming/phase-04-session-broker]]
5. [[plans/76-websocket-streaming/phase-05-websocket-hub]]
6. [[plans/76-websocket-streaming/phase-06-websocket-handler-server-wiring]]
7. [[plans/76-websocket-streaming/phase-07-loop-main-wiring]]
8. [[plans/76-websocket-streaming/phase-08-backend-tests-delete-sse]]
9. [[plans/76-websocket-streaming/phase-09-frontend-ws-client]]
10. [[plans/76-websocket-streaming/phase-10-frontend-hooks-consumers]]

Phases 1–3 (sink interface + dispatcher wiring) are sequential foundations.
Phase 4 (broker) before Phase 5 (hub) — hub's `wsClient` implements broker's `Subscriber` interface.
Phases 6–7 (server + loop wiring) connect everything.
Phase 8 (tests + SSE deletion) validates backend.
Phases 9–10 (frontend) can parallelize with phases 4–8 in a separate worktree.

## Key Design Decisions (from Codex review)

**Broker uses `Subscriber` interface, not `*wsClient`** — decouples broker from WS specifics, enables independent testing. `wsClient` implements `Subscriber` in Phase 5.

**Disconnect slow clients, don't drop events** — incremental `session_event` messages can't be silently dropped without permanent client divergence. Slow clients get disconnected, forcing reconnect + backfill.

**`sync.Once` teardown on `wsClient`** — `removeClient` can be triggered from both `readPump` and `writePump`. Without idempotent teardown, closing the `send` channel twice panics the process.

**Hub `Close()` for clean shutdown** — `http.Server.Shutdown` does not clean up hijacked WS connections. Explicit hub-wide close prevents goroutine/socket leaks.

**Register before pumps** — `addClient` must happen before `writePump`/`readPump` start to prevent a race where a pump fails and calls `removeClient` before the client is registered.

**Subscribe-first backfill ordering** — on subscribe: `broker.Subscribe()` first (live events queue in send channel), then read backfill from disk, then send as `{"type":"backfill"}` (cache replace, not append). Live events queued during disk read arrive after via send channel. Client dedupes overlap by timestamp.

**Broker created independently of server** — `cmd_start.go` creates loop before server. Broker is created first and passed to both, avoiding the circular dependency. The `newStartRuntimeLoop` factory signature must change to accept `Dependencies`.

**Dispatcher structs store sink, not just configs** — adding `Sink` to dispatcher configs is not enough; the dispatcher struct must persist the sink so `Dispatch()` can forward it when creating sessions later.

**`connectWS` at app root, not per-hook** — WS is a singleton connection. Calling it in `useEffect` from multiple hooks risks one unmount killing the shared connection (especially under Strict Mode). Called once at app level; hooks just read from the cache.

**`backfill` vs `session_event` message types** — backfill replaces the client cache; session_event appends. On reconnect, re-subscribe triggers a new backfill (cache replacement), preventing duplicate accumulation.

**`staleTime: Infinity` on session events query** — prevents React Query from auto-refetching (which could overwrite newer WS-pushed events with a stale REST response). WS is authoritative after initial seed.

**Reference-counted frontend subscriptions** — multiple components (`AgentFeed`, `ContextPanel`) subscribe to the same session. Ref-counting ensures one unmount doesn't kill another's stream.

**Client-generated control IDs for ack correlation** — `ControlCommand.id` is pre-populated by the client before sending over WS, enabling correct promise resolution when multiple commands are in-flight.

**All writes go through `send` channel** — gorilla/websocket requires a single writer goroutine. Backfill, snapshots, and all other messages must be enqueued on the `send` channel and written by `writePump`. Never write directly to conn from `readPump`.

**`Subscriber` interface includes `Close()`** — broker needs both `Send()` and `Close()` to implement the disconnect-slow-clients policy. Without `Close()`, the broker can remove a subscriber from its map but can't actually disconnect the client.

**`emitPromptEvent()` needs its own sink call** — this method writes events directly via `eventWriter.Append()` without going through `consumeCanonicalLine()`. Sink wiring in `consumeCanonicalLine` alone misses prompt events in real-time streaming.

**Constructor-time sink wiring only** — `defaultDependencies(sink)` threads the sink to dispatcher configs at construction. No post-construction patching — dispatchers must have the sink from the start.

**No REST `queryFn` for session events** — WS backfill is the sole data source. A REST `queryFn` races with WS backfill: if WS writes newer data first and REST resolves later, React Query overwrites with stale data.

**Single ref-count map for subscriptions** — `sessionRefCounts: Map<string, number>` serves both as the ref counter and as the reconnection tracking. No separate `subscribedSessions` set — two structures diverge silently.

## Verification

```bash
go test ./server/... ./dispatcher/... ./loop/...
go vet ./...
sh scripts/lint-arch.sh
pnpm --filter noodle-ui test
```

Runtime: start noodle, open UI, verify snapshot updates arrive in real-time, click into a session and see events stream without 3s delay, send a steer command via WS, confirm single WS connection in devtools Network tab (no SSE, no polling).
