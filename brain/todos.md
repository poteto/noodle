# Todos

<!-- next-id: 42 -->

## Tooling

1. [x] Noodle open-source architecture: redesign Noodle for OSS adoption — kitchen brigade naming (Chef/Sous Chef/Taster/Cook/Mise), skills as the only extension point, LLM-powered prioritization replacing deterministic scoring, adapter pattern for backlog/plans, aggressive Go code deletion, bootstrap skill. [[archived_plans/01-noodle-extensible-skill-layering/overview]]
2. [x] ~~Investigate automated post-reflect/meditate learning step~~ — superseded by the prioritize agent, which already does LLM-powered learning and queue scheduling based on session history and measured evidence.
39. [ ] Upgrade Bubble Tea to v2 using the upstream migration guide. https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md [[plans/39-charmbracelet-v2-upgrade/overview]]
40. [ ] Upgrade Lip Gloss to v2 using the upstream migration guide. https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md [[plans/39-charmbracelet-v2-upgrade/overview]]
41. [ ] Upgrade Bubbles to v2 using the upstream migration guide. https://github.com/charmbracelet/bubbles/blob/main/UPGRADE_GUIDE_V2.md [[plans/39-charmbracelet-v2-upgrade/overview]]

## Noodle Post-Plan 1

3. [ ] Bootstrap Noodle on itself — run the bootstrap skill on this repo to create `.noodle.toml`, install adapter scripts to `.noodle/adapters/`, build binary to `~/.noodle/bin/`, and verify the full toolchain end-to-end. Delete stale `.noodle/config.toml` (old pre-Plan 1 schema).
4. [ ] End-to-end smoke test — run `noodle start --once` against real `brain/todos.md`. Verify full cycle: mise gathers state, sous chef writes queue, cook spawns in worktree, taster reviews, worktree merges to main, adapter `done` marks item complete. This is the gap between "compiles" and "works."
5. [ ] Integration test for the scheduling loop — a single Go test that exercises the full loop with mock spawner (no real tmux): mise → queue → spawn → monitor → complete → taster → merge → done. Catches cross-package regressions that unit tests miss.
6. [x] ~~Fix Cook struct duplication~~ — completed. `model.Cook` now stores provider/model only under `Policy` in `model/types.go`. #cleanup
7. [ ] `go install` support — add proper module versioning so `go install github.com/poteto/noodle@latest` works without cloning the repo. Update README quick start accordingly.
8. [ ] Run-level cost budget — enforce a total-run budget cap (e.g. "stop after spending $50"). `SpawnRequest` has per-cook budget but nothing aggregates across the run. Add a `budget.max_run_cost` config field; the loop checks cumulative cost before spawning.
9. [x] ~~Autonomy dial~~ — completed. Three modes implemented: `full` (auto-merge, no human in loop), `review` (quality gate runs, auto-merge on APPROVE, default), `approve` (quality gate runs, human confirms merge via TUI). Wired through config, loop gating, runtime control commands, and TUI verdict cards. Pre-spawn gating was dropped in favor of post-completion approval.
10. [x] ~~TUI as default terminal mode~~ — completed. `noodle start` now launches the TUI in interactive terminals and keeps headless behavior in non-interactive mode.
11. [x] TUI revamp — 4B command center: left rail (agents + stats) + tabbed right pane (Feed, Queue, Brain, Config). Pastel color palette, lipgloss-inspired styling, autonomy dial (approve/review/full), quality verdicts as actionable feed cards, glamour markdown preview in Brain tab. [[archived_plans/25-tui-revamp/overview]]

## Skill Cleanup

11. [x] Remove old role-based skills — delete `.agents/skills/ceo`, `.agents/skills/cto`, `.agents/skills/director`, `.agents/skills/manager`, `.agents/skills/operator`. These are pre-kitchen-brigade roles (CEO/CTO/Director/Manager/Operator) that no longer exist in the architecture. The kitchen brigade model has Chef (human), Sous Chef, Taster, Cook. #cleanup
12. [x] Update worktree skill — `.agents/skills/worktree/SKILL.md` references `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` (old binary path). Update to `go run -C $CLAUDE_PROJECT_DIR . worktree hook` to match the repo-root module. #cleanup
13. [x] Update noodle skill — `.agents/skills/noodle/SKILL.md` references `~/.noodle/config.toml` (old config path). The new config is `.noodle.toml` at project root. Update to match the Plan 1 architecture. #cleanup
14. [ ] Evaluate interactive skill overlap with task-type skills — `.agents/skills/` has `commit`, `debugging`, `reflect`, `meditate`, `todo`, `plan` that partially overlap with task-type responsibilities. The interactive versions have different context (AskUserQuestion, human workflows), so they should be reviewed: either keep both with clear scoping, merge where possible, or delegate one to the other. The `todo` vs `backlog` naming mismatch is still the most confusing. #cleanup
15. [x] Bootstrap & onboarding — `brew install` + `noodle start` flow: Homebrew tap for macOS distribution, INSTALL.md as agent entry point (vision + install instructions), PHILOSOPHY.md (brain, self-learning, agent-pilled rationale), auto-generated noodle skill with CI snapshot guard, first-run scaffolding in `noodle start`, agent-driven skill/adapter/hook setup. [[archived_plans/15-bootstrap-onboarding/overview]]
23. [x] Task-type skill suite: create/rewrite a principle-grounded skill for each task type (prioritize, review, verify, taster, reflect, meditate, oops, debugging, debate), extract patterns from old role-based skills, delete CEO/CTO/Director/Manager/Operator. [[archived_plans/23-task-type-skill-suite/overview]]

## Bootstrap Skill Fixes

16. [x] Fix default routing model in docs and defaults — the README and `config/config.go` both default `routing.defaults.model` to `claude-sonnet-4-6`. If the intended default is `claude-opus-4-6`, update README minimal config example, `config/config.go:DefaultConfig()`, and `config/config_test.go`.
17. [x] ~~Remove `~/.noodle/skills` from default skills.paths~~ — completed/superseded. `defaultSkillPaths()` now defaults to `[".agents/skills"]` and no longer includes global `~/.noodle/skills`.
18. [ ] Fix adapter skill name assumptions — bootstrap/docs still assume `skill = "backlog"` and `skill = "plans"` without verifying those skill names resolve in the configured skill paths. Add detection or a validation warning when configured adapter skills are missing.
19. [ ] Update defaults tests and docs table after deciding #16 — if routing default model changes, update `config/config.go:DefaultConfig()`, `config/config_test.go`, and any default tables/docs that assert old values.
20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.
21. [x] Redesign fixture framework to directory-based state fixtures with metadata assertions [[archived_plans/21-fixture-directory-redesign/overview]]
22. [x] Make planning opinionated and first-class — native plan reader and brain/plans format done (Plan 23). Interactive TUI planning session was dropped; steering via control commands is sufficient.
24. [x] ~~Rewrite vision/noodle doc~~ — completed. Updated to reflect current architecture: file-based state (.noodle/ JSON/NDJSON), LLM judgment / Go mechanics split, frontmatter-based task type discovery, kitchen brigade as implemented (Prioritize/Quality/Cook/Mise), autonomy dial, adapters.
28. [ ] Rename `prioritize` skill/task type to `schedule` — better describes what it does (scheduling, not just prioritization). Rename skill directory, frontmatter, all references in loop, mise, config, tests, docs, and brain.
29. [ ] Queue item context passthrough — let the prioritization agent attach arbitrary context to a queue item. This context gets injected into the spawned agent's prompt, so the prioritizer can pass along reasoning, constraints, or instructions to any task it schedules.
30. [ ] Sprites session marked "failed" despite successful completion — `cmd.Wait()` returns a non-`ExitError` (likely connection drop after ~60s), causing `waitAndSync` to classify the session as "failed". The agent finishes work, push-back succeeds, but terminal status is wrong. Investigate what `sprites.Cmd.Wait()` actually returns on connection drop vs clean exit, and fix the status classification in `sprites_session.go:waitAndSync`. May need to inspect the raw stream for a `result` event with `subtype: "success"` as a fallback signal.
27. [ ] ~~Hot-reload skill registry via fsnotify~~ — subsumed by #38.
26. [x] ~~Context usage tracking — move beyond cost tracking to measure per-skill context footprint, per-session peak usage, compression events, and subagent context duplication. Surface to quality gate (flag sessions hitting compression), meditate (flag highest-footprint skills for investigation), prioritize (prefer lower-context approaches), and execute (inform delegation heuristics with actual context cost). Automates the manual audit-and-fix loop for skill/prompt bloat. [[archived_plans/26-context-usage-tracking/overview]]~~ — marked complete per user request.

## Loop Observability & DX

31. [ ] Structured loop logging — the loop produces zero stderr/stdout. Dispatch failures, queue validation rejections, session completions, retry decisions — all silent. Add structured stderr logging for key lifecycle events so you can tail the log and see what's happening.
32. [ ] `--project-dir` flag — `app.ProjectDir()` uses `os.Getwd()` as the only mechanism. Add a `--project-dir` flag (and/or `NOODLE_PROJECT_DIR` env var) so the binary can target a project without `cd`ing into it.
33. [ ] PID file and stale process detection — no guard against multiple noodle processes running against the same project. Write a PID file to `.noodle/noodle.pid`, check it on startup, warn or exit if another instance is alive.
34. [ ] Watch `failed.json` for changes (or add control command to reset) — failed targets are loaded at startup and cached in memory. Clearing the file while the loop runs has no effect. Either watch the file with fsnotify or expose a `clear-failed` control command.
35. [ ] Silent plan/skill discovery warnings — missing frontmatter, wrong directory structure, missing adapter scripts all fail silently. Plans just don't appear in mise.json with no indication why. `noodle debug` should surface discovery issues, and mise builder should emit warnings.
36. [ ] Default `recovery.max_retries` to at least 1 — currently defaults to 0, so any transient failure (sprites connection drop, temporary network issue) permanently kills the session with no retry.
37. [ ] Skip prioritize when queue already has items — every fresh start requires a 60-120s prioritize session before any work happens. If `queue.json` already has items, dispatch them immediately instead of bootstrapping a new prioritize cycle. Or add a `--queue` flag to pre-seed the queue.
38. [ ] Resilient skill resolution — never-fatal missing skills, built-in oops fallback, prioritize bootstrap agent, fsnotify hot-reload, queue audit. Only fatal error: cannot spawn agent at all. [[plans/38-resilient-skill-resolution/overview]]
