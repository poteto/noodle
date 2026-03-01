# Failed Target Auto Repair For Requeued Orders

- Sticky failed-target entries in `.noodle/failed.json` can block dispatch when the scheduler reintroduces an order with the same `order_id`.
- Loop cycle repair now clears those stale failed-target entries whenever a matching non-schedule order exists in `orders.json`.
- Repair is persisted atomically to `failed.json` before mutating in-memory state.
- For operator visibility, each cleared target emits `order.requeued` in `loop-events.ndjson`, which surfaces in UI/feed.

See also [[codebase/runtime-routing-owned-by-orders]], [[archive/plans/34-failed-target-reset-runtime/overview]], [[principles/make-operations-idempotent]]
