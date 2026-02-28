Back to [[plans/76-websocket-streaming/overview]]

# Phase 8 — Backend Tests + Delete SSE

## Goal

Replace SSE tests with WebSocket tests. Delete `server/sse.go`. Per migrate-callers-then-delete principle: callers are migrated (Phases 5-7), now delete the old API.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| claude | claude-opus-4-6 | Test design, judgment on coverage |

## Changes

**`server/server_test.go`**:
- Replace `TestSSEStream` (line ~203) with `TestWSConnection`:
  - Upgrade to WS, read initial snapshot, validate JSON structure
- Replace `TestSSEHubDiffGating` (line ~306) with `TestWSHubDiffGating`:
  - Same dedup logic validation but over WS
- Add `TestWSSubscribeSessionEvents`:
  - Subscribe to session, verify backfill events arrive
  - Write new event to disk, verify it arrives via WS
- Add `TestWSControl`:
  - Send control command via WS, verify ack message

**Delete `server/sse.go`**:
- All functionality now lives in `ws_hub.go`, `ws_handler.go`, `session_broker.go`
- Remove `newSSEHub()` call from server constructor
- Remove `handleSSE` route

## Verification

### Static
- `go test ./server/...` — all new WS tests pass
- `go vet ./server/...`
- `go build ./...` — no dangling references to SSE types

### Runtime
- Same as Phase 7 runtime verification
