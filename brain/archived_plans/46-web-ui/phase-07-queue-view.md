Back to [[archived_plans/46-web-ui/overview]]

# Phase 7: Board Header and Stats

## Goal

Build the board header bar — title, live stats, loop state indicator, and new task button. The header is the top-level control surface.

## Changes

- **`ui/src/components/BoardHeader.tsx`** — Header with: "noodle" title (Outfit 800, 3.5rem), stats badges (cooking count, done count, failed count, total cost), loop state pulse indicator, new task button.
- **`ui/src/components/LoopState.tsx`** — Animated pulse dot + state label. Reflects current loop state (running/paused/draining/idle).
- **`ui/src/components/StatBadge.tsx`** — Inverted badge (dark bg, accent text) showing a stat value.
- **New task button** — Opens task editor modal (wired in phase 10).

## Data structures

- Stats derived from `Snapshot`: `active.length`, `recent` filtered by status, `totalCostUSD`, `loopState`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` and `interaction-design` skills.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Stats update live as sessions change
- Loop state indicator pulses when running, stops when paused
- New task button triggers modal (or placeholder click handler)
