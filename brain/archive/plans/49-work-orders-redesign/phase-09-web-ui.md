Back to [[archive/plans/49-work-orders-redesign/overview]]

# Phase 9: Web UI

## Goal

Update the React web UI to render orders with stage pipelines instead of flat queue cards. Use `react-best-practices` and `ts-best-practices` skills.

## Changes

**`ui/src/client/types.ts`** — Replace types:
- Delete `QueueItem` interface
- Add `Order` interface: `{id, title?, plan?, rationale?, stages: Stage[], on_failure?: Stage[], status}`
- Add `Stage` interface: `{task_key?, prompt?, skill?, provider?, model?, runtime?, status, extra?: Record<string, unknown>}` (provider/model optional — they may be absent in `orders.json` before `ApplyOrderRoutingDefaults` runs; by snapshot time they should be populated, but the TS type should match the JSON schema)
- Update `Snapshot` interface: replace `queue: QueueItem[]` with `orders: Order[]`, replace `active_queue_ids` with `active_order_ids`
- Update `ControlCommand` payload: replace `item_id` with `order_id` in any TypeScript types/functions that send control commands to the server (must match the Go rename in phase 6)

**`ui/src/components/Board.tsx`** — Update kanban derivation:
- `deriveKanbanColumns()` operates on `snapshot.orders` instead of `snapshot.queue`
- "Queued" column: orders with first pending stage that aren't active
- "Cooking" column: orders with an active stage (in `active_order_ids`)
- "Review" column: pending reviews. **Display `PendingReviewItem.Reason`** — this is the primary new information the human needs (e.g., "merge conflict", "quality rejected", "approval required"). Show the reason prominently on the review card. Empty reason means normal completion review (no special label needed).
- "Done" column: recent completed sessions (unchanged)

**`ui/src/components/QueueCard.tsx`** → rename to **`OrderCard.tsx`**:
- Renders an Order with its stage pipeline
- Show order title/ID at top
- Render stages as a horizontal pipeline: `execute ✓ → quality ● → reflect ○`
- Stage status indicators: completed (checkmark), active (filled dot), pending (empty dot), failed (x), cancelled (dash)
- For `"failing"` orders (running OnFailure stages): show main pipeline with failure indicator, then OnFailure pipeline below/after. Visual separation between main stages and failure-routing stages.
- Badge shows task_key of the current active/next-pending stage
- Show model on the active stage

**`ui/src/components/Badge.tsx`** — No changes needed (already handles task_key strings).

## Data structures

- TypeScript `Order` and `Stage` interfaces mirror Go types
- Pipeline rendering is derived from `stages` array and each stage's `status`

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

UI design judgment for pipeline rendering. Use `react-best-practices`, `ts-best-practices`, `interaction-design` skills.

## Verification

### Static
- `npm run typecheck` passes (ui directory)
- No remaining references to `QueueItem` or `item_id` in TypeScript

### Runtime
- Manual: start noodle with orders.json containing a multi-stage order, verify Board shows pipeline
- Manual: verify active order shows correct stage highlighted
- Manual: verify completed stages show checkmarks, pending show empty dots
- Manual: verify single-stage orders (schedule, meditate) render cleanly without pipeline clutter
- Manual: verify `"failing"` order shows OnFailure stages with visual distinction from main pipeline
- Manual: verify review card displays `Reason` prominently (merge conflict, quality rejected)
- Manual: verify review card without Reason shows no spurious label
- Automated: `npm run typecheck` covers type safety. Add a component test for `OrderCard` rendering the pipeline from a fixture `Order` object (prevents regressions in stage status → indicator mapping after future refactors).
- Automated: component test for review card rendering `Reason` field
- Integration test note: verify snapshot API → Board column derivation → OrderCard rendering end-to-end in phase 10
