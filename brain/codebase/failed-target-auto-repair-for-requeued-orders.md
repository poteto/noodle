# Failed Target Auto Repair For Requeued Orders

- `orders.json` is the single source of truth for terminal stage state (no separate `failed.json` or in-memory `failedTargets`).
- `failStage` marks the active/pending/merging stage as `failed`; parent order is marked `failed` and retained.
- Dispatch only considers `order.status == active`, so failed orders are inert until explicit requeue.
- `control requeue` repairs state directly: sets order to `active`, resets failed/cancelled stages to `pending`, emits `order.requeued`.
- `consumeOrdersNext` supports scheduler-driven restart: incoming `active` order replaces existing `failed` order with the same ID (default remains idempotent duplicate-skip for crash safety).

See also [[codebase/runtime-routing-owned-by-orders]], [[principles/make-operations-idempotent]], [[principles/boundary-discipline]]
