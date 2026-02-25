Back to [[plans/46-web-ui/overview]]

# Phase 5: Shared Client — Types and SSE Hook

## Goal

Build the TypeScript data layer: types mirroring Go structs, SSE client, and React Query hooks. This is the foundation all UI components consume.

## Changes

- **`ui/src/client/types.ts`** — TypeScript interfaces mirroring Go `snapshot.*` types: `Snapshot`, `Session`, `QueueItem`, `EventLine`, `FeedEvent`, `ControlCommand`, `PendingReviewItem`. Derive from the JSON shapes the Go server produces.
- **`ui/src/client/sse.ts`** — `EventSource` wrapper that connects to `/api/events`, parses snapshot JSON, and feeds it into React Query's cache. Handles reconnection on disconnect.
- **`ui/src/client/api.ts`** — REST client functions: `fetchSnapshot()`, `fetchSessionEvents(id)`, `sendControl(cmd)`. Simple `fetch()` wrappers.
- **`ui/src/client/hooks.ts`** — React Query hooks: `useSnapshot()` (backed by SSE stream), `useSessionEvents(id)`, `useSendControl()` (mutation). These are the interface all components use.
- **`ui/src/client/index.ts`** — Barrel export.

## Data structures

- TypeScript `Snapshot` interface — mirrors Go `snapshot.Snapshot` JSON output
- `Session` interface — includes `remoteHost: string | null` for cloud icon display
- `ControlCommand` interface — mirrors `loop.ControlCommand`
- `ControlAck` interface — mirrors Go `ControlAck` (status, message, at)
- `ConfigDefaults` interface — mirrors `GET /api/config` response

## Kanban column derivation

The UI derives kanban columns client-side from the flat snapshot:
- **Queued:** `snapshot.queue` items where ID is NOT in `snapshot.activeQueueIDs`
- **Cooking:** `snapshot.active` sessions
- **Review:** `snapshot.pendingReviews`
- **Done:** `snapshot.recent` sessions (completed + failed)

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — designing the hook API and SSE reconnection logic requires judgment.

## Verification

### Static
- `npm run typecheck` passes
- Types match Go server JSON output (manually compare a `curl /api/snapshot` response against the TypeScript interface)

### Runtime
- Write a minimal test component that renders `useSnapshot()` data as JSON. Verify it updates live when `.noodle/` files change.
