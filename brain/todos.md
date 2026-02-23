# Todos

<!-- next-id: 27 -->

## Tooling

1. [x] Noodle open-source architecture: redesign Noodle for OSS adoption — kitchen brigade naming (Chef/Sous Chef/Taster/Cook/Mise), skills as the only extension point, LLM-powered prioritization replacing deterministic scoring, adapter pattern for backlog/plans, aggressive Go code deletion, bootstrap skill. [[plans/01-noodle-extensible-skill-layering/overview]]
2. [x] ~~Investigate automated post-reflect/meditate learning step~~ — superseded by the prioritize agent, which already does LLM-powered learning and queue scheduling based on session history and measured evidence.

## Noodle Post-Plan 1

3. [ ] Bootstrap Noodle on itself — run the bootstrap skill on this repo to create `.noodle.toml`, install adapter scripts to `.noodle/adapters/`, build binary to `~/.noodle/bin/`, and verify the full toolchain end-to-end. Delete stale `.noodle/config.toml` (old pre-Plan 1 schema).
4. [ ] End-to-end smoke test — run `noodle start --once` against real `brain/todos.md`. Verify full cycle: mise gathers state, sous chef writes queue, cook spawns in worktree, taster reviews, worktree merges to main, adapter `done` marks item complete. This is the gap between "compiles" and "works."
5. [ ] Integration test for the scheduling loop — a single Go test that exercises the full loop with mock spawner (no real tmux): mise → queue → spawn → monitor → complete → taster → merge → done. Catches cross-package regressions that unit tests miss.
6. [x] ~~Fix Cook struct duplication~~ — completed. `model.Cook` now stores provider/model only under `Policy` in `model/types.go`. #cleanup
7. [ ] `go install` support — add proper module versioning so `go install github.com/poteto/noodle@latest` works without cloning the repo. Update README quick start accordingly.
8. [ ] Run-level cost budget — enforce a total-run budget cap (e.g. "stop after spending $50"). `SpawnRequest` has per-cook budget but nothing aggregates across the run. Add a `budget.max_run_cost` config field; the loop checks cumulative cost before spawning.
9. [ ] Autonomy dial — the vision says "autonomy is a dial, not a switch." Add a config field (`autonomy = "full" | "review-all" | "approve-spawns"`) that gates loop behavior: `full` auto-spawns and auto-merges, `review-all` forces taster on every cook, `approve-spawns` pauses before each spawn and waits for chef confirmation via TUI.
10. [x] ~~TUI as default terminal mode~~ — completed. `noodle start` now launches the TUI in interactive terminals and keeps headless behavior in non-interactive mode.
25. [ ] TUI revamp — 4B command center: left rail (agents + stats) + tabbed right pane (Feed, Queue, Brain, Config). Pastel color palette, lipgloss-inspired styling, autonomy dial (approve/review/full), quality verdicts as actionable feed cards, glamour markdown preview in Brain tab. [[plans/25-tui-revamp/overview]]

## Skill Cleanup

11. [x] Remove old role-based skills — delete `.agents/skills/ceo`, `.agents/skills/cto`, `.agents/skills/director`, `.agents/skills/manager`, `.agents/skills/operator`. These are pre-kitchen-brigade roles (CEO/CTO/Director/Manager/Operator) that no longer exist in the architecture. The kitchen brigade model has Chef (human), Sous Chef, Taster, Cook. #cleanup
12. [x] Update worktree skill — `.agents/skills/worktree/SKILL.md` references `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` (old binary path). Update to `go run -C $CLAUDE_PROJECT_DIR . worktree hook` to match the repo-root module. #cleanup
13. [x] Update noodle skill — `.agents/skills/noodle/SKILL.md` references `~/.noodle/config.toml` (old config path). The new config is `.noodle.toml` at project root. Update to match the Plan 1 architecture. #cleanup
14. [ ] Evaluate interactive skill overlap with task-type skills — `.agents/skills/` has `commit`, `debugging`, `reflect`, `meditate`, `todo`, `plan` that partially overlap with task-type responsibilities. The interactive versions have different context (AskUserQuestion, human workflows), so they should be reviewed: either keep both with clear scoping, merge where possible, or delegate one to the other. The `todo` vs `backlog` naming mismatch is still the most confusing. #cleanup
15. [ ] Fix bootstrap skill installation model — bootstrap copies Noodle default skills to `~/.noodle/skills/` (for the Noodle resolver), but doesn't make them available as Claude Code skills in the project. The README only installs `bootstrap` to `~/.claude/skills/`. Decide: should bootstrap also install/symlink Noodle skills into the project's `.agents/skills/` (or `.claude/skills/`) so they're available in interactive sessions? Update the bootstrap skill and README bootstrap steps accordingly. The current model has two disconnected skill registries (Noodle's resolver vs Claude Code's) with no bridge.
23. [x] Task-type skill suite: create/rewrite a principle-grounded skill for each task type (prioritize, review, verify, taster, reflect, meditate, oops, debugging, debate), extract patterns from old role-based skills, delete CEO/CTO/Director/Manager/Operator. [[plans/23-task-type-skill-suite/overview]]

## Bootstrap Skill Fixes

16. [ ] Fix default routing model in docs and defaults — the README and `config/config.go` both default `routing.defaults.model` to `claude-sonnet-4-6`. If the intended default is `claude-opus-4-6`, update README minimal config example, `config/config.go:DefaultConfig()`, and `config/config_test.go`.
17. [x] ~~Remove `~/.noodle/skills` from default skills.paths~~ — completed/superseded. `defaultSkillPaths()` now defaults to `[".agents/skills"]` and no longer includes global `~/.noodle/skills`.
18. [ ] Fix adapter skill name assumptions — bootstrap/docs still assume `skill = "backlog"` and `skill = "plans"` without verifying those skill names resolve in the configured skill paths. Add detection or a validation warning when configured adapter skills are missing.
19. [ ] Update defaults tests and docs table after deciding #16 — if routing default model changes, update `config/config.go:DefaultConfig()`, `config/config_test.go`, and any default tables/docs that assert old values.
20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.
21. [x] Redesign fixture framework to directory-based state fixtures with metadata assertions [[plans/21-fixture-directory-redesign/overview]]
22. [x] Make planning opinionated and first-class — native plan reader and brain/plans format done (Plan 23). Interactive TUI planning session was dropped; steering via control commands is sufficient.
24. [ ] Rewrite vision/noodle doc — current vision doc is outdated. Update to reflect the current architecture: everything is a file (queue.json, mise.json, verdict JSON, control.ndjson), agent-pilled design (LLM prioritization, LLM quality gate, skills as system prompts), skills as the single extensibility point (frontmatter-based task type discovery, no hardcoded registries), and the kitchen brigade model as implemented.
26. [x] ~~Context usage tracking — move beyond cost tracking to measure per-skill context footprint, per-session peak usage, compression events, and subagent context duplication. Surface to quality gate (flag sessions hitting compression), meditate (flag highest-footprint skills for investigation), prioritize (prefer lower-context approaches), and execute (inform delegation heuristics with actual context cost). Automates the manual audit-and-fix loop for skill/prompt bloat. [[plans/26-context-usage-tracking/overview]]~~ — marked complete per user request.
