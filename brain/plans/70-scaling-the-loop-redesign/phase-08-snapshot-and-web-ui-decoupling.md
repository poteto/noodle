Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 8: Snapshot and web UI decoupling

## Goal

The web UI reads from in-memory state maintained by the HTTP server, not from session files on disk. `snapshot.LoadSnapshot()` no longer does O(n) directory scans. Session files become write-only debugging artifacts — written by runtimes for post-mortem inspection, never read on the hot path. Server and UI ship together.

## Changes

**`internal/snapshot/snapshot.go`** — `LoadSnapshot()` accepts a `LoopState` argument rather than scanning `.noodle/sessions/`. The session list comes from the loop's active cook tracking + recent history. Canonical event reading for the chat panel is done lazily per-session on demand (when the user opens a specific session).

**`server/server.go` and `server/sse.go`** — The HTTP server holds a reference to the loop via its `State()` method (RWMutex-protected read). The `/api/snapshot` endpoint calls `loop.State()` and returns the result. A new `/api/sessions/{id}/events` endpoint serves canonical events for a specific session on demand (lazy file read). Update all `snapshot.LoadSnapshot()` call sites (`server.go:151,165`, `sse.go:134,189`).

**`loop/loop.go`** — Export a `State()` method that returns a read-only snapshot of current loop state: orders, active cooks, recent completions, pending reviews. Protected by a RWMutex (loop writes under write lock at cycle boundaries, server reads under read lock on HTTP requests).

**`ui/src/client/types.ts`** — Update TypeScript types to match the new `LoopState` shape. Current types reference `sessions`, `active_order_ids`, `pending_reviews` etc. Map these to the new structure. This is a coordinated server + UI change — both must ship together.

**`ui/src/components/Board.tsx`** and related components — Update to consume the new snapshot shape. Session data that was previously per-session metadata is now derived from `CookSummary` entries in the loop state.

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

Each dot is clickable — clicking a stage dot opens that stage's session in the side panel (chat view). The active stage opens the live session; completed stages open the session's canonical event history. This gives direct access to any stage's work without navigating away from the board.

Implementation: a `StageRail` component that receives the order's stages array and the active stage index. Each dot is a button with `onClick` → `openSession(stage.sessionId)`. Uses the existing poster aesthetic (black/yellow/white palette, 4px card shadows). The rail sits in a narrow left column (~16px) with the existing card content in the right column.

**Session files** (`.noodle/sessions/{id}/`) — Still written by runtimes (meta.json, canonical.ndjson, spawn.json). But no code in the loop or snapshot hot path reads them. They're debugging artifacts and post-mortem data.

**Internal sequencing**: (a) Implement `State()` method on loop with RWMutex; (b) update `snapshot.LoadSnapshot()` to accept `LoopState`, update server endpoints; (c) add `/api/sessions/{id}/events` endpoint; (d) update TypeScript types to match new shape; (e) update Board.tsx and AgentCard.tsx components; (f) add `StageRail` component.

## Data structures

- `LoopState` — exported struct: `Orders []Order`, `ActiveCooks []CookSummary`, `PendingReviews []PendingReviewItem`, `RecentHistory []HistoryItem`, `Status LoopStatus`, `ActiveSummary ActiveSummary`
- `CookSummary` — `SessionID`, `OrderID`, `TaskKey`, `Skill`, `Runtime`, `Provider`, `Model`, `StartedAt`, `DisplayName`
- `StageRail` (React component) — receives `stages: Stage[]`, `activeIndex: number`, renders vertical dot rail

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — coordinated server + UI change requires judgment about the snapshot contract and what the UI actually needs

## Verification

### Static
- `go test ./...` — all tests pass
- `sessionmeta.ReadAll()` not called from snapshot package
- `ReadDir` of `.noodle/sessions/` only happens in runtime observation (phase 6), not in snapshot or server
- TypeScript types compile cleanly (`pnpm -C ui typecheck`)

### Runtime
- Web UI loads dashboard, verify session list matches loop state
- Open agent chat panel, verify events load on demand via `/api/sessions/{id}/events`
- Kill a session, verify it disappears from snapshot within 1-2 cycles
- With 100+ completed sessions in history, verify `/api/snapshot` responds in <50ms
- SSE stream delivers updates from in-memory state, not file reads
- Stage rail: cards show correct dot count matching order stages, active dot pulses, completed dots are green
- Stage rail: order with 3 completed + 1 active + 2 pending stages renders ● ● ● ◉ ○ ○ correctly
- Stage rail: orders with >8 stages compress with ellipsis
- Stage rail: clicking a completed dot opens that stage's session history in the side panel
- Stage rail: clicking the active dot opens the live session in the side panel
