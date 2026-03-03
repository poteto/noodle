# Scheduler Active Order Amendments

- `consumeOrdersNext` now treats duplicate `active` order IDs as amendable, not skip-only.
- Active-order amendment merge behavior:
  - preserve completed prefix from the existing order
  - if the current stage definition is unchanged, keep the running stage and replace only downstream stages
  - if the current stage definition changed, replace the current stage from scheduler input
- After promotion, the loop now compares each active cook against the promoted order's current stage:
  - if stage definitions no longer match (or the order disappeared), the cook is force-killed and worktree-cleaned
  - this prevents stale sessions from continuing after scheduler-amended pipelines

See also [[codebase/scheduler-steer-alias-prefers-live-session]], [[codebase/orders-lifecycle-defaults-on-promotion]], [[principles/make-operations-idempotent]], [[principles/serialize-shared-state-mutations]]
