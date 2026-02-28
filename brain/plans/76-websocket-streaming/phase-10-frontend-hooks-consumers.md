Back to [[plans/76-websocket-streaming/overview]]

# Phase 10 — Frontend Hooks + Consumer Updates

## Goal

Update hooks to use WebSocket and update all components that reference SSE status. This is the final frontend phase.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| codex | gpt-5.3-codex | Mechanical renames and hook updates |

## Changes

**`ui/src/client/hooks.tsx`**:
- `useSnapshot()` / `useSuspenseSnapshot()`: Remove `connectSSE` effect entirely — WS is connected at the app root, not per-hook. These hooks just read from the cache.
- `useWSStatus()` (rename from `useSSEStatus`): Read from `WS_STATUS_KEY` instead of `SSE_STATUS_KEY`
- `useSessionEvents(sessionId)`: Replace 3s polling with WS subscription:
  ```tsx
  useEffect(() => {
    if (!sessionId) return;
    subscribeSession(sessionId);
    return () => unsubscribeSession(sessionId);
  }, [sessionId]);

  return useQuery<EventLine[]>({
    queryKey: ["sessionEvents", sessionId],
    queryFn: () => [],            // No REST fetch — WS backfill is the sole data source
    enabled: Boolean(sessionId),
    staleTime: Infinity,          // WS is authoritative — prevent React Query from refetching
    refetchOnWindowFocus: false,
    refetchOnMount: false,        // Cache is populated by WS backfill, not REST refetch
  });
  ```
  - No REST `queryFn` — WS `backfill` message replaces the cache on subscribe; `session_event` messages append. A REST queryFn would race with WS backfill: if WS writes newer data first and REST resolves later, React Query overwrites with stale data.
- `useSendControl()`: Try WS first, fall back to REST on error

**Root route / app setup**:
- Call `connectWS(queryClient)` once at app level (e.g., in `__root.tsx` or QueryClient provider). Not in individual hooks.

**`ui/src/client/index.ts`**:
- Replace exports: `connectSSE` → `connectWS`, `SSEStatus` → `WSStatus`, `useSSEStatus` → `useWSStatus`
- Add exports: `subscribeSession`, `unsubscribeSession`
- Remove: `SSE_STATUS_KEY`

**`ui/src/components/Sidebar.tsx`**:
- `useSSEStatus()` → `useWSStatus()` (lines ~44, ~102)
- Rename `SSEDot` → `ConnectionDot` (line ~43)

**`ui/src/components/LoopState.tsx`**:
- `useSSEStatus()` → `useWSStatus()` (lines ~2, ~5)

**`ui/src/components/ContextPanel.tsx`**:
- Uses `useSessionEvents(session.id)` at line ~83 to derive files touched — no code change needed since hook API is unchanged, but must verify it still works with WS-backed data.

## Data Structures

- No new types — just hook API renames

## Verification

### Static
- `pnpm --filter noodle-ui exec tsc --noEmit`
- `pnpm --filter noodle-ui test`

### E2E smoke tests (full stack)
- `go test ./server/... ./dispatcher/... ./loop/...` — all backend tests pass
- `go vet ./...`
- `sh scripts/lint-arch.sh`
- `pnpm --filter noodle-ui test` — all UI tests pass
- No remaining references to SSE in codebase: `grep -r 'connectSSE\|useSSEStatus\|SSE_STATUS_KEY\|sseHub\|sseClient\|handleSSE' server/ ui/src/` should return nothing

### Runtime
- Open UI, verify sidebar shows "connected" status
- Navigate to agent session — events load immediately (WS backfill) and new events stream in real-time
- Send steer command — verify it works
- Kill backend, verify "disconnected" status appears, restart → auto-reconnect
- Browser devtools: single WS connection, no SSE, no polling requests
