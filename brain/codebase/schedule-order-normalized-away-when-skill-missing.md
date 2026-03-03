# Schedule Order Normalized Away When Skill Missing

- Symptom: `noodle start` logs `startup injected schedule order`, then `orders normalized`, then flips `state changed from=running to=idle` and `state changed from=idle to=running` every cycle.
- Runtime state signature:
  - `.noodle/orders.json` remains empty (`"orders": []`)
  - `.noodle/status.json` shows `"loop_state": "idle"`
  - no active sessions under `.noodle/sessions/`
- Root cause:
  - Startup reconcile always injects a `schedule` order when missing.
  - The first cycle normalizes orders before dispatch.
  - Order normalization drops stages whose task type cannot be resolved in the registry.
  - If `schedule` skill is not discoverable, the injected schedule stage is removed, so the order disappears before `spawnSchedule` can trigger bootstrap logic.
- Consequence: scheduler bootstrap path is bypassed, so initialization appears stuck, especially when backlog is empty.
- Related code paths:
  - `loop/reconcile.go` (`ensureScheduleOrderPresent`)
  - `loop/loop_cycle_pipeline.go` (`prepareOrdersForCycle` -> `normalizeOrders` before scheduling)
  - `internal/orderx/orders.go` (`NormalizeAndValidateOrders` stage filtering)

## Fix (2026-03-03)

- Startup reconcile now injects an `oops` bootstrap order (`oops-bootstrap-schedule`) when `schedule` task type is unavailable but `oops` exists.
- The bootstrap order prompt points to `github.com/poteto/noodle/.agents/skills/schedule/` as the reference example and instructs creation of `.agents/skills/schedule/SKILL.md`.
- When that bootstrap order completes, the loop refreshes skill resolution and injects a real `schedule` order immediately (without waiting for backlog changes), enabling recovery on the next cycle.
- Schedule-order insertion order on startup remains append-only to avoid changing existing dispatch priority behavior.

## Follow-up Fix (2026-03-03)

- Bare repo edge case remained: with neither `schedule` nor `oops` task type registered, startup injected `schedule` but two later paths could still remove it before bootstrap settled:
  - `NormalizeAndValidateOrders` filtered unknown task types.
  - `auditOrders()` dropped orders with no resolvable stages after registry rebuild.
- Added a foundational exception for scheduler bootstrap:
  - Preserve order `id == "schedule"` stage `task_key == "schedule"` through normalization and audit even when registry lacks schedule.
  - This keeps the bootstrap lane alive so `spawnSchedule` can dispatch built-in bootstrap (`bootstrap-schedule`), avoiding `running/idle` churn in empty repos.
