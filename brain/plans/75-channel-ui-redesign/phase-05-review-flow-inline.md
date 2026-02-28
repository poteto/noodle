Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 5: Review Flow Inline

## Goal

When an agent's work needs review (pending_reviews), surface it in the channel UI. The agent's conversation feed shows a review prompt at the bottom, and the context panel switches to show the diff. Action buttons (merge/reject/request changes) appear inline.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- UX decisions on how review integrates with conversation flow

## Changes

### Create
- `ui/src/components/ReviewBanner.tsx` — inline banner in agent feed when review is pending: "Ready for review" with merge/reject/request-changes buttons
- `ui/src/components/DiffViewer.tsx` — renders worktree diff in context panel with syntax highlighting (reuse existing `@wooorm/starry-night` dependency)

### Modify
- `ui/src/components/AgentFeed.tsx` — append ReviewBanner when agent's session has a pending review
- `ui/src/components/ContextPanel.tsx` — when review is pending, show DiffViewer instead of metrics
- `ui/src/client/hooks.ts` — `useReviewDiff(reviewId)` already exists, wire to DiffViewer

### Delete
- `ui/src/components/ChatPanel.tsx` — old side panel (replaced by AgentFeed)
- `ui/src/components/ReviewPanel.tsx` — old review drawer (replaced by inline review)

## Data Structures

- Match pending review to agent by `PendingReviewItem.session_id` or `order_id`
- Review actions map to existing control commands: `merge`, `reject`, `request-changes`

## Tests

- `ReviewBanner.test.tsx` — renders when review pending, merge/reject/request-changes buttons send correct control commands
- `DiffViewer.test.tsx` — renders diff lines with added/removed/context styling

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass, no references to deleted ChatPanel/ReviewPanel

### Runtime
- Agent with pending review shows review banner with action buttons
- Context panel shows diff with syntax-highlighted additions/removals
- Click "Merge" → sends control command → review clears → order advances
- Click "Reject" → sends control command → agent's work is rejected
