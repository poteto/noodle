# Failed Target Auto Repair For Requeued Orders

- `.noodle/failed.json` and in-memory `failedTargets` were removed.
- `orders.json` is now the single source of truth for terminal stage state:
  - `failStage` marks the active/pending/merging stage as `failed`.
  - The parent order is marked `failed` and retained in `orders.json` (not deleted).
- Dispatch only considers `order.status == active`, so failed orders do not run until explicit requeue.
- `control requeue` now repairs state directly in `orders.json`:
  - sets order back to `active`
  - resets failed/cancelled stages to `pending`
  - emits `order.requeued`
- `consumeOrdersNext` duplicate-ID merge now supports scheduler-driven restart for failed orders:
  - default remains idempotent duplicate-skip for crash safety
  - exception: incoming active order replaces existing failed order with the same ID
  - this lets scheduler re-schedule the same item ID via `orders-next.json` without requiring a separate control command
- This removes split-brain risk between queue state and a secondary failed-target ledger.

See also [[codebase/runtime-routing-owned-by-orders]], [[archive/plans/34-failed-target-reset-runtime/overview]], [[principles/make-operations-idempotent]]
