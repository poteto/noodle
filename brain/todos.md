# Todos

<!-- next-id: 11 -->

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
