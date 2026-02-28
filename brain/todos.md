# Todos

<!-- next-id: 76 -->

## Bootstrap Skill Fixes

20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.
29. [x] Backlog-only scheduling — simplify to backlog-only scheduling: remove native plan reader from mise, backlog adapter is the single integration point, plans become optional context on backlog items, first-run bootstrap prompts to create adapter. Context passthrough (extra_prompt) addressed by #66 events. [[plans/29-queue-item-context-passthrough/overview]]

## Loop Observability & DX

32. [ ] `--project-dir` flag — `app.ProjectDir()` uses `os.Getwd()` as the only mechanism. Add a `--project-dir` flag (and/or `NOODLE_PROJECT_DIR` env var) so the binary can target a project without `cd`ing into it.
33. [ ] PID file and stale process detection — no guard against multiple noodle processes running against the same project. Write a PID file to `.noodle/noodle.pid`, check it on startup, warn or exit if another instance is alive.
50. [ ] Reschedule button in web UI — add a dedicated button that spawns a reschedule agent at the top of the queue. Currently reschedule is only triggerable via steer; a visible button makes it discoverable. Reduced priority: #66 events enable reactive scheduling, but a manual button is still useful for DX.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## Web UI

51. [ ] Feed timeline — render `snapshot.feed_events` as a chronological activity stream. Cross-agent visibility: session starts, completions, failures, merges, brain writes. Data already exists server-side, just needs a UI component.
54. [ ] Skill registry browser — page or panel showing installed skills, their frontmatter (name, description, schedule, permissions), and which are registered as task types. Makes the system legible without filesystem access.
55. [ ] Health & stuck detection UI — visually differentiate `Session.health` (green/yellow/red) on agent cards. Surface `dispatch_warning` and idle-vs-stuck state. Make problems visible before opening the chat panel.

## Agent Conversations UI

75. [ ] Redesign UI from kanban to Slack/Discord-style channel layout. Schedule agent is the primary "manager" agent — user's main conversation partner — and can shut down other agents when their work is done. Each cook is a conversation channel in a sidebar grouped by status (active/idle/done). User talks to the scheduler by default but can switch to any specific agent. Backend: idle state detection via EventResult canonical event so orders advance without waiting for process exit. Auto-advance for mechanical stages (schedule, quality), explicit dismiss for execute stages.

## Modes

68. [ ] Unified involvement levels — replace `autonomy` and `schedule.run` with a single `mode` field that sets sensible defaults for scheduling, dispatch, and merge gating. Three levels: `auto` (Noodle runs the kitchen, respect per-skill `permissions.merge`), `supervised` (auto-schedule, auto-dispatch, human approves all merges), `manual` (user drives scheduling and dispatch, human approves all merges). Per-skill `permissions.merge` still works as a fine-grained override. Subsumes current `autonomy` (auto/approve) and `schedule.run` (after-each/after-n/manual) into one dial. [[plans/68-unified-involvement-levels/overview]]

