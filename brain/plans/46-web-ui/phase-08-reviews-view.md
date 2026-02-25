Back to [[plans/46-web-ui/overview]]

# Phase 8: Reviews View

## Goal

Build the Reviews tab — a list of pending reviews with merge/reject/request-changes actions. Parity with `tui/reviews_tab.go`.

## Changes

- **`ui/src/routes/reviews.tsx`** — Reviews route (`/reviews`). Uses `useSnapshot()` for `pendingReviews`.
- **`ui/src/components/ReviewItem.tsx`** — Review card showing: ID, task type, model, title/summary, worktree path. Selection on click.
- **Action buttons** — Merge, Reject, Request Changes. Each sends a `POST /api/control` with the appropriate action and item ID via `useSendControl()`.
- **Tab badge** — Reviews tab in navigation shows pending count when > 0.
- **Empty state** — "No pending reviews" message.

## Data structures

- Props from `Snapshot.pendingReviews: PendingReviewItem[]`
- Control commands: `{action: "merge", item: id}`, `{action: "reject", item: id}`, `{action: "request-changes", item: id}`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` skill.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Reviews list shows pending items
- Clicking Merge/Reject/Request Changes sends control command and item disappears on next snapshot update
- Tab badge shows count
- Empty state renders correctly
