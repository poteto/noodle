Back to [[plans/76-websocket-streaming/overview]]

# Phase 6 — WebSocket Handler + Server Wiring

## Goal

Add the HTTP upgrade handler and wire the WS hub into the server. Extract `processControl()` from `handleControl` so both REST and WS can use it.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| claude | claude-opus-4-6 | Server architecture, extracting shared logic |

## Changes

**`server/ws_handler.go`** (new):
- `handleWS(w, r)` — HTTP upgrade handler
  - Upgrade with localhost-only origin check (gorilla `CheckOrigin`)
  - Create `wsClient` with buffered send channel
  - Register with hub via `addClient` **before** starting pumps — if pumps start first, a pump can fail and call `removeClient` before registration, then `addClient` re-inserts a dead client
  - Send initial snapshot immediately after upgrade
  - Start `writePump` and `readPump` goroutines

**`server/server.go`**:
- Replace `sse *sseHub` field with `ws *wsHub`
- Accept `*SessionEventBroker` as an option (broker is created externally — see Phase 7 for why)
- Constructor: create `wsHub` with broker reference
- Add route: `mux.HandleFunc("GET /api/ws", s.handleWS)` (line ~85)
- Remove route: `GET /api/events` (SSE)
- In `Start()`: replace `go s.sse.watchAndBroadcast(...)` with `go s.ws.watchAndBroadcast(...)`. On shutdown, call `s.ws.Close()` to clean up hijacked WS connections.
- Extract `processControl(cmd ControlCommand) (ControlAck, error)` from `handleControl` (line ~221):
  - Shared: validate action, generate ID, append to control.ndjson, build ack
  - `handleControl` becomes thin wrapper: parse JSON body → `processControl` → write response
  - WS readPump calls `processControl` directly

**`go.mod`**:
- Promote `github.com/gorilla/websocket` from indirect to direct dependency

## Data Structures

- `processControl()` — pure function extracting shared control logic

## Verification

### Static
- `go build ./server/...`
- `go vet ./...`

### Runtime
- WS endpoint is live but no frontend connects yet
- REST `/api/control` still works (thin wrapper over shared logic)
- `wscat -c ws://localhost:PORT/api/ws` should connect and receive initial snapshot
