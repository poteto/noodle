# Treeview Schedule Node Routes To Live Feed

- Symptom: clicking a `schedule` stage node in topology opened `/actor/<scheduler-session-id>` instead of returning to the scheduler live feed.
- Root cause: tree node click routing treated every stage with `session_id` as an agent channel target.
- Fix: classify topology stage click targets by task key; `schedule` routes to the scheduler channel (`/`), non-scheduler stages route to `/actor/$id`.
- Regression test: `ui/src/components/TreeView.interaction.test.tsx` (`routes schedule-linked nodes back to live feed`).

See also [[codebase/scheduler-steer-alias-prefers-live-session]], [[principles/experience-first]]
