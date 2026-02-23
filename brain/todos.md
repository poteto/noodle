# Todos

<!-- next-id: 24 -->

## Tooling

1. [x] Noodle open-source architecture: redesign Noodle for OSS adoption — kitchen brigade naming (Chef/Sous Chef/Taster/Cook/Mise), skills as the only extension point, LLM-powered prioritization replacing deterministic scoring, adapter pattern for backlog/plans, aggressive Go code deletion, bootstrap skill. [[plans/01-noodle-extensible-skill-layering/overview]]
2. [ ] Investigate automated post-reflect/meditate learning step: convert measured evidence into constrained SKILL.md patch proposals with deterministic gates and shadow/canary validation before promotion.

## Noodle Post-Plan 1

3. [ ] Bootstrap Noodle on itself — run the bootstrap skill on this repo to create `noodle.toml`, install adapter scripts to `.noodle/adapters/`, build binary to `~/.noodle/bin/`, and verify the full toolchain end-to-end. Delete stale `.noodle/config.toml` (old pre-Plan 1 schema).
4. [ ] End-to-end smoke test — run `noodle start --once` against real `brain/todos.md`. Verify full cycle: mise gathers state, sous chef writes queue, cook spawns in worktree, taster reviews, worktree merges to main, adapter `done` marks item complete. This is the gap between "compiles" and "works."
5. [ ] Integration test for the scheduling loop — a single Go test that exercises the full loop with mock spawner (no real tmux): mise → queue → spawn → monitor → complete → taster → merge → done. Catches cross-package regressions that unit tests miss.
6. [ ] Fix Cook struct duplication — `model/types.go` has `Provider`/`Model` at top level AND inside `Policy`. Collapse to `Policy` only, update callers in `mise/builder.go` and `loop/loop.go`. #cleanup
7. [ ] `go install` support — add proper module versioning so `go install github.com/poteto/noodle@latest` works without cloning the repo. Update README quick start accordingly.
8. [ ] Run-level cost budget — enforce a total-run budget cap (e.g. "stop after spending $50"). `SpawnRequest` has per-cook budget but nothing aggregates across the run. Add a `budget.max_run_cost` config field; the loop checks cumulative cost before spawning.
9. [ ] Autonomy dial — the vision says "autonomy is a dial, not a switch." Add a config field (`autonomy = "full" | "review-all" | "approve-spawns"`) that gates loop behavior: `full` auto-spawns and auto-merges, `review-all` forces taster on every cook, `approve-spawns` pauses before each spawn and waits for chef confirmation via TUI.
10. [ ] TUI as default terminal mode — when `noodle start` runs in an interactive terminal, launch the TUI automatically (like `noodle start` = `noodle start` + `noodle tui` in the same process). Headless/pipe mode keeps current behavior.

## Skill Cleanup

11. [x] Remove old role-based skills — delete `.agents/skills/ceo`, `.agents/skills/cto`, `.agents/skills/director`, `.agents/skills/manager`, `.agents/skills/operator`. These are pre-kitchen-brigade roles (CEO/CTO/Director/Manager/Operator) that no longer exist in the architecture. The kitchen brigade model has Chef (human), Sous Chef, Taster, Cook. #cleanup
12. [x] Update worktree skill — `.agents/skills/worktree/SKILL.md` references `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` (old binary path). Update to `go run -C $CLAUDE_PROJECT_DIR . worktree hook` to match the repo-root module. #cleanup
13. [x] Update noodle skill — `.agents/skills/noodle/SKILL.md` references `~/.noodle/config.toml` (old config path). The new config is `noodle.toml` at project root. Update to match the Plan 1 architecture. #cleanup
14. [ ] Evaluate interactive skill overlap with Noodle defaults — `.agents/skills/` has `commit`, `debugging`, `reflect`, `meditate`, `todo`, `plan` which overlap with Noodle default skills in `skills/`. The interactive versions have different context (AskUserQuestion, human workflows) so they aren't straight duplicates, but they should be reviewed: either keep both with clear scoping, merge where possible, or have the interactive skill delegate to the Noodle default. The `todo` vs `backlog` naming mismatch is the most confusing. #cleanup
15. [ ] Fix bootstrap skill installation model — bootstrap copies Noodle default skills to `~/.noodle/skills/` (for the Noodle resolver), but doesn't make them available as Claude Code skills in the project. The README only installs `bootstrap` to `~/.claude/skills/`. Decide: should bootstrap also install/symlink Noodle skills into the project's `.agents/skills/` (or `.claude/skills/`) so they're available in interactive sessions? Update the bootstrap skill, README step 2, and the config-schema reference accordingly. The current model has two disconnected skill registries (Noodle's resolver vs Claude Code's) with no bridge.
23. [x] Task-type skill suite: create/rewrite a principle-grounded skill for each task type (prioritize, review, verify, taster, reflect, meditate, oops, debugging, debate), extract patterns from old role-based skills, delete CEO/CTO/Director/Manager/Operator. [[plans/23-task-type-skill-suite/overview]]

## Bootstrap Skill Fixes

16. [ ] Fix default routing model in config-schema — `skills/bootstrap/references/config-schema.md` and the README both default `routing.defaults.model` to `claude-sonnet-4-6`. Should be `claude-opus-4-6`. Update the config-schema reference, README minimal config example, and the Go `DefaultConfig()` in `config/config.go`.
17. [ ] Remove `~/.noodle/skills` from default skills.paths — `config-schema.md` defaults to `["skills", "~/.noodle/skills"]`. The global dir shouldn't be in the default paths — it assumes a specific installation model that isn't settled yet (see #15). Default should just be `["skills"]`.
18. [ ] Fix adapter skill name assumptions — `config-schema.md` hardcodes `skill = "backlog"` and `skill = "plans"` but the bootstrap skill doesn't verify these skills exist. The user's project may have `todo` instead of `backlog`, or `plan` instead of `plans`. Bootstrap should detect actual skill names from the resolved skill paths, or at minimum warn when the configured skill name doesn't resolve.
19. [ ] Update `DefaultConfig()` and config defaults test — after fixing #16 and #17, update `config/config.go:DefaultConfig()` and `config/config_test.go` to match the new defaults. Also update the plan overview's default config table.
20. [ ] Clarify skills.paths default for Noodle's own repo vs user projects — in the Noodle repo, `skills/` contains Noodle's default skills (backlog, sous-chef, etc.) while the developer's project skills are in `.agents/skills/` (todo, plan, etc.). For user projects, `skills/` would be the project's own skills. The bootstrap skill and config-schema need to distinguish between these two cases: bootstrapping the Noodle repo itself vs bootstrapping a user's project. The default `skills.paths` should reflect where the user's actual skills live, not assume `skills/` is always right.
21. [x] Redesign fixture framework to directory-based state fixtures with metadata assertions [[plans/21-fixture-directory-redesign/overview]]
22. [x] Make planning opinionated and first-class — native plan reader and brain/plans format done (Plan 23). Interactive TUI planning session was dropped; steering via control commands is sufficient.

