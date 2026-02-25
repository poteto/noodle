Back to [[archived_plans/46-web-ui/overview]]

# Phase 6: Kanban Board

## Goal

Build the main board view — a kanban layout with Queued → Cooking → Review → Done columns. This is the single primary view, not one of many tabs. Design matches `ui_prototype/board.html`.

## Changes

- **`ui/src/routes/index.tsx`** — Board route (default `/`). Uses `useSnapshot()` hook. Derives kanban columns client-side (see phase 5 column mapping).
- **`ui/src/components/Board.tsx`** — Four-column kanban layout. Header with title, stats bar, loop state indicator, new task button.
- **`ui/src/components/BoardColumn.tsx`** — Column with title, count badge, scrollable card list. Columns: Queued, Cooking, Review, Done.
- **`ui/src/components/AgentCard.tsx`** — Card for active/recent sessions. Shows: type badge, name, task description, context progress bar, duration, cost, model tag. Remote agents show cloud icon with host tooltip. Clickable — opens chat panel (phase 9).
- **`ui/src/components/QueueCard.tsx`** — Card for queued items. Shows: type badge, name, task description. Simpler than agent cards (no progress/cost).
- **`ui/src/components/ReviewCard.tsx`** — Card in Review column. Shows: type badge, name, task, merge/reject buttons inline.
- **`ui/src/components/Badge.tsx`** — Task type badge (execute, plan, review, reflect, schedule) with per-type warm palette colors.
- **Stats bar** — Active count, done count, failed count, total cost, loop state pulse.

## Data structures

- Kanban column derivation from `Snapshot` (see phase 5)
- `AgentCard` props: `session: Session` — includes `remoteHost` for cloud icon
- `QueueCard` props: `item: QueueItem`
- `ReviewCard` props: `review: PendingReviewItem`, `onMerge`, `onReject`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` and `interaction-design` skills.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Board shows four columns with correct card placement
- Cards update live as sessions change via SSE
- Remote agents show cloud icon, hover shows host name
- Clicking cooking/done agent cards opens chat panel
- Review cards have working merge/reject buttons
- Empty columns show gracefully
