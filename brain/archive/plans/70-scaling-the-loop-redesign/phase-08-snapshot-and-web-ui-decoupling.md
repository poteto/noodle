Back to [[archive/plans/70-scaling-the-loop-redesign/overview]]

# Phase 8: Snapshot and web UI decoupling

## Goal

The web UI reads from in-memory state maintained by the HTTP server, not from session files on disk. `snapshot.LoadSnapshot()` no longer does O(n) directory scans. Session files become write-mostly artifacts — written by runtimes and only read on non-hot paths (per-session events, runtime health observation). Server and UI ship together.

## Changes

**`internal/snapshot/snapshot.go`** — `LoadSnapshot()` (currently at `snapshot.go:26`) accepts a `LoopState` argument rather than scanning `.noodle/sessions/` via `sessionmeta.ReadAll()` (`snapshot.go:128`). The session list comes from the loop's active cook tracking + recent history. Canonical event reading for the chat panel is done lazily per-session on demand (when the user opens a specific session). Note: `LoadSnapshot()` currently also reads all session events (`snapshot.go:40-52`) — this bulk read is eliminated; events are fetched per-session on demand.

**`server/server.go` and `server/sse.go`** — The HTTP server holds a reference to the loop via its `State()` method. The `/api/snapshot` endpoint calls `loop.State()` and returns the result. The `/api/sessions/{id}/events` endpoint already exists (`server.go:77`, handler at `:159-191`) — refactor it to lazy per-session read from `LoopState` + on-demand canonical file read, instead of loading the full snapshot. Update all `snapshot.LoadSnapshot()` call sites (find via grep for `LoadSnapshot`).

**Server-loop lifecycle wiring**: The server currently starts independently with only `RuntimeDir` (`cmd_start.go:91-105`) and no loop reference. `startRuntimeLoop` interface (`cmd_start.go:18-22`) lacks a `State()` method. Add a `LoopStateProvider` interface with `State() LoopState` method. The server receives this interface at construction. Define startup ordering: loop starts first (owns state), server starts second (reads state). Shutdown: server stops first (no more reads), loop stops second.

**`loop/loop.go`** — Export a `State()` method that returns a read-only snapshot of current loop state: orders, active cooks, recent completions, pending reviews. Contract: `State()` returns an immutable detached DTO (deep-copied map/slice fields, no internal references). Publish snapshot via `atomic.Pointer[LoopState]` at cycle boundaries so HTTP/SSE readers are lock-free and cannot race loop writes.

**`ui/src/client/types.ts`** — Update TypeScript types to match the new `LoopState` shape. Current types reference `sessions`, `active_order_ids`, `pending_reviews` etc. Map these to the new structure. This is a coordinated server + UI change — both must ship together.

**`ui/src/components/Board.tsx`** and related components — Update to consume the new snapshot shape. Full component inventory that consumes snapshot/session data:
- `Board.tsx` — main board layout, `deriveKanbanColumns()`
- `BoardHeader.tsx` — `max_cooks`, `loop_state`, `autonomy`
- `DoneCard.tsx` — recent/completed session rendering
- `ChatPanel.tsx` — session events display
- `client/types.ts` — `Snapshot`, `Session`, `KanbanColumns`, `deriveKanbanColumns()`
- Optimistic reducers that update snapshot state

Session data that was previously per-session metadata (`Session` interface fields) is now derived from `CookSummary` entries in the loop state. Publish a precise API diff documenting old→new JSON field mapping.

**`ui/src/components/AgentCard.tsx`** — Add stage visualization via a side rail with dots. Design:

```
┌──────────────────────────────┐
│ ● │ feat(auth): Add OAuth flow │
│ ● │ execute · tmux             │
│ ◉ │ ░░░░░░░▓▓▓▓ 2m 34s        │
│ ○ │                            │
│ ○ │                            │
└──────────────────────────────┘
```

A vertical dot rail on the left edge of each card. Each dot represents one stage in the order. Color coding: green filled (●) = completed, yellow pulsing (◉) = active, gray outline (○) = pending, red (✗) = failed. The rail is always visible — no hover or expand needed. For orders with many stages (>8), the rail compresses with an ellipsis dot. The active dot pulses subtly to draw the eye without being distracting.

Each dot is clickable — clicking a stage dot opens that stage's session in the side panel (chat view). The active stage opens the live session; completed stages open the session's canonical event history. This gives direct access to any stage's work without navigating away from the board. **Prerequisite**: per-stage session ID linkage. The current `Stage` type (`ui/src/client/types.ts:53-62`) does not carry a `session_id` field. Add `session_id?: string` to the `Stage` interface (Go side: add `SessionID` to the stage struct or `Extra` field, populated on dispatch and preserved through completion). Without this, clicking a dot can't route to the correct session.

Implementation: a `StageRail` component that receives the order's stages array and the active stage index. Each dot is a button with `onClick` → `openSession(stage.sessionId)`. Uses the existing poster aesthetic (black/yellow/white palette, 4px card shadows). The rail sits in a narrow left column (~16px) with the existing card content in the right column.

**Session files** (`.noodle/sessions/{id}/`) — Still written by runtimes (meta.json, canonical.ndjson, spawn.json). After this phase, session files are not read on the `/api/snapshot` hot path. They are still read: (a) on demand by `/api/sessions/{id}/events` for the chat panel, (b) by the runtime observation layer (phase 6) for health monitoring. They're write-mostly artifacts — hot-path reads are eliminated but they're not purely debugging artifacts.

**Internal sequencing**: (a) Add `LoopStateProvider` interface + implement immutable `State()` publication via `atomic.Pointer`; (b) add per-stage session ID linkage (Go + TS types); (c) wire server to `LoopStateProvider`, update lifecycle ordering; (d) update `snapshot.LoadSnapshot()` to accept `LoopState`, refactor server endpoints; (e) publish old→new API field mapping; (f) update TypeScript types to match new shape; (g) update all consuming components (Board, BoardHeader, DoneCard, ChatPanel, reducers); (h) add `StageRail` component.

## Data structures

- `LoopState` — exported struct: `Orders []Order`, `ActiveCooks []CookSummary`, `PendingReviews []PendingReviewItem`, `PendingReviewCount int`, `RecentHistory []HistoryItem`, `Status LoopStatus`, `ActiveSummary ActiveSummary`, `ActiveOrderIDs []string`, `TotalCostUSD float64`, `MaxCooks int`, `Autonomy string`, `ActionNeeded []string` — all fields currently in `Snapshot` (`ui/src/client/types.ts:8-24`) must be present or derivable from `LoopState`
- `CookSummary` — `SessionID`, `OrderID`, `TaskKey`, `Skill`, `Runtime`, `Provider`, `Model`, `StartedAt`, `DisplayName`
- `StageRail` (React component) — receives `stages: Stage[]`, `activeIndex: number`, renders vertical dot rail

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — coordinated server + UI change requires judgment about the snapshot contract and what the UI actually needs

## Verification

### Static
- `go test ./...` — all tests pass
- `sessionmeta.ReadAll()` not called from snapshot `LoadSnapshot()` path
- `ReadDir` of `.noodle/sessions/` only happens in runtime observation (phase 6) and on-demand event loading, not in `/api/snapshot`
- TypeScript types compile cleanly (`pnpm -C ui typecheck`)
- All current `Snapshot` fields (`types.ts:8-24`) are present or derivable in new `LoopState`
- `LoopStateProvider` interface wired in server with correct lifecycle ordering
- `State()` deep-copy contract test: mutating returned DTO in test does not mutate loop-owned state

### Runtime
- Web UI loads dashboard, verify session list matches loop state
- Open agent chat panel, verify events load on demand via `/api/sessions/{id}/events`
- Kill a session, verify it disappears from snapshot within 1-2 cycles
- With 100+ completed sessions in history, verify `/api/snapshot` responds in <50ms
- SSE stream delivers updates from in-memory state, not file reads
- Load test: 50 concurrent `/api/snapshot` requests do not increase cycle p99 beyond baseline regression budget
- Stage rail: cards show correct dot count matching order stages, active dot pulses, completed dots are green
- Stage rail: order with 3 completed + 1 active + 2 pending stages renders ● ● ● ◉ ○ ○ correctly
- Stage rail: orders with >8 stages compress with ellipsis
- Stage rail: clicking a completed dot opens that stage's session history in the side panel
- Stage rail: clicking the active dot opens the live session in the side panel
