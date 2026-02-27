Back to [[archive/plans/23-task-type-skill-suite/overview]]

# Phase 12: Cleanup — Stale References, Delete Old Skills

## Goal

Fix stale CLI references across all skills, delete the 5 old role-based skills, and rename sous-chef→prioritize.

## Changes

### Fix stale CLI references

#### `.agents/skills/todo/SKILL.md`
- Replace `go -C noodle run . todo` commands with backlog adapter equivalents
- After Phase 2, `plan` commands are native but `todo` commands still route through the backlog adapter

#### `.agents/skills/noodle/SKILL.md` + `references/config.md`
- Update `~/.noodle/config.toml` references to `noodle.toml` (project root)
- Update `[adapters.plans]` references to reflect plans are now native

#### `.agents/skills/reflect/SKILL.md`
- Remove references to deleted skills (manager, director, operator)
- Update routing examples to current skill names

#### `.agents/skills/worktree/SKILL.md`
- Update `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` to current binary path

### Delete old role-based skills

Remove entirely:
- `.agents/skills/ceo/`
- `.agents/skills/cto/`
- `.agents/skills/director/`
- `.agents/skills/manager/`
- `.agents/skills/operator/`

All valuable patterns extracted in Phases 4–6.

### Rename sous-chef → prioritize

Update any remaining references to `sous-chef` in config, documentation, or Go code.

### Mark todos done

- #11 (Remove old role-based skills)
- #12 (Update worktree skill)
- #14 (Evaluate interactive skill overlap)

## Verification

- `make ci` passes
- Old skill directories gone
- No references to `sous-chef` in Go code or config
- Grep for `noodle todo`, `old_noodle`, `~/.noodle/config.toml`, deleted skill names — zero matches across `.agents/skills/`
- Skill resolver finds all new skills
