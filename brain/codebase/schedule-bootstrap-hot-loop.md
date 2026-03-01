# Schedule Bootstrap Hot Loop

- Root cause: `prepareOrdersForCycle` bootstrapped a schedule order whenever there were no non-schedule orders, even if a schedule order already existed.
- In the failure case (`failedTargets["schedule"]` set), dispatch skipped schedule, but bootstrap still rewrote `orders.json` every cycle.
- Rewriting `orders.json` triggered fsnotify, which immediately triggered another cycle, causing rapid log spam: `"orders empty, bootstrapping schedule"`.
- Fix: only bootstrap when a schedule order is absent (`!hasScheduleOrder(orders)`), so existing schedule state is preserved and the write loop is eliminated.
- Related UX polish: startup now normalizes loopback display URLs to `localhost` (`localhost:3000`) while still binding loopback addresses internally.

See also [[principles/fix-root-causes]], [[principles/make-operations-idempotent]]
