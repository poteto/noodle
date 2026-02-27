Back to [[archive/plans/46-web-ui/overview]]

# Phase 8: Review Actions

## Goal

Build the review workflow — merge/reject/request-changes actions on review cards in the Review column. Not a separate view — reviews live in the kanban board's Review column.

## Changes

- **`ui/src/components/ReviewCard.tsx`** — Extend the review card from phase 6 with full action buttons: Merge, Reject, Request Changes. Request Changes opens a text input for the prompt.
- **`ui/src/components/ReviewActions.tsx`** — Button group component. Merge is primary (green), Reject is secondary. Each sends `POST /api/control` via `useSendControl()`.
- **Confirmation** — Merge/Reject trigger immediately. Request Changes shows inline text input before submitting.
- **Optimistic update** — Remove card from Review column on action, restore if control command fails.

## Data structures

- Props from `Snapshot.pendingReviews: PendingReviewItem[]` — includes worktree path
- Control commands: `{action: "merge", item: id}`, `{action: "reject", item: id}`, `{action: "request-changes", item: id, prompt: string}`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` and `interaction-design` skills.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Merge/Reject buttons send control commands, card moves to Done on next snapshot
- Request Changes shows text input, submits with prompt
- Failed actions restore the card
