Back to [[plans/76-websocket-streaming/overview]]

# Phase 9 — Frontend WebSocket Client

## Goal

Create `ws.ts` replacing `sse.ts`. Single WebSocket connection to `/api/ws` that pushes snapshots and session events into React Query cache.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| claude | claude-opus-4-6 | Architecture, reconnection logic, React Query integration |

## Changes

**`ui/src/client/ws.ts`** (new):
- `WSStatus` type: `"connected" | "connecting" | "disconnected"`
- `WS_STATUS_KEY` = `["wsStatus"]`
- `SNAPSHOT_KEY` = `["snapshot"]` (same as current SSE)
- `connectWS(queryClient: QueryClient): () => void`
  - **Called once at app root** (e.g., root route or QueryClient setup), not per-hook. Multiple hooks read from the cache the WS populates. The cleanup function is a no-op ref-decrement; the socket stays alive for the app lifetime.
  - Connect to `ws://${location.host}/api/ws`
  - On open: set status "connected"
  - On message, dispatch by `type`:
    - `"snapshot"` → `queryClient.setQueryData(SNAPSHOT_KEY, normalizeSnapshot(msg.data))`
    - `"backfill"` → `queryClient.setQueryData(["sessionEvents", msg.session_id], msg.data)` — **replaces** the cache (not append). Server sends this on subscribe with full event history.
    - `"session_event"` → `queryClient.setQueryData(["sessionEvents", msg.session_id], (old = []) => [...old, msg.data])` — **appends** a single live event. Dedupe by `at` timestamp if overlap with backfill tail.
    - `"subscribed"` / `"unsubscribed"` — no-op (confirmation)
    - `"control_ack"` — resolve pending control promise by matching `id`
    - `"error"` — log warning
  - On close: set status "disconnected", reconnect after 2s, re-subscribe active sessions
  - On error: close and let onclose handle reconnect
  - On close: re-subscribe all sessions with refcount > 0 from the ref-count map
- `subscribeSession(sessionId: string)` — **reference-counted** via a single `Map<string, number>`. Increment ref count, only send WS `subscribe` message when count goes from 0→1. This same map is used for reconnection — no separate tracking set. Multiple components (AgentFeed, ContextPanel, SchedulerFeed) may subscribe to the same session simultaneously.
- `unsubscribeSession(sessionId: string)` — decrement ref count, only send WS `unsubscribe` when count hits 0. Prevents one component's unmount from killing another's stream.
- `sendWSControl(cmd: ControlCommand): Promise<ControlAck>` — pre-populate `cmd.id` (client-generated UUID) before sending. Match incoming `control_ack` messages by `id` to resolve the correct pending promise. The `ControlCommand` type already has an optional `id` field.

**Delete `ui/src/client/sse.ts`**

## Data Structures

- `WSMessage` — discriminated union for server→client messages
- `WSClientMessage` — discriminated union for client→server messages
- Module-level `ws` ref and `sessionRefCounts: Map<string, number>` for both counting and reconnect tracking (single source of truth)

## Verification

### Static
- `pnpm --filter noodle-ui exec tsc --noEmit`
- No import errors from deleted sse.ts

### Runtime
- Verify WS connection in browser devtools Network → WS tab
- Snapshot updates arrive without page reload
