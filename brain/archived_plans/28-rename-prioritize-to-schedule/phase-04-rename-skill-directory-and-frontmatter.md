Back to [[archived_plans/28-rename-prioritize-to-schedule/overview]]

# Phase 4: Rename skill directory and frontmatter

## Goal

Rename the skill directory and update its frontmatter so the scheduling skill is discovered as "schedule" by the task registry. After this phase, `.agents/skills/schedule/SKILL.md` exists with `name: schedule` and the old `prioritize` directory is gone.

## Changes

### Directory rename

- `git mv .agents/skills/prioritize/ .agents/skills/schedule/`

### `.agents/skills/schedule/SKILL.md`

- Frontmatter `name: prioritize` becomes `name: schedule`
- Frontmatter `description:` update "Queue scheduler" text is fine as-is — just ensure no stale "prioritize" references remain
- Heading `# Prioritize` becomes `# Schedule`
- Body text: any remaining "prioritize" references in prose become "schedule" (check section headings, heuristic descriptions, principle references)

## Data structures

No code struct changes. The skill frontmatter `name` field drives task registry discovery — changing it from "prioritize" to "schedule" means `taskreg.NewFromSkills()` will register a `TaskType{Key: "schedule"}`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Skill prose requires reading comprehension to update references without changing meaning |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- `ls .agents/skills/schedule/SKILL.md` exists, `.agents/skills/prioritize/` does not
- Skill discovery finds `name: schedule` and registers it as a task type
- `grep -r 'prioritize' .agents/skills/schedule/` returns zero hits
