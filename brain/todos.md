# Todos

<!-- next-id: 83 -->

## Bootstrap Skill Fixes

20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

77. [ ] Add react-window (https://github.com/bvaughn/react-window) — virtualized list rendering for large datasets

## Modes

68. [ ] Unified involvement levels — replace `autonomy` and `schedule.run` with a single `mode` field that sets sensible defaults for scheduling, dispatch, and merge gating. Three levels: `auto` (Noodle runs the kitchen, respect per-skill `permissions.merge`), `supervised` (auto-schedule, auto-dispatch, human approves all merges), `manual` (user drives scheduling and dispatch, human approves all merges). Per-skill `permissions.merge` still works as a fine-grained override. Subsumes current `autonomy` (auto/approve) and `schedule.run` (after-each/after-n/manual) into one dial. [[plans/68-unified-involvement-levels/overview]]

## Backend

82. [ ] Backend V2 first-principles rewrite — redesign loop internals around a canonical order-centric state model and reducer/effect pipeline while preserving files-as-API and skills-only extensibility. Unify oversight into `mode`, standardize runtime capability contracts, align projections/files/UI streams from one state source, then cut over and delete legacy paths. [[plans/82-backend-v2-first-principles/overview]]
