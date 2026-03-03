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

## Follow-up Fix 2 (2026-03-03)

- New symptom after startup loop fix: Noodle no longer churned `idle/running`, but UI looked stuck during bootstrap with little visible progress.
- Root cause:
  - `spawnSchedule()` called `ensureSkillFresh("schedule")` every cycle while bootstrap was already in flight.
  - Each `ensureSkillFresh` call forced a full registry rebuild and emitted `registry.rebuilt`.
  - In active loops this produced event spam (`loop-events.ndjson`) that obscured meaningful progress and made the scheduler channel look broken/noisy.
- Backend fix:
  - In `spawnSchedule`, skip `ensureSkillFresh` when the `schedule` task type is missing but `bootstrapInFlight` is already set.
  - Keep one initial rebuild, avoid repeated rebuild churn until bootstrap completes.
  - Regression test: `TestBootstrapInFlightDoesNotRebuildRegistryEveryCycle`.
- Watcher hardening:
  - `skill/watcher.go` now filters fsnotify events to configured skill search paths (and their ancestors for path creation), avoiding unrelated parent-watch churn.
  - Regression test: `TestWatcherIgnoresSiblingEventsWhenSearchPathMissing`.
- UI follow-up:
  - Sidebar now surfaces schedule bootstrap as a visible order during active bootstrap sessions.
  - Scheduler feed now shows explicit status text: `Bootstrapping schedule skill...`.
