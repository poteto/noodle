# Sidebar Stage List Labels and Visibility

- Stage labels in `ui/src/components/Sidebar.tsx` should include the 1-based stage index (`<task_key or skill> <n>`) so repeated task keys like `execute` remain distinguishable.
- Stage row React keys in `ui/src/components/Sidebar.tsx` must include order ID + stage index. Keys derived from `{task_key, session_id, status}` collide for repeated unsessioned stages (`execute:none:pending`) and can misrender the list.
- Active/merging stage rows should have stronger visual contrast than pending rows (accent text/background + larger play icon) to make the currently running stage obvious.
- `ui/src/noodle.css` should not hard-cap `.tree-stages.open` to a fixed pixel max-height; long orders must remain fully visible (or scrollable) instead of truncating the stage list.

See also [[plans/90-interactive-sessions/phase-06-ui-sidebar-launcher]] and [[principles/experience-first]]
