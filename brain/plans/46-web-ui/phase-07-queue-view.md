Back to [[plans/46-web-ui/overview]]

# Phase 7: Queue View

## Goal

Build the Queue tab — a table showing queued work items with status indicators. Parity with `tui/queue.go`.

## Changes

- **`ui/src/routes/queue.tsx`** — Queue route (`/queue`). Uses `useSnapshot()` hook for `queue`, `activeQueueIDs`, `actionNeeded`, `loopState`.
- **`ui/src/components/QueueTable.tsx`** — Table with columns: #, Type, Item, Status. Row selection/highlight on click. Status derived from activeIDs/actionNeeded (cooking, reviewing, ready, planned, no plan).
- **`ui/src/components/QueueProgress.tsx`** — Progress bar header showing "X/Y cooked" with visual bar.
- **Empty state** — Message varies by loop state (idle vs paused vs running with empty queue).

## Data structures

- Queue item status derivation: same logic as `tui/queue.go:statusForItem()` — check `activeIDs`, `actionNeeded`, presence of `Plan`, task type.

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` skill.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Queue table shows items with correct statuses
- Progress bar reflects cooked/total ratio
- Clicking a row selects it (preparation for detail view navigation in phase 9)
- Empty state displays correctly per loop state
