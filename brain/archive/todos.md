# Done

29. [x] Backlog-only scheduling — simplify to backlog-only scheduling: remove native plan reader from mise, backlog adapter is the single integration point, plans become optional context on backlog items, first-run bootstrap prompts to create adapter. Context passthrough (extra_prompt) addressed by #66 events. [[archive/plans/29-queue-item-context-passthrough/overview]]
50. [x] Reschedule button in web UI — superseded by #75 channel UI redesign
51. [x] Feed timeline — superseded by #75 channel UI redesign
54. [x] Skill registry browser — superseded by #75 channel UI redesign
55. [x] Health & stuck detection UI — superseded by #75 channel UI redesign
32. [x] `--project-dir` flag — `app.ProjectDir()` uses `os.Getwd()` as the only mechanism. Add a `--project-dir` flag (and/or `NOODLE_PROJECT_DIR` env var) so the binary can target a project without `cd`ing into it.
33. [x] Instance lock via flock — `noodle start` acquires an advisory file lock on `.noodle/noodle.lock`. Kernel releases the lock automatically on process exit (no stale lockfiles).
