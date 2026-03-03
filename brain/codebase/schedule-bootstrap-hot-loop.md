# Schedule Bootstrap Hot Loop

- Root cause: `prepareOrdersForCycle` bootstrapped a schedule order whenever there were no non-schedule orders, even if a schedule order already existed.
- This caused `orders.json` rewrites every cycle → fsnotify triggers → rapid log spam.
- Fix: only bootstrap when a schedule order is absent (`!hasScheduleOrder(orders)`).
- Schedule must be exempt from sticky failure semantics — dispatch planning ignores failure blocks for the schedule order.
- Startup invariant: scheduler must be dispatchable immediately after `noodle start`.
  - Reconcile injects a schedule order if missing.
  - Reconcile resets stale schedule stage status `active -> pending` when no live session was recovered.

See also [[principles/fix-root-causes]], [[principles/make-operations-idempotent]]
