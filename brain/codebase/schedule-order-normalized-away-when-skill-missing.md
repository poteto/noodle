# Schedule Order Normalized Away When Skill Missing

- Symptom: `noodle start` injects a schedule order, normalization removes it (unknown task type), loop churns `running/idle` every cycle.
- Root cause: startup reconcile injects `schedule` → `NormalizeAndValidateOrders` drops stages with unresolvable task types → order disappears before bootstrap can trigger.
- Fix: foundational exception preserves order `id == "schedule"` with stage `task_key == "schedule"` through normalization and audit even when registry lacks the skill. This keeps the bootstrap lane alive for `spawnSchedule`.
- Fallback: when `schedule` task type is unavailable but `oops` exists, startup injects an `oops-bootstrap-schedule` order to create the schedule skill.
- When `bootstrapInFlight`, skip `ensureSkillFresh` to avoid repeated registry rebuild churn.

See also [[principles/fix-root-causes]], [[principles/make-operations-idempotent]]
