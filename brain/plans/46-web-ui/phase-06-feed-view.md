Back to [[plans/46-web-ui/overview]]

# Phase 6: Feed View

## Goal

Build the Feed tab — the default view showing agent cards and a live event timeline. Parity with `tui/feed.go` + `tui/feed_item.go`.

## Changes

- **`ui/src/routes/index.tsx`** — Feed route (default `/`). Uses `useSnapshot()` hook. Renders agent cards for active and recent sessions.
- **`ui/src/components/AgentCard.tsx`** — Individual agent card showing: health indicator, display name, task type badge, model, last action, context window progress bar, duration, cost. Invoke `frontend-design` for styling.
- **`ui/src/components/ProgressBar.tsx`** — Context window usage bar. Color shifts at 75%/95% thresholds.
- **`ui/src/components/Badge.tsx`** — Task type badge (execute, plan, review, reflect, prioritize) with per-type colors.
- **Stats footer** — Active count, queued count, pending reviews, loop state.

## Data structures

- Component props derived from `Snapshot.active` and `Snapshot.recent` arrays
- `AgentCard` props: `session: Session`, `lastAction: string`, `lastLabel: string`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` skill for component styling.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Feed shows live agent cards that update as sessions change
- Cards show correct health colors, progress bars, cost, duration
- Empty state renders when no sessions exist
