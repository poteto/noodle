---
id: 28
created: 2026-02-24
status: done
---

# Rename Prioritize To Schedule

## Context

The scheduling subsystem is called "prioritize" everywhere — constants, function names, config sections, skill directory, TUI colors, test fixtures. The name was inherited from an early prototype where the agent's only job was to sort items by priority. The agent now does full scheduling: reading mise, synthesizing follow-up tasks, routing to providers, assigning runtimes. "Schedule" describes what it actually does.

This is a wide mechanical rename touching ~82 files across Go source, config, skill frontmatter, TUI, tests, test fixtures, docs, and generated code. No behavioral changes — every instance of "prioritize" becomes "schedule" with no dual-path or backward compatibility shims.

## Scope

**In:**
- Constants: `PrioritizeTaskKey()`, `prioritizeQueueID`, `prioritizeTaskKey` (queuex)
- Functions: `isPrioritizeItem()`, `bootstrapPrioritizeQueue()`, `prioritizeQueueItem()`, `spawnPrioritize()`, `buildPrioritizePrompt()`, `reprioritizeForChefPrompt()`, `isPrioritizeTarget()`, `isPrioritizeBootstrapItem()`
- Loop callers: `loop/reconcile.go` (`prioritizePromptRegexp`, `prioritizeQueueID` reference), `loop/loop.go` (`bootstrapPrioritizeQueue()` call, comments), `loop/queue.go` (comment)
- Config: `PrioritizeConfig` type, `Prioritize` struct field, `[prioritize]` TOML section, validation error messages, metadata checks
- Skill: `.agents/skills/prioritize/` directory and `SKILL.md` frontmatter `name: prioritize`
- TUI: `Prioritize` color field (Theme structs in `tui/styles.go` and `tui/components/theme.go`), `TaskTypeColor()` case, `noPlanTaskTypes` map, `taskTypes` array, empty queue message, `steerTargets()` call, `inferTaskType()` known slice in `tui/model_snapshot.go`
- Tests: 15 test files with "prioritize" string literals (including `tui/components/components_test.go`), 4 testdata directories with "prioritize" in name
- Docs: README.md, brain/todos.md
- Generated code: `generate/skill_noodle.go` config field descriptions and noodle skill docs, `internal/schemadoc/specs.go` description strings, `dispatcher/types.go` comment

**Out:**
- Other plans that reference "prioritize" in their own phase files (plans 29, 31, 37, 38, 43) — those are documentation of what was true at the time, not live code
- Worktree copies — those are ephemeral checkouts
- Session directories under `.noodle/sessions/` — runtime artifacts, not source
- Archived plan files in `brain/archived_plans/` — historical record
- `brain/principles/` — English usage of "prioritize" (not the task type name)
- `brain/vision/slides/` — presentation artifacts, out of scope
- `dispatcher/preamble.go` "Prioritized work queue" — English adjective, not the task type name

## Constraints

- No dual-path support. Rename everything atomically — the old name must not appear in source after completion.
- The `.noodle.toml` section rename (`[prioritize]` to `[schedule]`) changes the config file contract. Existing `.noodle.toml` files in user projects will need manual updates. This is acceptable per the no-backward-compatibility rule.
- Test fixture directories must be renamed on disk (e.g. `empty-state-should-schedule-prioritize` to `empty-state-should-schedule`). Test code referencing those paths must update in the same commit.
- The skill directory rename (`.agents/skills/prioritize/` to `.agents/skills/schedule/`) must use `git mv` to preserve history.

## Applicable skills

- `go-best-practices` — Go naming conventions
- `testing` — fixture and assertion updates

## Phases

- [[archived_plans/28-rename-prioritize-to-schedule/phase-01-update-constants-and-types]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-02-rename-config-struct-and-toml-section]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-03-rename-loop-functions-and-logic]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-04-rename-skill-directory-and-frontmatter]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-05-update-tui-references]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-06-update-tests-and-test-data]]
- [[archived_plans/28-rename-prioritize-to-schedule/phase-07-update-docs-brain-and-generated-code]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

After all phases: `grep -r 'prioritize' --include='*.go' --include='*.toml' --include='*.md' .` scoped to source (excluding `.worktrees/`, `.noodle/sessions/`, `brain/archived_plans/`, `brain/plans/28-*/`, `brain/plans/29-*/`, `brain/plans/31-*/`, `brain/plans/37-*/`, `brain/plans/38-*/`, `brain/plans/43-*/`, `brain/principles/`, `brain/vision/slides/`) must return zero hits. Note: `brain/codebase/` may have stale references — verify and update if needed. `dispatcher/preamble.go` uses "Prioritized" as English (not the task type name) and is excluded.
