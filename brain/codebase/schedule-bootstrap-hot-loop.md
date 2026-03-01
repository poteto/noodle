# Schedule Bootstrap Hot Loop

- Root cause: `prepareOrdersForCycle` bootstrapped a schedule order whenever there were no non-schedule orders, even if a schedule order already existed.
- In the failure case (`failedTargets["schedule"]` set), dispatch skipped schedule, but bootstrap still rewrote `orders.json` every cycle.
- Rewriting `orders.json` triggered fsnotify, which immediately triggered another cycle, causing rapid log spam: `"orders empty, bootstrapping schedule"`.
- Fix: only bootstrap when a schedule order is absent (`!hasScheduleOrder(orders)`), so existing schedule state is preserved and the write loop is eliminated.
- Follow-up: schedule must be exempt from sticky failed-target semantics.
  - `failed.json` entries for `"schedule"` are ignored on load.
  - `markFailed("schedule", ...)` is a no-op.
  - Dispatch planning ignores failed-target blocks for the schedule order.
  - Added directory fixture: `loop/testdata/schedule-failed-target-does-not-block-dispatch`.
- Startup invariant: scheduler must be dispatchable immediately after `noodle start`.
  - Reconcile now injects a schedule order if missing.
  - Reconcile resets stale schedule stage status `active -> pending` when no live session was recovered for `schedule`.
  - Cycle idle gating no longer suppresses dispatch when a schedule order already exists and backlog is empty.
- Related UX polish: startup now normalizes loopback display URLs to `localhost` (`localhost:3000`) while still binding loopback addresses internally.

See also [[principles/fix-root-causes]], [[principles/make-operations-idempotent]]
