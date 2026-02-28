Back to [[plans/76-websocket-streaming/overview]]

# Phase 5 — WebSocket Hub

## Goal

Build the WebSocket hub that replaces `sseHub`. Manages connected clients, broadcasts snapshots via fsnotify, and coordinates with the session broker.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| claude | claude-opus-4-6 | Architecture decision, gorilla/websocket lifecycle |

## Changes

**`server/ws_hub.go`** (new):
- `wsClient` struct: `conn *websocket.Conn`, `send chan []byte`, `hub *wsHub`, `closeOnce sync.Once`
  - Implements `Subscriber` interface from Phase 4: `Send(msg []byte) bool` does non-blocking channel send, returns false if full
  - `closeOnce` guards teardown — `removeClient` can be triggered from both `readPump` (read error) and `writePump` (write error). Without `sync.Once`, closing the `send` channel twice panics.
- `wsHub` struct: `sync.RWMutex`, `clients map[*wsClient]struct{}`, `broker *SessionEventBroker`, `lastHash [sha256.Size]byte`, `done chan struct{}`
- `watchAndBroadcast(ctx, runtimeDir)` — same fsnotify + 300ms debounce + SHA256 dedup logic as current `sseHub.watchAndBroadcast()` in `server/sse.go` (line 76). Sends `{"type":"snapshot","data":<json>}` instead of SSE `data:` format
- `broadcastSnapshot(data []byte)` — non-blocking send to all clients
- `addClient(c)` / `removeClient(c)` — register/unregister. `removeClient` uses `closeOnce` to: unsubscribe from broker, close send channel, close conn. Called idempotently from either pump.
- `Close()` — hub-wide shutdown: close all client connections. Required because `http.Server.Shutdown` does not clean up hijacked WebSocket connections — without explicit cleanup, goroutines and sockets leak.
- `client.writePump()` — single goroutine drains `send` channel to conn
- `client.readPump(s *Server)` — reads upstream JSON messages, dispatches by type:
  - `subscribe` → `broker.Subscribe()` first (live events start queuing in send channel via writePump), then read backfill from disk, then enqueue backfill as `{"type":"backfill","session_id":"...","data":[...EventLine]}` onto the `send` channel (NOT directly to conn — gorilla/websocket requires a single writer goroutine, which is `writePump`). Live events queued during disk read arrive after via the same channel. Client treats backfill as cache replacement (not append). Any overlap between backfill tail and queued live events is deduped by timestamp on the client.
  - `unsubscribe` → `broker.Unsubscribe()` + ack
  - `control` → call `server.processControl()` → send ack
  - Calls `removeClient` on read error/close

## Data Structures

- `wsClient` — per-connection state, implements `Subscriber`, `sync.Once` teardown
- `wsHub` — connection registry + snapshot broadcaster + `Close()` for clean shutdown
- `wsMessage` — typed envelope for client→server messages

## Verification

### Static
- `go build ./server/...`

### Runtime
- Hub is built but not wired to routes yet (Phase 6)
