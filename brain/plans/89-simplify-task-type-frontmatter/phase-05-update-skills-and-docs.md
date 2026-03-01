Back to [[plans/89-simplify-task-type-frontmatter/overview]]

# Phase 5 — Update skill files and documentation

## Goal

Migrate all skill frontmatter from nested `noodle: schedule:` to top-level `schedule:` and update documentation.

## Changes

**All task type skills** (`.agents/skills/*/SKILL.md`):
- Replace `noodle:\n  schedule: "..."` with top-level `schedule: "..."`
- Remove any remaining `permissions:` or `domain_skill:` lines
- Grep for `noodle:` in all skill files to confirm none remain

**`generate/skill_noodle.go`**:
- Update example frontmatter to show top-level `schedule:`
- Remove `noodle.permissions.merge` from field table
- Remove `domain_skill` documentation
- Remove the `noodle:` wrapper from all examples
- Use `skill-creator` skill to validate template changes

**`.agents/skills/noodle/SKILL.md`** (generated):
- Regenerate from template after updating `generate/skill_noodle.go`

## Principles

- **migrate-callers-then-delete-legacy-apis** — update every skill file, leave no dual-path

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Use `skill-creator` skill when updating skill files.

## Verification

```
go generate ./generate/...
```

- Grep for `noodle:` in all skill frontmatter — should match zero files
- Grep for `permissions:` in skill frontmatter — zero
- Grep for `domain_skill:` in skill frontmatter — zero
- Confirm generated noodle skill doc matches template output
