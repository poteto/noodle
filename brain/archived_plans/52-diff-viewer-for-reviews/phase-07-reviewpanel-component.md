Back to [[archived_plans/52-diff-viewer-for-reviews/overview]]

# Phase 7: ReviewPanel component

## Goal

Create the ReviewPanel that composes SidePanel, DiffViewer, and ReviewActions into a complete review experience. This is the panel that opens when clicking a ReviewCard.

## Changes

**`ui/src/components/ReviewPanel.tsx`** (new file)
- Props: `item: PendingReviewItem`, `onClose: () => void`
- Structure:
  1. `<SidePanel defaultWidth={800} onClose={onClose}>` — wider default than ChatPanel's 560px since code needs horizontal space
  2. **Header** (fixed, not scrollable): review item title, badge, worktree label, model tag — similar layout to ChatPanel header but with review-specific metadata
  3. **DiffViewer** (scrollable, flex-1): fetches diff using `useReviewDiff(item.id)` and renders the stat + diff
  4. **ReviewActions** (fixed at bottom): merge/reject/request-changes buttons. Add an `onAction?: (action: string) => void` callback prop to `ReviewActions` (currently it has no callback). ReviewPanel passes a callback that closes the panel on `merge` and `reject`, but NOT on `request-changes` — request-changes can no-op at max concurrency (`control.go:277-280`), keeping the item in review. The panel stays open so the user sees the item hasn't moved.
- The panel should feel like a focused review workspace: header gives context, diff is the main content, actions are always visible at the bottom.

**`ui/src/components/ReviewActions.tsx`** (modification)
- Add optional `onAction?: (action: string) => void` prop
- Call `onAction(action)` after each `send()` call (merge, reject, request-changes)

## Data structures

- `ReviewPanelProps { item: PendingReviewItem; onClose: () => void }`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Composition of existing components with clear layout spec |

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Render ReviewPanel with a real pending review item — verify header, diff, and actions all render
- Verify the panel is wider than ChatPanel (~800px vs 560px)
- Take an action (merge) — verify panel closes and item disappears from review column
- Verify request-changes flow works within the panel (feedback input appears, can submit)
