Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 11: Fix Stale References Across Remaining Skills

## Goal

Update all existing skills (other than plan, which was updated in Phase 10) that reference CLI commands or paths that no longer exist.

## Stale References Found

### `.agents/skills/todo/SKILL.md`
- References `noodle todo` CLI commands throughout (`go -C noodle run . todo add`, etc.)
- These commands don't exist yet — after Phase 9 the catalog will include `plan create/done/phase-add`, but `todo` commands still route through the backlog adapter
- Should use backlog adapter scripts

### `.agents/skills/noodle/SKILL.md` + `references/config.md`
- Multiple references to `~/.noodle/config.toml` as the config path
- The project config is now `noodle.toml` at project root

### `.agents/skills/reflect/SKILL.md`
- References old role-based skills in routing examples: "update `.agents/skills/manager/SKILL.md`", "update `.agents/skills/director/SKILL.md`"
- These skills are deleted in Phase 12

### `.agents/skills/worktree/SKILL.md`
- References `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` (old binary path)
- Todo #12 tracks this

## Backlog Adapter Commands

Skills should invoke backlog operations through the configured adapter scripts in `[adapters.backlog]`, not hard-coded paths. The adapter contract defines these operations:

- `sync` — list all items (NDJSON)
- `add` — create a new item (stdin: `{"title":"...","section":"..."}`)
- `done <ID>` — mark item complete
- `edit <ID>` — update an item (stdin: `{"title":"..."}`)

## Changes

### `.agents/skills/todo/SKILL.md`
- Replace all `go -C noodle run . todo` commands with backlog adapter equivalents
- Update the commands section to show adapter invocations with stdin JSON format
- Keep the rule "never edit `brain/todos.md` directly" — route through adapters

### `.agents/skills/noodle/SKILL.md` + `references/config.md`
- Update `~/.noodle/config.toml` references to `noodle.toml` (project root)
- Update `[adapters.plans]` references to reflect that plans are now native (Phase 9)
- Clarify config location hierarchy if relevant

### `.agents/skills/reflect/SKILL.md`
- Remove references to manager, director, operator skills
- Update routing examples to reference current skill names (quality, prioritize, etc.)

### `.agents/skills/worktree/SKILL.md`
- Update `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` to `go run -C $CLAUDE_PROJECT_DIR . worktree hook`
- Closes todo #12

## Verification

- Grep for `noodle todo` across all `.agents/skills/` — zero matches
- Grep for `old_noodle` across all `.agents/skills/` — zero matches
- Grep for `~/.noodle/config.toml` across all `.agents/skills/` — zero matches (unless referring to global config)
- Grep for references to deleted skills (ceo, cto, director, manager, operator) in non-deleted skill files — zero matches
- Backlog adapter commands in updated todo skill actually work: configured `sync` command produces valid NDJSON
