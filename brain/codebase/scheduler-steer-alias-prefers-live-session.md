# Scheduler Steer Alias Prefers Live Session

- Symptom: Scheduler UI steer (`target: "schedule"`) was acknowledged (`status=ok`) but appeared to do nothing while the scheduler session was already running.
- Root cause: `Loop.steer` special-cased `target == "schedule"` to rewrite `orders.json` for a future schedule run, bypassing live session steering.
- Fix: when target is `schedule`, steer the active schedule cook first (interrupt + send for steerable runtimes, respawn fallback for non-steerable); only rewrite orders when no schedule cook is active.
- Regression test: `TestSteerScheduleTargetsActiveScheduleSession` in `loop/cook_steer_test.go`.

See also [[principles/fix-root-causes]], [[principles/experience-first]]
