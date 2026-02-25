Back to [[plans/46-web-ui/overview]]

# Phase 9: Session Detail View

## Goal

Build the session detail view — a scrollable event timeline for a selected session. Parity with the detail view in `tui/model_render.go:renderActorDetail`.

## Changes

- **`ui/src/routes/session.$id.tsx`** — Session detail route (`/session/:id`). Uses `useSessionEvents(id)` hook to fetch events. Header shows session metadata (name, status, model, duration, cost, context %).
- **`ui/src/components/EventTimeline.tsx`** — Scrollable list of events. Each event shows timestamp, label (colored by category: tool/think/cost/ticket), and body text. Auto-scrolls to bottom for active sessions. Manual scroll disables auto-scroll.
- **`ui/src/components/SessionHeader.tsx`** — Session metadata bar: health dot, display name, status, model, duration, cost, context window progress.
- **Navigation** — Clicking an agent card in Feed or a queue row navigates to this view. Back button returns to previous tab.

## Data structures

- `EventLine` props: `at: string`, `label: string`, `body: string`, `category: string`
- Session metadata from `Snapshot.sessions` filtered by ID

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` skill. Auto-scroll behavior needs judgment.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Clicking an agent card navigates to detail view
- Events stream in and timeline scrolls to bottom for active sessions
- Scrolling up disables auto-scroll; scrolling to bottom re-enables it
- Back navigation works
- Event labels colored by category
