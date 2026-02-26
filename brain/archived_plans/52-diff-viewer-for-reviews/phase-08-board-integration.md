Back to [[archived_plans/52-diff-viewer-for-reviews/overview]]

# Phase 8: Board integration

## Goal

Wire ReviewCard clicks to open the ReviewPanel in the Board component. This is the final integration phase that connects all the pieces.

## Changes

**`ui/src/components/ReviewCard.tsx`**
- Add `onClick` prop: `onClick?: () => void`
- Attach click handler to the card's outer div
- Ensure clicks on ReviewActions buttons don't propagate to the card click handler (they should still work independently for quick merge/reject without opening the panel)

**`ui/src/components/Board.tsx`**
- Replace `selectedSessionId` state with a single `panelState` discriminated union:
  ```
  PanelState = { type: "chat"; sessionId: string } | { type: "review"; item: PendingReviewItem } | null
  ```
  This ensures only one panel is open at a time — selecting a review item closes the chat panel, and vice versa. Remove `selectedSessionId` and `setSelectedSessionId` entirely.
- Derive `selectedSession` from `panelState.type === "chat"` (same lookup as before).
- Pass `onClick={() => setPanelState({ type: "review", item })}` to each ReviewCard.
- Render `<ReviewPanel>` when `panelState?.type === "review"`, `<ChatPanel>` when `panelState?.type === "chat"`.
- Auto-close: when `panelState?.type === "review"`, check if the selected item is still in `optimisticSnapshot.pending_reviews`. If not (removed by optimistic merge/reject), set `panelState` to `null`. Use a `useEffect` for this.
- **Fix optimistic reducer for `request-changes`:** The current `applyOptimisticSnapshot` reducer (`Board.tsx:66-74`) removes the item from `pending_reviews` for ALL three actions including `request-changes`. But `request-changes` can no-op at max concurrency (`control.go:277`), keeping the item in review. Change the reducer to only remove the item for `merge` and `reject`, NOT for `request-changes`. This prevents the auto-close effect from firing and keeps the panel open so the user sees the item hasn't moved. The next SSE snapshot will reflect the real state.

## Data structures

- `PanelState = { type: "chat"; sessionId: string } | { type: "review"; item: PendingReviewItem } | null`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | State wiring with clear spec, follows existing Board patterns |

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Click a ReviewCard — ReviewPanel opens with diff
- Click a cooking card while ReviewPanel is open — ReviewPanel closes, ChatPanel opens
- Click backdrop or press Escape — panel closes
- Merge from within ReviewPanel — panel closes, item moves to done
- Quick-merge from ReviewCard buttons (without opening panel) — still works
- Resize the ReviewPanel — verify wider default width
- End-to-end: park an agent for review, open the diff, read the changes, merge — the full review workflow
